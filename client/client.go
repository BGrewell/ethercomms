package client

import (
	"errors"
	"fmt"
	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/raw"
	"log"
	"net"
	"time"
)

var (
	etherType uint16 = 0x080D
)

type VLAN struct {
	Priority int
	DropEligible bool
	ID uint16
}

type MessageFrame struct {
	Source net.HardwareAddr
	Destination net.HardwareAddr
	Addr net.Addr
	EtherType uint16
	ServiceVLAN *VLAN
	VLAN *VLAN
	Payload []byte
}

type EtherClient struct {
	connection *raw.Conn
	iface *net.Interface
	ReceiveChannel chan MessageFrame
	SendChannel chan MessageFrame
	TerminateReceiver chan struct{}
	TerminateSender chan struct{}
}

func (ec *EtherClient) Initialize(networkInterface string) (err error) {
	ec.TerminateReceiver = make(chan struct{})
	ec.TerminateSender = make(chan struct{})
	ec.ReceiveChannel = make(chan MessageFrame)
	ec.SendChannel = make(chan MessageFrame)
	ec.iface, err = net.InterfaceByName(networkInterface)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to get interface: %v", err))
	}
	ec.connection, err = raw.ListenPacket(ec.iface, etherType, nil)
	if err != nil {
		return errors.New(fmt.Sprintf("failed to open socket: %v", err))
	}
	ec.startReceiver()
	ec.startSender()
	return nil
}

func (ec *EtherClient) startReceiver() {

	go func() {
		for {
			select {
			case <-ec.TerminateReceiver:
				log.Println("terminating receive thread")
				return
			default:
				var buffer = make([]byte, ec.iface.MTU)
				var frame ethernet.Frame
				ec.connection.SetReadDeadline(time.Now().Add(1 * time.Second))
				read, addr, err := ec.connection.ReadFrom(buffer)
				if err != nil {
					log.Printf("failed to receive message: %v", err)
					continue
				}
				if err := (&frame).UnmarshalBinary(buffer); err != nil {
					log.Printf("failed to unmarshal ethernet frame: %v", err)
				}
				mf := MessageFrame{
					Source:      frame.Source,
					Destination: frame.Destination,
					Addr:        addr,
					EtherType:   uint16(frame.EtherType),
					VLAN: &VLAN{
						Priority:     int(frame.VLAN.Priority),
						DropEligible: frame.VLAN.DropEligible,
						ID:           frame.VLAN.ID,
					},
					ServiceVLAN: &VLAN{
						Priority:     int(frame.ServiceVLAN.Priority),
						DropEligible: frame.ServiceVLAN.DropEligible,
						ID:           frame.ServiceVLAN.ID,
					},
					Payload: frame.Payload[:read],
				}
				ec.ReceiveChannel <- mf
				log.Printf("Processed message frame: %v", mf)
			}
		}
	}()
}

func (ec *EtherClient) startSender() {

	go func() {
		for {
			select {
			case <-ec.TerminateSender:
				log.Println("terminating receive thread")
				return
			default:
				mf := <- ec.SendChannel
				f := new(ethernet.Frame)
				f.Source = mf.Source
				f.Destination = mf.Destination
				f.EtherType = ethernet.EtherType(mf.EtherType)
				f.Payload = mf.Payload
				buffer, err := f.MarshalBinary()
				if err != nil {
					log.Printf("failed to marshal ethernet frame: %v", err)
				}
				addr := &raw.Addr{
					HardwareAddr: mf.Source,
				}
				if _, err := ec.connection.WriteTo(buffer, addr); err != nil {
					log.Fatalf("failed to write frame: %v", err)
				}
				fmt.Println("Wrote frame.")
			}

		}
	}()
}
