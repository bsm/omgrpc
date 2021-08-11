package testpb

import (
	"context"
	"errors"
	"io"
)

// TODO: review and maybe switch to https://pkg.go.dev/google.golang.org/grpc@v1.39.1/test/grpc_testing#UnsafeTestServiceServer

type TestServerImpl struct {
	UnimplementedTestServer

	UnaryError  error
	StreamError error
}

func (s *TestServerImpl) Unary(ctx context.Context, req *Message) (*Message, error) {
	if s.UnaryError != nil {
		return nil, s.UnaryError
	}
	return &Message{Payload: "Unary: " + req.Payload}, nil
}
func (s *TestServerImpl) Stream(ss Test_StreamServer) error {
	if s.StreamError != nil {
		return s.StreamError
	}

	for {
		req, err := ss.Recv()
		if errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return err
		}
		if err := ss.Send(&Message{Payload: "Stream: " + req.Payload}); err != nil {
			return err
		}
	}
}
