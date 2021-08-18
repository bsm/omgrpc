package omgrpc

import (
	"context"
	"net"
	"sync"

	"google.golang.org/grpc/stats"
)

// ConnStatus holds connection status.
type ConnStatus int8

const (
	Connected ConnStatus = iota
	Disconnected
)

// ConnStats holds connection stats.
type ConnStats struct {
	IsClient bool // indicates client-side stats
	Status   ConnStatus

	LocalAddr, RemoteAddr net.Addr
	BytesRecv, BytesSent  int // supported only for server side, only when Connected=false
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

// TagConn implements grpc/stats.Handler interface and does nothing.
func (h ConnStatsHandler) TagRPC(ctx context.Context, info *stats.RPCTagInfo) context.Context {
	return ctx
}

// TagConn tracks server-side
func (h ConnStatsHandler) HandleRPC(ctx context.Context, stat stats.RPCStats) {
	if stat.IsClient() {
		return // ctx has tagged conn only server-side
	}

	conn := getConnStats(ctx)

	switch s := stat.(type) {

	case *stats.InHeader:
		conn.BytesRecv += s.WireLength

	case *stats.InPayload:
		conn.BytesRecv += s.WireLength

	case *stats.InTrailer:
		conn.BytesRecv += s.WireLength

	// case *stats.OutHeader: // no WireLength in OutHeader and OutTrailer (at least as of grpc@1.40.0)
	// case *stats.OutTrailer: // WireLength is deprecated here

	case *stats.OutPayload:
		conn.BytesSent += s.WireLength

	}
}

// TagRPC attaches omgrpc-internal data to connection context.
func (h ConnStatsHandler) TagConn(ctx context.Context, info *stats.ConnTagInfo) context.Context {
	conn := connStatsPool.Get().(*ConnStats)
	conn.LocalAddr = info.LocalAddr
	conn.RemoteAddr = info.RemoteAddr
	return setConnStats(ctx, conn)
}

// TagRPC attaches omgrpc-internal data to connection context.
func (h ConnStatsHandler) HandleConn(ctx context.Context, stat stats.ConnStats) {
	conn := getConnStats(ctx)

	switch s := stat.(type) {
	case *stats.ConnBegin:
		conn.Status = Connected
		conn.IsClient = s.Client
		h(conn)

	case *stats.ConnEnd:
		conn.Status = Disconnected
		h(conn)
		*conn = ConnStats{}
		connStatsPool.Put(conn)
	}
}
