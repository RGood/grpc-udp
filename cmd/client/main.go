package main

import (
	"context"
	"fmt"
	"strconv"
	"sync"

	"github.com/RGood/go-grpc-udp/internal/generated/example"
	"github.com/RGood/go-grpc-udp/pkg/udpclient"
)

func main() {
	conn, err := udpclient.NewClient("localhost:1234", 100000000)
	if err != nil {
		panic(err)
	}

	c := example.NewExampleClient(conn)

	wg := sync.WaitGroup{}
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {

			fmt.Printf("Sending %d\n", id)
			res, _ := c.Ping(context.Background(), &example.PingMessage{
				Message: strconv.Itoa(id),
			})
			fmt.Printf("%d => %s\n", id, res.Message)
			wg.Done()
		}(i)
	}

	wg.Wait()

	pingStream, err := c.PingStream(context.TODO())
	if err != nil {
		panic(err)
	}

	println("Stream initialized")

	pingStream.Send(&example.PingMessage{
		Message: "Foo",
	})

	pongMessage, _ := pingStream.Recv()
	fmt.Printf("%v\n", pongMessage)

	pingStream.CloseSend()

}
