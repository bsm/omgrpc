package omgrpc

import (
	"github.com/bsm/openmetrics"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
)

// NewDefaultCallStatsHandler builds a CallStatsHandler that tracks default metrics.
// It will default to openmetrics.DefaultRegistry() on nil.
//
//   - "grpc_call" counter with "method", "status" labels
//   - "grpc_call" histogram with "method", "status" labels; seconds unit; .1, .2, 0.5, 1 buckets
//
func NewDefaultCallStatsHandler(reg *openmetrics.Registry) stats.Handler {
	if reg == nil {
		reg = openmetrics.DefaultRegistry()
	}

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

	return CallStatsHandler(func(call *CallStats) {
		s, _ := status.FromError(call.Error) // returns Unknown status instead of nil
		status := s.Code().String()

		callCount.With(call.FullMethodName, status).Add(1)
		callDuration.With(call.FullMethodName, status).Observe(call.EndTime.Sub(call.BeginTime).Seconds())
	})
}

// ----------------------------------------------------------------------------

// NewDefaultConnStatsHandler builds a ConnStatsHandler that tracks default metrics.
// It will default to openmetrics.DefaultRegistry() on nil.
//
//   - "grpc_active_conns" gauge with no labels
//
func NewDefaultConnStatsHandler(reg *openmetrics.Registry) stats.Handler {
	if reg == nil {
		reg = openmetrics.DefaultRegistry()
	}

	activeConnGauge := reg.Gauge(openmetrics.Desc{
		Name:   "grpc_active_conns",
		Help:   "gRPC active connections gauge",
		Labels: []string{},
	})

	return ConnStatsHandler(func(conn *ConnStats) {
		switch conn.Status {
		case Connected:
			activeConnGauge.With().Add(1)
		case Disconnected:
			activeConnGauge.With().Add(-1)
		}
	})
}
