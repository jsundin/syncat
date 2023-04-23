package main

import (
	"encoding/binary"
	"fmt"
	"net"

	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"github.com/sirupsen/logrus"
)

var ttl = uint8(127)
var window_size = uint16(65535)

func XOR(data, key []byte) []byte {
	out := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		out[i] = data[i] ^ key[i%len(key)]
	}
	return out
}

func ParseIP(ip string) net.IP {
	net_ip := net.ParseIP(ip)
	if net_ip == nil {
		panic(fmt.Errorf("cannot parse ip: %v", ip))
	}
	return net_ip
}

func GetIPLayer(dst, src net.IP) *layers.IPv4 {
	ip := &layers.IPv4{
		SrcIP:    src,
		DstIP:    dst,
		Protocol: layers.IPProtocolIPv4,
		TTL:      ttl,
	}
	return ip
}

func GetTCPLayer(dst, src int, seqb []byte, key []byte) *layers.TCP {
	if len(seqb) != 4 {
		panic(fmt.Errorf("expected 4 bytes seq, got %d", len(seqb)))
	}
	eseqb := XOR(seqb, key)
	seq := binary.LittleEndian.Uint32(eseqb)
	tcp := &layers.TCP{
		SrcPort: layers.TCPPort(src),
		DstPort: layers.TCPPort(dst),
		Seq:     seq,
		Window:  window_size,
	}
	return tcp
}

func GetTCPIPLayers(dst *net.TCPAddr, src *net.TCPAddr, seqb []byte, key []byte) (*layers.TCP, *layers.IPv4) {
	ip := GetIPLayer(dst.IP, src.IP)
	tcp := GetTCPLayer(dst.Port, src.Port, seqb, key)
	tcp.SetNetworkLayerForChecksum(ip)
	return tcp, ip
}

func SendRaw(packetConn net.PacketConn, tcp *layers.TCP, ip *layers.IPv4) {
	buf := gopacket.NewSerializeBuffer()
	opts := gopacket.SerializeOptions{ComputeChecksums: true, FixLengths: true}
	if err := gopacket.SerializeLayers(buf, opts, tcp); err != nil {
		panic(err)
	}
	logrus.Debugf("sending %v:%d -> %v:%d: 0x%x (%d) [syn=%v,ack=%v,rst=%v]", ip.SrcIP, tcp.SrcPort, ip.DstIP, tcp.DstPort, tcp.Seq, tcp.Seq, tcp.SYN, tcp.ACK, tcp.RST)
	if _, err := packetConn.WriteTo(buf.Bytes(), &net.IPAddr{IP: ip.DstIP}); err != nil {
		panic(err)
	}
}
