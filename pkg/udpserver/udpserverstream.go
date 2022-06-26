package udpserver

import (
	"context"
	"errors"
	"net"

	"github.com/RGood/go-grpc-udp/internal/generated/packet"
	"github.com/RGood/go-grpc-udp/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type UDPServerStream struct {
	conn        net.PacketConn
	caller      net.Addr
	streamID    string
	method      string
	md          []*packet.MapEntry
	dataChannel chan []byte
}

var _ grpc.ServerStream = (*UDPServerStream)(nil)

func NewUDPServerStream(conn net.PacketConn, caller net.Addr, id, method string) *UDPServerStream {
	return &UDPServerStream{
		conn:        conn,
		caller:      caller,
		streamID:    id,
		method:      method,
		dataChannel: make(chan []byte),
	}
}

func (ss *UDPServerStream) SetHeader(md metadata.MD) error {
	ss.md = utils.MetadataToMapEntries(md)
	return nil
}

func (ss *UDPServerStream) SendHeader(md metadata.MD) error {
	return nil
}

func (ss *UDPServerStream) SetTrailer(md metadata.MD) {

}

func (ss *UDPServerStream) Context() context.Context {
	return context.Background()
}

func (ss *UDPServerStream) SendMsg(m interface{}) error {
	streamData, err := protojson.Marshal(m.(proto.Message))
	if err != nil {
		return err
	}

	p := &packet.Packet{
		Id: ss.streamID,
		Payload: &packet.Packet_StreamData{
			StreamData: streamData,
		},
	}

	packetData, err := protojson.Marshal(p)
	if err != nil {
		return err
	}

	_, err = ss.conn.WriteTo(packetData, ss.caller)

	return err
}

func (ss *UDPServerStream) RecvMsg(m interface{}) error {
	b, ok := <-ss.dataChannel
	if !ok {
		return errors.New("error reading from stream")
	}
	return protojson.Unmarshal(b, m.(proto.Message))
}
