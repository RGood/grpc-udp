package main

import (
	"context"
	"log"
	"net"

	"github.com/RGood/go-grpc-udp/internal/generated/example"
	"github.com/RGood/go-grpc-udp/pkg/udpserver"
)

type mockExample struct {
	example.UnimplementedExampleServer
}

func (mockExample *mockExample) Ping(ctx context.Context, p *example.PingMessage) (*example.PongMessage, error) {
	text := p.Message
	return &example.PongMessage{
		Message: "Pong: " + text,
	}, nil
}

func (mockExample *mockExample) PingStream(stream example.Example_PingStreamServer) error {
	for {
		ping, err := stream.Recv()
		if err != nil {
			return err
		}

		pong := &example.PongMessage{
			Message: "Ponging " + ping.Message,
		}

		stream.Send(pong)
	}
}

func main() {
	lis, err := net.ListenPacket("udp", "localhost:1234")
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	srv := udpserver.NewServer(lis, 100000000)
	srv.RegisterService(&example.Example_ServiceDesc, &mockExample{})
	srv.Listen()
}
