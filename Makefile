genproto:
	protoc --proto_path=protos --go_out=protogen --go_opt=paths=source_relative \
    --go-grpc_out=protogen --go-grpc_opt=paths=source_relative \
    protos/*.proto	

cleanproto:
	rm protogen/*

server:
	go run cmd/server/main.go -port 8090

client:
	go run cmd/client/main.go -port 8090

test:
	go test -cover -race ./...

dockershell:
	docker exec -it fileUpAPI sh

.PHONY:
	genproto cleanproto test server client dockershell