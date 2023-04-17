package logger

import (
	"log"

	"google.golang.org/grpc"
)

type wrappedStream struct {
	grpc.ServerStream
}

func (w *wrappedStream) RecvMsg(m interface{}) error {
	log.Printf("Received %T - %v", m, m)
	return w.ServerStream.RecvMsg(m)
}

func (w *wrappedStream) SentMsg(m interface{}) error {
	log.Printf("Sent %T - %v", m, m)
	return w.ServerStream.SendMsg(m)
}

func newWrappedStream(s grpc.ServerStream) grpc.ServerStream {
	return &wrappedStream{s}
}

// StreamServerInterceptor adds basic request logging
// more exhastive logging, like adding requestID, can be achieved
// using third party library since it involves modifying serverStream context
func StreamServerInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	log.Println("StreamServerInterceptor method invoked START:", info.FullMethod)

	err := handler(srv, newWrappedStream(ss))

	log.Println("StreamServerInterceptor method invoked END:", info.FullMethod)

	return err
}
