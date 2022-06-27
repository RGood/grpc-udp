package main

import (
	"context"
	"fmt"
	"time"

	"github.com/RGood/go-grpc-udp/internal/generated/example"
	"github.com/RGood/go-grpc-udp/pkg/udpclient"
)

func main() {
	conn, err := udpclient.NewClient("localhost:1234", 100000000)
	if err != nil {
		panic(err)
	}

	c := example.NewExampleClient(conn)

	//totalSyncRequestTime := time.Second * 0
	pings := 5
	/*
		wg := sync.WaitGroup{}
		for i := 0; i < pings; i++ {
			wg.Add(1)
			go func(id int) {
				fmt.Printf("Sending %d\n", id)
				start := time.Now()
				res, _ := c.Ping(context.Background(), &example.PingMessage{
					Message: strconv.Itoa(id),
				})
				end := time.Now()
				duration := end.Sub(start)
				fmt.Printf("Duration: %dμs\n", duration.Microseconds())
				totalSyncRequestTime += duration
				fmt.Printf("%d => %s\n", id, res.Message)
				wg.Done()
			}(i)
		}

		pingStream, err := c.PingStream(context.TODO())
		if err != nil {
			panic(err)
		}

		start := time.Now()
		for i := 0; i < pings; i++ {
			pingStream.Send(&example.PingMessage{
				Message: fmt.Sprintf("Foo(%d)", i),
			})
		}

		for i := 0; i < pings; i++ {
			pongMessage, _ := pingStream.Recv()
			fmt.Printf("%d: %v\n", i, pongMessage)
		}
		end := time.Now()
		totalStreamTime := end.Sub(start)

		pingStream.CloseSend()

	*/
	ss, err := c.SendStream(context.Background())
	if err != nil {
		panic(err)
	}
	for i := 0; i < pings; i++ {
		ss.Send(&example.PingMessage{
			Message: fmt.Sprintf("%d", i),
		})
	}

	time.Sleep(time.Millisecond * 10)

	pongMessage, _ := ss.CloseAndRecv()
	fmt.Printf("%s\n", pongMessage.Message)

	rs, err := c.RecvStream(context.Background(), &example.PingMessage{
		Message: "bar",
	})

	for i := 0; i < 5; i++ {
		m, _ := rs.Recv()
		fmt.Printf("%s\n", m.Message)
	}

	//wg.Wait()

	//fmt.Printf("Avg Sync Request Time: %d\nAvg Stream Request Time: %d\n", totalSyncRequestTime.Microseconds()/int64(pings), totalStreamTime.Microseconds()/int64(pings))
}
