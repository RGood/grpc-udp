package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/RGood/go-grpc-udp/internal/generated/packet"
	"github.com/RGood/go-grpc-udp/pkg/udpserver"
)

func oldMain() {
	//addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("localhost:%d", 6969))
	//net.Listen()
	lis, err := net.ListenPacket("udp", "localhost:1234")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	buffer := make([]byte, 2147483647)
	for {
		payloadSize, remoteAddr, err := lis.ReadFrom(buffer)
		if err != nil {
			fmt.Printf("Err reading from UDP: %v\n", err)
		}
		fmt.Printf("%s (%d) => %v\n", remoteAddr.String(), payloadSize, buffer[:payloadSize])
	}
}

type mockUDPServer struct {
	packet.UnimplementedUDPServerServer
}

func reverse(str string) (result string) {
	for _, v := range str {
		result = string(v) + result
	}
	return
}

func (mockUDPServer *mockUDPServer) Send(ctx context.Context, p *packet.Packet) (*packet.Packet, error) {
	text := string(p.Payload)
	p.Payload = []byte(reverse(text))
	fmt.Printf("Got Packet: %v\n", p)
	return p, nil
}

func main() {
	lis, err := net.ListenPacket("udp", "localhost:1234")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	srv := udpserver.NewServer(lis, 100000)
	srv.RegisterService(&packet.UDPServer_ServiceDesc, &mockUDPServer{})
	srv.Listen()
}
