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
	conn       net.PacketConn
	bufferSize uint32
	services   map[string]*Service
}

type Service struct {
	impl    interface{}
	methods map[string]grpc.MethodDesc
	streams map[string]grpc.StreamDesc
}

func NewServer(conn net.PacketConn, bufferSize uint32) *UDPServerConn {
	return &UDPServerConn{
		conn:       conn,
		bufferSize: bufferSize,
		services:   map[string]*Service{},
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

func (udpServer *UDPServerConn) handleIncoming(caller net.Addr, data []byte) {
	var payload packet.Packet
	protojson.Unmarshal(data, &payload)

	routingSegments := strings.Split(payload.Method, "/")
	serviceName := routingSegments[1]
	methodName := routingSegments[2]

	service, _ := udpServer.services[serviceName]
	method := service.methods[methodName]

	context.Background()

	md := map[string][]string{}
	for _, entry := range payload.Metadata {
		md[entry.Key] = entry.Values
	}

	ctx := metadata.NewIncomingContext(context.Background(), md)

	println(payload.GetMethod())
	fmt.Printf("%v\n", udpServer.services)

	res, err := method.Handler(service.impl, ctx, func(in interface{}) error {
		return protojson.Unmarshal(payload.Payload, in.(proto.Message))
	}, func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		return handler(ctx, req)
	})

	if err != nil {
		panic(err)
	}

	responseBytes, err := protojson.Marshal(res.(proto.Message))
	if err != nil {
		panic(err)
	}

	responsePacket := &packet.Packet{
		Id:      payload.Id,
		Method:  payload.Method,
		Payload: responseBytes,
	}

	responsePacketBytes, err := protojson.Marshal(responsePacket)

	_, err = udpServer.conn.WriteTo(responsePacketBytes, caller)
	if err != nil {
		fmt.Printf("Error sending bytes: %v\n", err)
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
