package udpserver

import (
	"context"
	"fmt"
	"net"
	"reflect"
	"strings"

	"github.com/RGood/go-grpc-udp/internal/generated/packet"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

type UDPServerConn struct {
	conn          net.PacketConn
	bufferSize    uint32
	services      map[string]*Service
	activeStreams map[string]*UDPServerStream
}

type Service struct {
	impl    interface{}
	methods map[string]grpc.MethodDesc
	streams map[string]grpc.StreamDesc
}

func NewServer(conn net.PacketConn, bufferSize uint32) *UDPServerConn {
	return &UDPServerConn{
		conn:          conn,
		bufferSize:    bufferSize,
		services:      map[string]*Service{},
		activeStreams: map[string]*UDPServerStream{},
	}
}

func (udpServer *UDPServerConn) RegisterService(desc *grpc.ServiceDesc, impl interface{}) {
	if impl != nil {
		ht := reflect.TypeOf(desc.HandlerType).Elem()
		st := reflect.TypeOf(impl)
		if !st.Implements(ht) {
			panic(fmt.Sprintf("grpc: Server.RegisterService found the handler of type %v that does not satisfy %v", st, ht))
		}
	}

	newService := &Service{
		impl:    impl,
		methods: map[string]grpc.MethodDesc{},
		streams: map[string]grpc.StreamDesc{},
	}

	for _, m := range desc.Methods {
		newService.methods[m.MethodName] = m
	}

	for _, s := range desc.Streams {
		newService.streams[s.StreamName] = s
	}

	udpServer.services[desc.ServiceName] = newService
}

func (udpServer *UDPServerConn) sendErrorResponse(caller net.Addr, payload *packet.Packet, err error) {
	fmt.Printf("Sending error response: %s\n", err.Error())
	errorResponse := &packet.Packet{
		Id:       payload.Id,
		Method:   payload.Method,
		Metadata: payload.Metadata,
		Payload: &packet.Packet_Error{
			Error: err.Error(),
		},
	}

	responsePacketBytes, err := protojson.Marshal(errorResponse)

	_, err = udpServer.conn.WriteTo(responsePacketBytes, caller)
	if err != nil {
		fmt.Printf("Error sending bytes: %v\n", err)
	}
}

func (udpServer *UDPServerConn) handleOpenStream(caller net.Addr, payload *packet.Packet) {
	md := map[string][]string{}
	for _, entry := range payload.Metadata {
		md[entry.Key] = entry.Values
	}

	ctx := metadata.NewIncomingContext(context.Background(), md)

	newStream := NewUDPServerStream(udpServer.conn, caller, payload.Id, payload.Method)

	headerMD, _ := metadata.FromIncomingContext(ctx)
	newStream.SetHeader(headerMD)

	udpServer.activeStreams[payload.Id] = newStream

	routingSegments := strings.Split(payload.Method, "/")
	serviceName := routingSegments[1]
	streamName := routingSegments[2]
	service, _ := udpServer.services[serviceName]
	streamDesc := service.streams[streamName]

	go streamDesc.Handler(service.impl, newStream)

	responsePacketBytes, err := protojson.Marshal(payload)

	_, err = udpServer.conn.WriteTo(responsePacketBytes, caller)
	if err != nil {
		fmt.Printf("Error sending bytes: %v\n", err)
	}
}

func (udpServer *UDPServerConn) handleCloseStream(caller net.Addr, payload *packet.Packet) {
	closingStream, ok := udpServer.activeStreams[payload.Id]
	if ok {
		delete(udpServer.activeStreams, payload.Id)
		close(closingStream.dataChannel)
	}
}

func (udpServer *UDPServerConn) handleMessage(caller net.Addr, payload *packet.Packet) {

	md := map[string][]string{}
	for _, entry := range payload.Metadata {
		md[entry.Key] = entry.Values
	}

	ctx := metadata.NewIncomingContext(context.Background(), md)

	routingSegments := strings.Split(payload.Method, "/")
	serviceName := routingSegments[1]
	methodName := routingSegments[2]
	service, _ := udpServer.services[serviceName]
	method := service.methods[methodName]
	res, err := method.Handler(service.impl, ctx, func(in interface{}) error {
		return protojson.Unmarshal(payload.GetData(), in.(proto.Message))
	}, func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	})
	if err != nil {
		udpServer.sendErrorResponse(caller, payload, err)
		return
	}

	responseBytes, err := protojson.Marshal(res.(proto.Message))
	if err != nil {
		udpServer.sendErrorResponse(caller, payload, err)
		return
	}

	responsePacket := &packet.Packet{
		Id:     payload.Id,
		Method: payload.Method,
		Payload: &packet.Packet_Data{
			Data: responseBytes,
		},
	}

	responsePacketBytes, err := protojson.Marshal(responsePacket)

	_, err = udpServer.conn.WriteTo(responsePacketBytes, caller)
	if err != nil {
		fmt.Printf("Error sending bytes: %v\n", err)
	}
}

func (udpServer *UDPServerConn) handleStreamMessage(caller net.Addr, payload *packet.Packet) {
	serverStream, ok := udpServer.activeStreams[payload.GetId()]
	if ok {
		go func() { serverStream.dataChannel <- payload.GetStreamData() }()
	}
}

func (udpServer *UDPServerConn) handleIncoming(caller net.Addr, data []byte) {
	var payload packet.Packet
	protojson.Unmarshal(data, &payload)

	//protoreflect.Message.WhichOneof()
	descriptor := payload.ProtoReflect().WhichOneof(payload.ProtoReflect().Descriptor().Oneofs().ByName("payload"))

	packetType := descriptor.JSONName()
	switch packetType {
	case "data":
		udpServer.handleMessage(caller, &payload)
	case "openStream":
		udpServer.handleOpenStream(caller, &payload)
	case "closeStream":
		udpServer.handleCloseStream(caller, &payload)
	case "streamData":
		udpServer.handleStreamMessage(caller, &payload)
	}
}

func (udpServer *UDPServerConn) Listen() {
	for {
		buffer := make([]byte, udpServer.bufferSize)
		bytesRead, caller, err := udpServer.conn.ReadFrom(buffer)
		if err != nil {
			fmt.Printf("Error (%s) => %s", caller.String(), err.Error())
		} else {
			go udpServer.handleIncoming(caller, buffer[:bytesRead])
		}
	}
}
