package udpclient

import (
	"context"
	"errors"
	"net"

	"github.com/RGood/go-grpc-udp/internal/generated/packet"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type UDPClientConn struct {
	conn           net.Conn
	readBufferSize uint32
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

	p := &packet.Packet{
		Method:   method,
		Metadata: md,
		Payload:  data,
	}

	packetBytes, err := protojson.Marshal(p)
	_, err = conn.conn.Write(packetBytes)

	responseData, err := func() ([]byte, error) {
		buffer := make([]byte, conn.readBufferSize)
		responseSize, err := conn.conn.Read(buffer)
		if err != nil {
			return nil, err
		}

		return buffer[:responseSize], nil
	}()

	err = protojson.Unmarshal(responseData, reply.(proto.Message))
	return err
}

func (conn *UDPClientConn) NewStream(ctx context.Context, desc *grpc.StreamDesc, method string, opts ...grpc.CallOption) (grpc.ClientStream, error) {
	return nil, nil
}

func NewClient(addr string, readBufferSize uint32) (grpc.ClientConnInterface, error) {
	networkConn, err := net.Dial("udp", addr)
	if err != nil {
		return nil, err
	}

	return &UDPClientConn{
		conn:           networkConn,
		readBufferSize: readBufferSize,
	}, nil
}
