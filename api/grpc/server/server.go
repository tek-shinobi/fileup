package server

import (
	"context"
	"fileup/protogen"
	"fileup/service"
	"io"
	"log"

	"fileup/constants"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type Server interface {
	protogen.FileUploadServiceServer
}

type server struct {
	service service.Service
	log     *log.Logger
	protogen.UnimplementedFileUploadServiceServer
}

func NewServer(service service.Service, log *log.Logger) Server {
	return &server{
		service: service,
		log:     log,
	}
}

// FileUpload is a client-streaming RPC to upload a file
func (s *server) FileUpload(fStream protogen.FileUploadService_FileUploadServer) error {
	req, err := fStream.Recv()
	if err != nil {
		return handleError(status.Errorf(codes.Unknown, "file info not received:%v", err))
	}

	fileName := req.GetInfo().GetFileName()
	fileType := req.GetInfo().GetFileType()
	s.log.Printf("upload request received for %s", fileName)

	fileSize := int64(0)

	// get service
	fileNameOnDisk, err := s.service.Init(fileType)
	if err != nil {
		return handleError(status.Errorf(codes.Internal, "could not create file:%v", err))
	}
	defer s.service.Close(fileNameOnDisk)

	for {
		// handle request cancellation and request timeout
		if err := contextError(fStream.Context()); err != nil {
			return err
		}

		s.log.Print("waiting for next chunk")

		req, err := fStream.Recv()

		// exit the loop when EOF detected
		if err == io.EOF {
			s.log.Print("end of file upload")
			break
		}

		if err != nil {
			return handleError(status.Errorf(codes.Unknown, "cannot receive file chunk:%v", err))
		}

		chunk := req.GetChunk()
		size := len(chunk)

		s.log.Printf("received chunk with size: %d", size)

		fileSize += int64(size)

		if fileSize > constants.MaxFileSize {
			return handleError(status.Errorf(codes.InvalidArgument, "file size too large: %d > %d", fileSize, constants.MaxFileSize))
		}

		// persist the data
		err = s.service.SaveChunk(fileNameOnDisk, chunk)
		if err != nil {
			return handleError(status.Errorf(codes.Internal, "cannot write chunk data: %v", err))
		}
	}

	// create and send response
	res := &protogen.FileUploadResponse{
		Id:   fileNameOnDisk,
		Size: uint32(fileSize),
	}

	err = fStream.SendAndClose(res)
	if err != nil {
		return handleError(status.Errorf(codes.Unknown, "cannot send response: %v", err))
	}

	log.Printf("saved file:%s as %s and size:%d", fileName, fileNameOnDisk, fileSize)

	return nil
}

// helper methods

func handleError(err error) error {
	if err != nil {
		log.Printf("cannot receive file info:%v", err)
	}
	return err
}

func contextError(ctx context.Context) error {
	switch ctx.Err() {
	case context.Canceled:
		return handleError(status.Error(codes.Canceled, "request canceled"))
	case context.DeadlineExceeded:
		return handleError(status.Error(codes.DeadlineExceeded, "deadline exceeded"))
	default:
		return nil
	}
}
