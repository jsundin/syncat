package main

import (
	"crypto/rand"
	"encoding/binary"
	mrand "math/rand"
	"net"
	"time"

	"github.com/google/gopacket/layers"
)

const readDelay = 100 * time.Millisecond

type Connection struct {
	packetConn  net.PacketConn
	src         *net.TCPAddr
	dst         *net.TCPAddr
	tcpListener *net.TCPListener
	listener    *connectionListener
	key         []byte
}

func NewServerConnection(listen_addr *net.TCPAddr, key []byte) *Connection {
	tcpListener, err := net.ListenTCP("tcp", listen_addr)
	if err != nil {
		panic(err)
	}

	packetConn, err := net.ListenPacket("ip4:tcp", listen_addr.IP.String())
	if err != nil {
		panic(err)
	}

	conn := &Connection{
		packetConn:  packetConn,
		src:         listen_addr,
		key:         key,
		tcpListener: tcpListener,
	}

	tcpFilter := func(tcp *layers.TCP) bool {
		if tcp.ACK || tcp.RST || !tcp.SYN {
			return false
		}
		if listen_addr.Port != int(tcp.DstPort) {
			return false
		}
		return true
	}

	conn.listener = conn.StartListener(tcpFilter)
	return conn
}

func NewClientConnection(dst, src *net.TCPAddr, key []byte) *Connection {
	packetConn, err := net.ListenPacket("ip4:tcp", "0.0.0.0")
	if err != nil {
		panic(err)
	}
	if src.Port <= 0 {
		src.Port = 40000 + (mrand.New(mrand.NewSource(time.Now().UnixMilli())).Int() % 20000)
	}

	conn := &Connection{
		packetConn: packetConn,
		src:        src,
		dst:        dst,
		key:        key,
	}

	tcpFilter := func(tcp *layers.TCP) bool {
		if len(tcp.Options) > 0 {
			return false
		}
		if !tcp.SYN || !tcp.ACK || tcp.RST {
			return false
		}
		return true
	}

	conn.listener = conn.StartListener(tcpFilter)
	return conn
}

func (c *Connection) Close() {
	c.listener.Close()
	if c.tcpListener != nil {
		c.tcpListener.Close()
	}
	c.packetConn.Close()
}

func (c *Connection) SendSYNACK(dst *net.TCPAddr, syn_seq_for_ack uint32, cmd byte, data []byte) {
	seqb := append([]byte{byte(cmd<<4) | byte(len(data))}, data...)
	if len(seqb) < 4 {
		pad := make([]byte, 4-len(seqb))
		rand.Read(pad)
		seqb = append(seqb, pad...)
	}
	tcp, ip := GetTCPIPLayers(dst, c.src, seqb, c.key)
	tcp.SYN = true
	tcp.ACK = true
	tcp.Ack = syn_seq_for_ack + 1
	SendRaw(c.packetConn, tcp, ip)
}

func (c *Connection) SendSYN(cmd byte, data []byte) {
	seqb := append([]byte{byte(cmd<<4) | byte(len(data))}, data...)
	if len(seqb) < 4 {
		pad := make([]byte, 4-len(seqb))
		rand.Read(pad)
		seqb = append(seqb, pad...)
	}
	tcp, ip := GetTCPIPLayers(c.dst, c.src, seqb, c.key)
	tcp.SYN = true
	tcp.Options = []layers.TCPOption{
		{
			OptionType:   layers.TCPOptionKindMSS,
			OptionLength: 4,
			OptionData:   []byte{0x05, 0xb4}, // 1460
		},
		{
			OptionType: layers.TCPOptionKindNop,
		},
		{
			OptionType:   layers.TCPOptionKindWindowScale,
			OptionLength: 3,
			OptionData:   []byte{8},
		},
		{
			OptionType: layers.TCPOptionKindNop,
		},
		{
			OptionType: layers.TCPOptionKindNop,
		},
		{
			OptionType:   layers.TCPOptionKindSACKPermitted,
			OptionLength: 2,
		},
	}

	SendRaw(c.packetConn, tcp, ip)
}

func (c *Connection) Read() (byte, []byte, *net.TCPAddr, uint32) {
	for {
		p, n := c.listener.Poll()
		if n == 0 {
			time.Sleep(readDelay)
			continue
		}
		seqb := make([]byte, 4)
		binary.LittleEndian.PutUint32(seqb, p.seq)
		data := XOR(seqb, c.key)
		cmd := data[0] >> 4
		arglen := data[0] & 0xf
		return cmd, data[1 : 1+arglen], p.src, p.seq
	}
}
