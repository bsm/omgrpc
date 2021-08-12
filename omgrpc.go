package omgrpc

import (
	"context"
	"time"

	"google.golang.org/grpc/stats"
)

// DataDirection defines data direction as seen by gRPC peer (either Client or Server).
type DataDirection string

const (
	DirectionIn  DataDirection = "in"
	DirectionOut DataDirection = "out"
)

func (d DataDirection) String() string {
	return string(d)
}

// DataPhase defines when data transfer occurs.
type DataPhase string

const (
	PhaseHeader  DataPhase = "header"
	PhasePayload DataPhase = "payload"
	PhaseTrailer DataPhase = "trailer"
)

func (p DataPhase) String() string {
	return string(p)
}

// StatsHandler is a https://pkg.go.dev/google.golang.org/grpc/stats#Handler implementation.
//
// All the properties/callbacks are optional.
type StatsHandler struct {
	// OnData runs when data transfer event occurs.
	// Size argument is a compressed, signed and encrypted data size,
	// just like what is transferred over the network.
	OnData func(fullMethod string, phase DataPhase, dir DataDirection, size int)

	// OnCall runs when RPC call is served.
	// It is provided with a (nil-able) handler error,
	// which can be examined further with e.g. grpc/status.FromError(err).Code().
	OnCall func(fullMethod string, err error, elapsed time.Duration)

	// OnConn runs when connections are made (positive increment)
	// or when connections are closed (negative increment).
	OnConn func(increment int)
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
		h.onData(method, PhaseHeader, DirectionIn, s.WireLength)

	case *stats.InPayload:
		h.onData(method, PhasePayload, DirectionIn, s.WireLength)

	case *stats.InTrailer:
		h.onData(method, PhaseTrailer, DirectionIn, s.WireLength)

	// case *stats.OutHeader: // no WireLength/Length data provided

	case *stats.OutPayload:
		h.onData(method, PhasePayload, DirectionOut, s.WireLength)

	// case *stats.OutTrailer: // WireLength is deprecated here

	case *stats.End:
		elapsed := s.EndTime.Sub(s.BeginTime)
		h.onCall(method, s.Error, elapsed)
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
		h.onConn(1)
	case *stats.ConnEnd:
		h.onConn(-1)
	}
}

func (h *StatsHandler) onData(fullMethod string, phase DataPhase, dir DataDirection, size int) {
	if cb := h.OnData; cb != nil {
		cb(fullMethod, phase, dir, size)
	}
}

func (h *StatsHandler) onCall(fullMethod string, err error, elapsed time.Duration) {
	if cb := h.OnCall; cb != nil {
		cb(fullMethod, err, elapsed)
	}
}

func (h *StatsHandler) onConn(increment int) {
	if cb := h.OnConn; cb != nil {
		cb(increment)
	}
}
