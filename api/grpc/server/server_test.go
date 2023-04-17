package server_test

import (
	"bufio"
	"context"
	"fileup/api/grpc/server"
	"fileup/protogen"
	"fileup/repo"
	"fileup/service"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const serverFilePrefix = "../../../tmp/server"
const serverJSONFilePrefix = "../../../tmp/json"

func TestFileUpload(t *testing.T) {
	t.Parallel()

	logger := log.New(os.Stdout, "", log.Lmsgprefix)
	repo := repo.NewRepo(logger, serverFilePrefix, serverJSONFilePrefix)
	service := service.NewService(repo, logger)

	address := startTestServer(t, service, logger)
	client := newTestClient(t, address)

	filePath := "../../../tmp/client/testing.json"
	file, err := os.Open(filePath)
	require.NoError(t, err)
	defer file.Close()

	fStream, err := client.FileUpload(context.Background())
	require.NoError(t, err)

	fileType := filepath.Ext(filePath)

	req := &protogen.FileUploadRequest{
		Data: &protogen.FileUploadRequest_Info{
			Info: &protogen.FileInfo{
				FileName: filePath,
				FileType: fileType,
			},
		},
	}

	err = fStream.Send(req)
	require.NoError(t, err)

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)
	size := 0

	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}

		size += n

		req := &protogen.FileUploadRequest{
			Data: &protogen.FileUploadRequest_Chunk{
				Chunk: buffer[:n],
			},
		}

		err = fStream.Send(req)
		require.NoError(t, err)
	}

	res, err := fStream.CloseAndRecv()
	require.NoError(t, err)
	// id should not be zero value
	require.NotZero(t, res.GetId())
	// the sizes should be equal
	require.EqualValues(t, size, res.GetSize())

	// check the file actually got saved
	savedFilePath := fmt.Sprintf("%s/%s", serverFilePrefix, res.GetId())
	require.FileExists(t, savedFilePath)

	// remove the file at the end of the test
	require.NoError(t, os.Remove(savedFilePath))
}

func newTestClient(t *testing.T, serverAddress string) protogen.FileUploadServiceClient {
	conn, err := grpc.Dial(serverAddress, grpc.WithTransportCredentials(insecure.NewCredentials()))
	require.NoError(t, err)
	return protogen.NewFileUploadServiceClient(conn)
}

func startTestServer(t *testing.T, service service.Service, logger *log.Logger) string {
	serverRPCs := server.NewServer(service, logger)
	server := grpc.NewServer()

	// register rpcs with server
	protogen.RegisterFileUploadServiceServer(server, serverRPCs)

	// configure and launch the server
	listener, err := net.Listen("tcp", ":0") // random available port
	require.NoError(t, err)

	go server.Serve(listener)

	return listener.Addr().String()
}
