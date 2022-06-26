package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"github.com/RGood/go-grpc-udp/internal/generated/packet"
	"github.com/RGood/go-grpc-udp/pkg/udpserver"
)

type mockUDPServer struct {
	packet.UnimplementedUDPServerServer
}

func (mockUDPServer *mockUDPServer) Send(ctx context.Context, p *packet.Packet) (*packet.Packet, error) {
	text := string(p.Payload)
	p.Payload = []byte("Pong: " + text)
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
