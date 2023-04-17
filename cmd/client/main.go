package main

import (
	"bufio"
	"context"
	"fileup/protogen"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

/*
This client application is for testing server.
*/

const filePathPrefix = "tmp/client/"

func main() {
	port := flag.Int("port", 0, "server port")
	host := flag.String("host", "0.0.0.0", "host address")
	file := flag.String("file", "testing.json", "file to upload")
	flag.Parse()

	address := fmt.Sprintf("%s:%d", *host, *port)

	log.Printf("dial server %s", address)

	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatal("cannot dial server:", err)
	}

	fileUpClient := protogen.NewFileUploadServiceClient(conn)

	log.Printf("uploading file: %s", *file)

	testFileUpload(fileUpClient, *file)
}

// tests the file upload
func testFileUpload(fileUpClient protogen.FileUploadServiceClient, fileName string) {
	filePath := fmt.Sprintf("%s/%s", filePathPrefix, fileName)

	file, err := os.Open(filePath)
	if err != nil {
		log.Fatal("cannot open file:", err)
	}
	defer file.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	fStream, err := fileUpClient.FileUpload(ctx)
	if err != nil {
		log.Fatal("cannot upload file:", err)
	}

	req := &protogen.FileUploadRequest{
		Data: &protogen.FileUploadRequest_Info{
			Info: &protogen.FileInfo{
				FileName: fileName,
				FileType: filepath.Ext(fileName),
			},
		},
	}

	err = fStream.Send(req)
	if err != nil {
		log.Fatal("cannot upload file:", err, fStream.RecvMsg(nil))
	}

	reader := bufio.NewReader(file)
	buffer := make([]byte, 1024)

	for {
		n, err := reader.Read(buffer)
		if err == io.EOF {
			break
		}

		req := &protogen.FileUploadRequest{
			Data: &protogen.FileUploadRequest_Chunk{
				Chunk: buffer[:n],
			},
		}

		err = fStream.Send(req)
		if err != nil {
			log.Fatal("cannot send chunk to server:", err, fStream.RecvMsg(nil))
		}
	}

	res, err := fStream.CloseAndRecv()
	if err != nil {
		log.Fatal("cannot receive response:", err)
	}

	log.Printf("file uploaded with id: %s and size: %d", res.GetId(), res.GetSize())
}
