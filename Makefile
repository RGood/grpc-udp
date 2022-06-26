proto:
	protoc -I=protos \
		--go_out=. --go_opt=module=github.com/RGood/go-grpc-udp \
		--go-grpc_out=. --go-grpc_opt=module=github.com/RGood/go-grpc-udp \
		$$(find ./protos -name "*.proto")