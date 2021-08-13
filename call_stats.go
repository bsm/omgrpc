package omgrpc

import (
	"context"
	"net"
	"sync"
	"time"

	"github.com/bsm/openmetrics"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
)

// CallStats holds all the RPC call-related data
// that can be collected by stats handler.
type CallStats struct {
	Client   bool // indicates client-side stats
	FailFast bool // only valid for client

	FullMethodName                 string
	IsClientStream, IsServerStream bool
	BeginTime, EndTime             time.Time
	InHeader, InTrailer            metadata.MD
	OutHeader, OutTrailer          metadata.MD
	LocalAddr, RemoteAddr          net.Addr
	BytesRecv, BytesSent           int

	Error error // RPC call error, can be examined with s, _ := grpc/status.FromError(err); s.Code()
}

var callStatsPool = sync.Pool{
	New: func() interface{} {
		return new(CallStats)
	},
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
// CallStats argument is reused (pooled), so pointer cannot be stored - copy instead.
//
// It assumes that stats.Handler methods are never called concurrently.
type CallStatsHandler func(*CallStats)

// NewDefaultCallStatsHandler builds a CallStatsHandler that tracks default metrics.
//
//   - "grpc_call" counter with "method", "status" labels
//   - "grpc_call" histogram with "method", "status" labels; seconds unit; .1, .2, 0.5, 1 buckets
//
func NewDefaultCallStatsHandler(reg openmetrics.Registry) CallStatsHandler {
	callCount := reg.Counter(openmetrics.Desc{
		Name:   "grpc_call",
		Help:   "gRPC call counter",
		Labels: []string{"method", "status"},
	})
	callDuration := reg.Histogram(openmetrics.Desc{
		Name:   "grpc_call",
		Unit:   "seconds",
		Help:   "gRPC call timing",
		Labels: []string{"method", "status"},
	}, []float64{.1, .2, 0.5, 1})

	return func(call *CallStats) {
		s, _ := status.FromError(call.Error) // returns Unknown status instead of nil
		status := s.Code().String()

		callCount.With(call.FullMethodName, status).Add(1)
		callDuration.With(call.FullMethodName, status).Observe(call.EndTime.Sub(call.BeginTime).Seconds())
	}
}

// TagRPC attaches omgrpc-internal data to RPC context.
func (h CallStatsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	// this method is called before HandleRPC, init CallStats at this point:
	call := callStatsPool.Get().(*CallStats)
	call.FullMethodName = info.FullMethodName
	call.FailFast = info.FailFast
	return setCallStats(ctx, call)
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
		call.BytesRecv += s.WireLength

	case *stats.InPayload:
		call.BytesRecv += s.WireLength

	case *stats.InTrailer:
		call.InTrailer = s.Trailer
		call.BytesRecv += s.WireLength

	case *stats.OutHeader:
		call.OutHeader = s.Header
		if s.Client { // client
			call.RemoteAddr = s.RemoteAddr
			call.LocalAddr = s.LocalAddr
		}
		// no WireLength here (at least as of grpc@1.40.0)

	case *stats.OutPayload:
		call.BytesSent += s.WireLength

	case *stats.OutTrailer:
		call.OutTrailer = s.Trailer
		// WireLength is deprecated here

	case *stats.End:
		call.EndTime = s.EndTime
		call.Error = s.Error
		h(call) // "submit" collected stats
		*call = CallStats{}
		callStatsPool.Put(call)

	}
}

// TagConn implements grpc/stats.Handler interface and does nothing.
func (h CallStatsHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	return ctx
}

// HandleConn implements grpc/stats.Handler interface and does nothing.
func (h CallStatsHandler) HandleConn(ctx context.Context, stat stats.ConnStats) {}
