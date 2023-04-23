package main

import (
	"encoding/hex"
	"flag"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/sirupsen/logrus"
)

const sendDelay = 100 * time.Millisecond

func main() {
	var logLevel string
	var iface string
	var local_ip_addr string
	var key_hex string

	flag.StringVar(&logLevel, "level", "info", "")
	flag.StringVar(&iface, "if", "", "")
	flag.StringVar(&local_ip_addr, "ip", "", "")
	flag.StringVar(&key_hex, "key", "00000000", "")
	flag.Parse()

	if lvl, err := logrus.ParseLevel(logLevel); err != nil {
		panic(err)
	} else {
		logrus.SetLevel(lvl)
	}

	key, err := hex.DecodeString(key_hex)
	if err != nil {
		panic(err)
	}

	var localIP net.IP
	if iface != "" || local_ip_addr == "" {
		localIP = find_ip_by_iface(iface, local_ip_addr)
	} else {
		localIP = ParseIP(local_ip_addr)
	}

	args := flag.Args()
	if len(args) < 2 {
		usage()
	}

	if args[0] == "server" {
		if len(args) != 3 {
			usage()
		}
		listenPort, err := strconv.Atoi(args[1])
		if err != nil {
			panic(err)
		}
		cmd := args[2]
		serverMain(localIP.String(), listenPort, cmd, key)
	} else if args[0] == "client" {
		if len(args) != 2 {
			usage()
		}
		r := strings.Split(args[1], ":")
		if len(r) != 2 {
			usage()
		}
		serverIP := ParseIP(r[0])
		serverPort, err := strconv.Atoi(r[1])
		if err != nil {
			panic(err)
		}
		clientMain(localIP.String(), 0, serverIP.String(), serverPort, key)
	} else {
		usage()
	}
}

func usage() {
	fmt.Println("usage:")
	fmt.Println("syncat [flags] server 8000 dmesg")
	fmt.Println("syncat [flags] client 1.2.3.4:8000")
	os.Exit(1)
}

func find_ip_by_iface(if_filter string, ip_filter string) net.IP {
	ifaces, err := net.Interfaces()
	if err != nil {
		panic(err)
	}

	var if_ip net.IP
	if ip_filter != "" {
		if_ip = ParseIP(ip_filter)
	}

	found := []net.IP{}
	for _, iface := range ifaces {
		if if_filter == "" || if_filter == iface.Name {
			if_addrs, err := iface.Addrs()
			if err != nil {
				panic(err)
			}
			for _, if_addr := range if_addrs {
				ip := if_addr.(*net.IPNet).IP
				if if_ip == nil || if_ip.Equal(ip) {
					found = append(found, ip)
				}
			}
		}
	}

	if len(found) != 1 {
		panic(fmt.Errorf("could not find any matching ips"))
	}
	return found[0]
}
