package omgrpc

import (
	"context"
	"fmt"
	"os"

	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
)

// StatsHandler is a https://pkg.go.dev/google.golang.org/grpc/stats#Handler implementation.
type StatsHandler struct {
	// OnData func(sz int, dir Direction(In/Out), when Lifecycle(Header/Payload/Trailer))
	// OnRequest func(fullMethod string, status *grpc/status.Status, elapsed time.Duration)
	// OnConnect func(increment int)
}

// to have something default:
// func NewDefaultStatsHandler(reg openmetrics.Registry) *StatsHandler { ... }

// TagRPC attaches omgrpc-internal data to RPC context.
func (h *StatsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	// this method is called before HandleRPC
	// and it seems to be impossible to extract full method from Unary ctx,
	// so "remember" it here:
	return setContextMethod(ctx, info.FullMethodName)
}

// HandleRPC processes the RPC stats.
func (h *StatsHandler) HandleRPC(ctx context.Context, stat stats.RPCStats) {
	// we handle almost all the RPCStats types, so can extract method before switch:
	method := getContextMethod(ctx)

	switch s := stat.(type) {

	// grpc/stats declares WireLength to be "compressed, signed, encrypted" data size
	// just Length is not that informative, it's a raw data size.

	case *stats.InHeader:
		fmt.Fprintf(os.Stderr, "- transfer size: method=%s, wire_length=%d, when=%T\n", method, s.WireLength, s)

	case *stats.InPayload:
		fmt.Fprintf(os.Stderr, "- transfer size: method=%s, wire_length=%d, when=%T\n", method, s.WireLength, s)

	case *stats.InTrailer:
		fmt.Fprintf(os.Stderr, "- transfer size: method=%s, wire_length=%d, when=%T\n", method, s.WireLength, s)

	// case *stats.OutHeader: // no WireLength/Length data provided

	case *stats.OutPayload:
		fmt.Fprintf(os.Stderr, "- transfer size: method=%s, wire_length=%d, when=%T\n", method, s.WireLength, s)

	// case *stats.OutTrailer: // WireLength is deprecated here

	case *stats.End:
		elapsed := s.EndTime.Sub(s.BeginTime)
		status, _ := status.FromError(s.Error) // can return Unknown status, but never nil

		fmt.Fprintf(os.Stderr, "- request timing: method=%s, elapsed=%f, status=%s\n", method, elapsed.Seconds(), status.Code().String())
	}
}

// TagConn implements grpc/stats.Handler interface and does nothing.
func (h *StatsHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return ctx
}

// HandleConn processes the Conn stats.
func (h *StatsHandler) HandleConn(ctx context.Context, stat stats.ConnStats) {
	switch stat.(type) {
	case *stats.ConnBegin:
		fmt.Fprintf(os.Stderr, "- connection: +1\n")
	case *stats.ConnEnd:
		fmt.Fprintf(os.Stderr, "- connection: -1\n")
	}
}
