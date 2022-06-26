package main

import (
	"context"
	"fmt"

	"github.com/RGood/go-grpc-udp/internal/generated/packet"
	"github.com/RGood/go-grpc-udp/pkg/udpclient"
)

func main() {
	conn, err := udpclient.NewClient("localhost:1234", 2147483647)
	if err != nil {
		panic(err)
	}

	c := packet.NewUDPServerClient(conn)
	for i := 0; i < 10; i++ {
		fmt.Printf("Sending %d\n", i)
		res, err := c.Send(context.Background(), &packet.Packet{
			Method:  "foo",
			Payload: []byte("Hello world!"),
		})
		fmt.Printf("%v : %v\n", res, err)
	}

}
