package main

import (
	lgr "fileup/api/grpc/middlewares/logger"
	"fileup/api/grpc/server"
	"fileup/protogen"
	"fileup/repo"
	"fileup/service"
	"flag"
	"fmt"
	"log"
	"net"
	"os"

	"google.golang.org/grpc"
)

/*
Server application for launching FileUp server
*/

func main() {
	port := flag.Int("port", 8090, "server port")
	host := flag.String("host", "0.0.0.0", "host address")
	fileDir := flag.String("dir", "tmp/server", "folder where files uploaded")
	processedJSONfileDir := flag.String("jsondir", "tmp/json", "folder where files uploaded")
	flag.Parse()

	// init logger
	logger := log.New(os.Stdout, "", log.Lmsgprefix)

	address := fmt.Sprintf("%s:%d", *host, *port)
	logger.Printf("server listening on %s", address)

	// instantiate stakeholder services
	repo := repo.NewRepo(logger, *fileDir, *processedJSONfileDir)
	service := service.NewService(repo, logger)
	serverRPCs := server.NewServer(service, logger)
	server := grpc.NewServer(
		grpc.StreamInterceptor(
			lgr.StreamServerInterceptor,
		),
	)

	// register rpcs with server
	protogen.RegisterFileUploadServiceServer(server, serverRPCs)

	// configure and launch the server
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Fatal("cannot start server:", err)
	}

	err = server.Serve(listener)
	if err != nil {
		log.Fatal("cannot start server:", err)
	}
}
