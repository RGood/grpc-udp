package utils

import (
	"context"

	"github.com/RGood/go-grpc-udp/internal/generated/packet"
	"google.golang.org/grpc/metadata"
)

func ContextToMetadata(ctx context.Context) []*packet.MapEntry {
	md, _ := metadata.FromOutgoingContext(ctx)
	return MetadataToMapEntries(md)
}

func MetadataToMapEntries(md metadata.MD) []*packet.MapEntry {
	mdList := []*packet.MapEntry{}
	for k, v := range md {
		mdList = append(mdList, &packet.MapEntry{
			Key:    k,
			Values: v,
		})
	}

	return mdList
}
