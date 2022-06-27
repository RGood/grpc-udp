package main

import (
	"context"
	"fmt"
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

func (mockExample *mockExample) SendStream(stream example.Example_SendStreamServer) error {
	resText := ""
	for ping, err := stream.Recv(); err == nil; ping, err = stream.Recv() {
		resText += ping.Message
	}

	stream.SendAndClose(&example.PongMessage{
		Message: resText,
	})

	return nil
}

func (mockExample *mockExample) RecvStream(ping *example.PingMessage, stream example.Example_RecvStreamServer) error {
	fmt.Printf("In RecvStream server\n")
	msg := ping.Message
	for i := 0; i < 5; i++ {
		stream.Send(&example.PongMessage{
			Message: fmt.Sprintf("%s(%d)", msg, i),
		})
	}

	return nil
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
