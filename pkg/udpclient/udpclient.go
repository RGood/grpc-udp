package udpclient

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"github.com/RGood/go-grpc-udp/internal/generated/packet"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type UDPClientConn struct {
	conn            net.Conn
	responseStreams *sync.Map
}

type ResponseStream struct {
	res chan *packet.Packet
}

func (conn *UDPClientConn) Invoke(ctx context.Context, method string, args interface{}, reply interface{}, opts ...grpc.CallOption) error {
	a, ok := args.(protoreflect.ProtoMessage)
	if !ok {
		return errors.New("args do not implement protoreflect")
	}
	data, err := protojson.Marshal(a)
	if err != nil {
		return errors.New("Could not marshal payload")
	}

	md := []*packet.MapEntry{}
	mdFromCtx, ok := metadata.FromOutgoingContext(ctx)
	if ok {
		for k, v := range mdFromCtx {
			md = append(md, &packet.MapEntry{
				Key:    k,
				Values: v,
			})
		}
	}

	streamID, _ := uuid.NewRandom()
	streamIDString := streamID.String()

	p := &packet.Packet{
		Id:       streamIDString,
		Method:   method,
		Metadata: md,
		Payload: &packet.Packet_Data{
			Data: data,
		},
	}

	resChan := make(chan *packet.Packet)
	defer close(resChan)
	conn.responseStreams.Store(streamIDString, &ResponseStream{
		res: resChan,
	})

	packetBytes, err := protojson.Marshal(p)
	_, err = conn.conn.Write(packetBytes)

	responseData := <-resChan
	protojson.Unmarshal(responseData.GetData(), reply.(proto.Message))

	return err
}

func (conn *UDPClientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return NewUDPClientStream(ctx, method, conn)
}

func NewClient(addr string, readBufferSize uint32) (grpc.ClientConnInterface, error) {
	networkConn, err := net.Dial("udp", addr)
	if err != nil {
		return nil, err
	}

	client := &UDPClientConn{
		conn:            networkConn,
		responseStreams: &sync.Map{},
	}

	go func() {
		buffer := make([]byte, readBufferSize)
		for {
			responseData, _ := func() ([]byte, error) {
				responseSize, err := client.conn.Read(buffer)
				if err != nil {
					return nil, err
				}

				res := make([]byte, responseSize)
				copy(res, buffer[:responseSize])
				return res, nil
			}()

			go func(c *UDPClientConn, data []byte) {
				var responsePacket packet.Packet
				protojson.Unmarshal(data, &responsePacket)
				responseStream, ok := client.responseStreams.Load(responsePacket.GetId())
				if !ok {
					fmt.Printf("couldn't find response stream for (%s)", responsePacket.Id)
					return
				}
				rs, _ := responseStream.(*ResponseStream)
				rs.res <- &responsePacket
			}(client, responseData)
		}
	}()

	return client, nil
}
