package main

import (
	"fmt"
	"github.com/BGrewell/ethercomms/client"
	"net"
	"time"
)

func main() {

	etherClient := client.EtherClient{}
	recv := etherClient.ReceiveChannel
	send := etherClient.SendChannel
	etherClient.Initialize("lo")
	for {
		select {
			case rmf := <- recv:
				fmt.Printf("Received: %s", string(rmf.Payload))
			default:
				time.Sleep(5*time.Second)
				smf := client.MessageFrame{
					Source:      net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
					Destination: net.HardwareAddr{0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
					EtherType:   0xcccc,
					Payload:     []byte("this is a test"),
				}
				fmt.Printf("sending\n")
				send <- smf
		}
	}
}
