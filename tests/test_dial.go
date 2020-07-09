package main

import (
	"flag"
	"fmt"
	"log"
	"net"
	"os"
	"smarteragents/ellimango"
	"time"
)

var (
	network = flag.String("network", "", "Network.") // tcp, udp, unix, ip, unixgram
	address = flag.String("address", "", "Address.") // host with port
	timeout = flag.Int("timeout", 0, "Connect timeout.")
)

func main() {
	flag.Parse()
	if *network == "" {
		log.Println("Please specify network")
		os.Exit(1)
	}
	if *address == "" {
		log.Println("Please specify address")
		os.Exit(1)
	}
	if *timeout == 0 {
		log.Println("Please specify timeout")
		os.Exit(1)
	}
	helper := ellimango.Helper{Env: "local"}

	conn, err := net.DialTimeout(*network, *address, time.Duration(*timeout)*time.Second)
	// handle error
	if err != nil {
		log.Println("error", err)
		helper.SendEmail("Rafik Majidov", "rmajidov@reol.com", fmt.Sprintf("DialTimeout error %v", err), "DialTimeout error")
	} else {
		log.Println("conn", conn)
		err = conn.Close()
		log.Println("conn close error", err)
	}
}
