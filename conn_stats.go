package omgrpc

import (
	"context"
	"net"
	"sync"

	"google.golang.org/grpc/stats"
)

// ConnStats holds connection stats.
type ConnStats struct {
	Client                bool // indicates client-side stats
	LocalAddr, RemoteAddr net.Addr
	Connected             bool // indicates if emitted stats is "client connected" or "client disconnected" event
}

var connStatsPool = sync.Pool{
	New: func() interface{} {
		return new(ConnStats)
	},
}

var contextKeyConnStats struct{}

func setConnStats(ctx context.Context, conn *ConnStats) context.Context {
	return context.WithValue(ctx, contextKeyConnStats, conn)
}

func getConnStats(ctx context.Context) *ConnStats {
	// internal, expected to be used carefully and never panic:
	return ctx.Value(contextKeyConnStats).(*ConnStats)
}

// ----------------------------------------------------------------------------

// ConnStatsHandler implements https://pkg.go.dev/google.golang.org/grpc/stats#Handler for RPC connections.
// ConnStats argument is reused (pooled), so pointer cannot be stored - copy instead.
//
// It assumes that stats.Handler methods are never called concurrently.
type ConnStatsHandler func(*ConnStats)

// TODO: default ConnStats handler tracking default metrics.
// func NewDefaultConnStatsHandler(reg openmetrics.Registry) ConnStatsHandler

// TagConn implements grpc/stats.Handler interface and does nothing.
func (h ConnStatsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	return ctx
}

// TagConn implements grpc/stats.Handler interface and does nothing.
func (h ConnStatsHandler) HandleRPC(ctx context.Context, stat stats.RPCStats) {}

// TagRPC attaches omgrpc-internal data to connection context.
func (h ConnStatsHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	conn := connStatsPool.Get().(*ConnStats)
	conn.LocalAddr = info.LocalAddr
	conn.RemoteAddr = info.RemoteAddr
	return setConnStats(ctx, conn)
}

// HandleConn implements grpc/stats.Handler interface and does nothing.
func (h ConnStatsHandler) HandleConn(ctx context.Context, stat stats.ConnStats) {
	conn := getConnStats(ctx)

	switch s := stat.(type) {
	case *stats.ConnBegin:
		conn.Connected = true
		conn.Client = s.Client
		h(conn)

	case *stats.ConnEnd:
		conn.Connected = false
		h(conn)
		*conn = ConnStats{}
		connStatsPool.Put(conn)
	}
}
