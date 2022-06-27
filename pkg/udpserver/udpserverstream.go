package udpserver

import (
	"context"
	"errors"
	"net"
	"sync"

	"github.com/RGood/go-grpc-udp/internal/generated/packet"
	"github.com/RGood/go-grpc-udp/pkg/utils"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type UDPServerStream struct {
	conn         net.PacketConn
	caller       net.Addr
	streamID     string
	method       string
	md           []*packet.MapEntry
	dataChannel  chan []byte
	done         chan bool
	pendingInput sync.WaitGroup
}

var _ grpc.ServerStream = (*UDPServerStream)(nil)

func NewUDPServerStream(conn net.PacketConn, caller net.Addr, id, method string) *UDPServerStream {
	return &UDPServerStream{
		conn:         conn,
		caller:       caller,
		streamID:     id,
		method:       method,
		dataChannel:  make(chan []byte),
		done:         make(chan bool),
		pendingInput: sync.WaitGroup{},
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
	select {
	case b := <-ss.dataChannel:
		defer ss.pendingInput.Done()
		return protojson.Unmarshal(b, m.(proto.Message))
	case <-ss.done:
		return errors.New("stream closed")
	}

}
