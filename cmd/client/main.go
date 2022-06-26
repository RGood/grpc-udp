package main

import (
	"context"
	"fmt"
	"sync"

	"github.com/RGood/go-grpc-udp/internal/generated/packet"
	"github.com/RGood/go-grpc-udp/pkg/udpclient"
)

func main() {
	conn, err := udpclient.NewClient("localhost:1234", 10000)
	if err != nil {
		panic(err)
	}

	c := packet.NewUDPServerClient(conn)

	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {

			fmt.Printf("Sending %d\n", id)
			res, _ := c.Send(context.Background(), &packet.Packet{
				Method:  "asdf",
				Payload: []byte(fmt.Sprintf("%d", id)),
			})
			fmt.Printf("%s : %d\n", res.Payload, id)
			wg.Done()
		}(i)
	}

	wg.Wait()

}
