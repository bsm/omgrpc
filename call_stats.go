package omgrpc

import (
	"context"
	"net"
	"time"

	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
)

// CallStats holds all the RPC call-related data
// that can be collected by stats handler.
type CallStats struct {

	// Fields can be aggregated when it makes sense, e.g.:
	// grpc/stats.*.WireLength
	//
	// Duplicated fields are set on first occurrence, e.g.:
	// grpc/stats.RPCTagInfo.FailFast
	//
	// Ambiguous fields are not collected at all, e.g.:
	// grpc/stats.*.Compression

	// https://pkg.go.dev/google.golang.org/grpc/stats#RPCTagInfo

	// FullMethodName is the RPC method in the format of /package.service/method.
	FullMethodName string
	// FailFast indicates if this RPC is failfast.
	// This field is only valid on client side, it's always false on server side.
	FailFast bool

	// https://pkg.go.dev/google.golang.org/grpc/stats#Begin

	// Client is true if this Begin is from client side.
	Client bool
	// BeginTime is the time when the RPC begins.
	BeginTime time.Time
	// IsClientStream indicates whether the RPC is a client streaming RPC.
	IsClientStream bool
	// IsServerStream indicates whether the RPC is a server streaming RPC.
	IsServerStream bool

	// https://pkg.go.dev/google.golang.org/grpc/stats#InHeader

	// InHeader contains the header metadata received.
	InHeader metadata.MD

	// Assumed to happen once:
	// https://pkg.go.dev/google.golang.org/grpc/stats#InTrailer

	// InTrailer contains the trailer metadata received from the server. This
	// field is only valid if for client stats.
	InTrailer metadata.MD

	// https://pkg.go.dev/google.golang.org/grpc/stats#OutHeader

	// OutHeader contains the header metadata sent.
	OutHeader metadata.MD

	// https://pkg.go.dev/google.golang.org/grpc/stats#OutTrailer

	// OutTrailer contains the trailer metadata sent to the client. This
	// field is only valid if this OutTrailer is from the server side.
	OutTrailer metadata.MD

	// https://pkg.go.dev/google.golang.org/grpc/stats#End

	// EndTime is the time when the RPC ends.
	EndTime time.Time
	// Error is the error the RPC ended with. It is an error generated from
	// status.Status and can be converted back to status.Status using
	// status.FromError if non-nil.
	Error error

	// Client/Server exclusive fields:

	// The following fields are set from:
	// for server stats:
	// https://pkg.go.dev/google.golang.org/grpc/stats#InHeader
	// for client stats:
	// https://pkg.go.dev/google.golang.org/grpc/stats#OutHeader

	// RemoteAddr is the remote address of the corresponding connection.
	RemoteAddr net.Addr
	// LocalAddr is the local address of the corresponding connection.
	LocalAddr net.Addr

	// Agregated (multi-event) fields:

	// InWireLength is all compressed, signed, encrypted INPUT data size.
	InWireLength int
	// OutWireLength is all compressed, signed, encrypted OUTPUT data size.
	OutWireLength int
}

var contextKeyCallStats struct{}

func setCallStats(ctx context.Context, call *CallStats) context.Context {
	return context.WithValue(ctx, contextKeyCallStats, call)
}

func getCallStats(ctx context.Context) *CallStats {
	// internal, expected to be used carefully and never panic:
	return ctx.Value(contextKeyCallStats).(*CallStats)
}

// --------------------------------------------------------------------------------------

// CallStatsHandler implements https://pkg.go.dev/google.golang.org/grpc/stats#Handler for RPC calls.
//
// It assumes that stats.Handler methods are never called concurrently.
type CallStatsHandler func(*CallStats)

// TagRPC attaches omgrpc-internal data to RPC context.
func (h CallStatsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	// this method is called before HandleRPC, init CallStats at this point:
	return setCallStats(ctx, &CallStats{
		FullMethodName: info.FullMethodName,
		FailFast:       info.FailFast,
	})
}

// HandleRPC processes the RPC stats.
func (h CallStatsHandler) HandleRPC(ctx context.Context, stat stats.RPCStats) {
	// pretty much all of the RPCStats types are handled,
	// so prepare CallStats once:
	call := getCallStats(ctx)

	switch s := stat.(type) {

	case *stats.Begin:
		call.Client = s.Client
		call.BeginTime = s.BeginTime
		call.IsClientStream = s.IsClientStream
		call.IsServerStream = s.IsServerStream

	case *stats.InHeader:
		call.InHeader = s.Header
		if !s.Client { // server
			call.RemoteAddr = s.RemoteAddr
			call.LocalAddr = s.LocalAddr
		}

	case *stats.InPayload:

	case *stats.InTrailer:
		call.InTrailer = s.Trailer

	case *stats.OutHeader:
		call.OutHeader = s.Header
		if s.Client { // client
			call.RemoteAddr = s.RemoteAddr
			call.LocalAddr = s.LocalAddr
		}

	case *stats.OutPayload:

	case *stats.OutTrailer:
		call.OutTrailer = s.Trailer

	case *stats.End:
		call.EndTime = s.EndTime
		call.Error = s.Error

	}
}

// TagConn implements grpc/stats.Handler interface and does nothing.
func (h CallStatsHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return ctx
}

// HandleConn implements grpc/stats.Handler interface and does nothing.
func (h CallStatsHandler) HandleConn(ctx context.Context, stat stats.ConnStats) {}
