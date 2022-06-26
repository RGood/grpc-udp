package udpclient

import (
	"context"
	"errors"

	"github.com/RGood/go-grpc-udp/internal/generated/packet"
	"github.com/RGood/go-grpc-udp/pkg/utils"
	"github.com/google/uuid"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/reflect/protoreflect"
)

type UDPClientStream struct {
	streamContext context.Context
	client        *UDPClientConn
	streamID      string
	method        string
	rs            *ResponseStream
}

var _ grpc.ClientStream = (*UDPClientStream)(nil)

func NewUDPClientStream(ctx context.Context, method string, client *UDPClientConn) (*UDPClientStream, error) {
	rs := &ResponseStream{
		res: make(chan *packet.Packet),
	}

	streamID := uuid.New().String()

	client.responseStreams.Store(streamID, rs)

	md := utils.ContextToMetadata(ctx)
	packetBytes, _ := protojson.Marshal(&packet.Packet{
		Id:       streamID,
		Method:   method,
		Metadata: md,
		Payload: &packet.Packet_OpenStream{
			OpenStream: true,
		},
	})
	client.conn.Write(packetBytes)
	streamInitResponse := <-rs.res
	err, ok := streamInitResponse.Payload.(*packet.Packet_Error)
	if ok {
		return nil, errors.New(err.Error)
	}

	return &UDPClientStream{
		streamContext: ctx,
		client:        client,
		streamID:      streamID,
		method:        method,
		rs:            rs,
	}, nil
}

func (cs *UDPClientStream) Header() (metadata.MD, error) {
	md, _ := metadata.FromOutgoingContext(cs.streamContext)
	return md, nil
}

func (cs *UDPClientStream) Trailer() metadata.MD {
	md, _ := metadata.FromOutgoingContext(cs.streamContext)
	return md
}

func (cs *UDPClientStream) CloseSend() error {
	cs.client.responseStreams.Delete(cs.streamID)
	p := &packet.Packet{
		Id:       cs.streamID,
		Method:   cs.method,
		Metadata: utils.ContextToMetadata(cs.streamContext),
		Payload: &packet.Packet_CloseStream{
			CloseStream: true,
		},
	}

	packetBytes, err := protojson.Marshal(p)
	_, err = cs.client.conn.Write(packetBytes)

	return err
}

func (cs *UDPClientStream) Context() context.Context {
	return cs.streamContext
}

func (cs *UDPClientStream) SendMsg(m interface{}) error {
	a, ok := m.(protoreflect.ProtoMessage)
	if !ok {
		return errors.New("args do not implement protoreflect")
	}
	data, err := protojson.Marshal(a)
	if err != nil {
		return errors.New("Could not marshal payload")
	}

	md := []*packet.MapEntry{}
	mdFromCtx, ok := metadata.FromOutgoingContext(cs.streamContext)
	if ok {
		for k, v := range mdFromCtx {
			md = append(md, &packet.MapEntry{
				Key:    k,
				Values: v,
			})
		}
	}

	p := &packet.Packet{
		Id:       cs.streamID,
		Method:   cs.method,
		Metadata: md,
		Payload: &packet.Packet_StreamData{
			StreamData: data,
		},
	}

	packetBytes, err := protojson.Marshal(p)
	_, err = cs.client.conn.Write(packetBytes)

	return err
}

func (cs *UDPClientStream) RecvMsg(m interface{}) error {
	responseData := <-cs.rs.res
	return protojson.Unmarshal(responseData.GetStreamData(), m.(proto.Message))
}
