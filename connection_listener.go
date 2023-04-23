package main

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/sirupsen/logrus"
)

const ReadTimeout = 1 * time.Second

type Packet struct {
	seq uint32
	src *net.TCPAddr
}

type connectionListener struct {
	conn                  *Connection
	packetQueue           []Packet
	queueLock             *sync.Mutex
	shutdownCommandAtomic uint32
	shudownComplete       *sync.WaitGroup
}

type TcpFilter func(*layers.TCP) bool

func (c *Connection) StartListener(tcpFilter TcpFilter) *connectionListener {
	listener := &connectionListener{
		packetQueue:           make([]Packet, 0),
		queueLock:             &sync.Mutex{},
		shutdownCommandAtomic: 0,
		shudownComplete:       &sync.WaitGroup{},
		conn:                  c,
	}
	listener.shudownComplete.Add(1)

	go listener.listenerLoop(tcpFilter)

	return listener
}

func (cl *connectionListener) listenerLoop(tcpFilter TcpFilter) {
	buf := make([]byte, 128)
	for atomic.LoadUint32(&cl.shutdownCommandAtomic) == 0 {
		cl.conn.packetConn.SetReadDeadline(time.Now().Add(ReadTimeout))
		n, src_addr, err := cl.conn.packetConn.ReadFrom(buf)

		if err != nil {
			if err, ok := err.(net.Error); ok && err.Timeout() {
				continue
			}
			panic(err)
		}

		packet := gopacket.NewPacket(buf[:n], layers.LayerTypeTCP, gopacket.Default)
		tcpLayer := packet.Layer(layers.LayerTypeTCP)
		if tcpLayer == nil {
			continue
		}
		tcp, _ := tcpLayer.(*layers.TCP)
		if !tcpFilter(tcp) {
			continue
		}

		src_ip := src_addr.(*net.IPAddr).IP
		new_packet := Packet{
			seq: tcp.Seq,
			src: &net.TCPAddr{IP: src_ip, Port: int(tcp.SrcPort)},
		}
		logrus.Debugf("received data from %v: 0x%x (%d) [syn=%v,ack=%v,rst=%v]", new_packet.src, new_packet.seq, new_packet.seq, tcp.SYN, tcp.ACK, tcp.RST)

		cl.queueLock.Lock()
		cl.packetQueue = append(cl.packetQueue, new_packet)
		cl.queueLock.Unlock()
	}
	cl.shudownComplete.Done()
}

func (cl *connectionListener) Close() {
	atomic.StoreUint32(&cl.shutdownCommandAtomic, 1)
	cl.shudownComplete.Wait()
}

func (cl *connectionListener) Poll() (Packet, int) {
	cl.queueLock.Lock()
	defer cl.queueLock.Unlock()

	if len(cl.packetQueue) == 0 {
		return Packet{}, 0
	}
	p := cl.packetQueue[0]
	n := len(cl.packetQueue)
	cl.packetQueue = cl.packetQueue[1:]
	return p, n
}
