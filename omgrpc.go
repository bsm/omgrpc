package omgrpc

import (
	"context"
	"time"

	"github.com/bsm/openmetrics"
	"google.golang.org/grpc"
	"google.golang.org/grpc/status"
)

// ServerMetrics defines common gRPC server metrics bundle.
type ServerMetrics interface {
	// RequestCount should return counter which accepts 2 labels:
	// full gRPC method name (like "/package.service/method")
	// and gRPC status label (like "OK" or "Unknown" - see https://pkg.go.dev/google.golang.org/grpc/codes)
	RequestCount() openmetrics.CounterFamily

	// RequestCount should return histogram which accepts 2 labels:
	// full gRPC method name (like "/package.service/method")
	// and gRPC status label (like "OK" or "Unknown" - see https://pkg.go.dev/google.golang.org/grpc/codes)
	RequestDuration() openmetrics.HistogramFamily
}

// UnaryServerMetrics defines gRPC unary server metrics bundle.
type UnaryServerMetrics interface {
	ServerMetrics

	// can be extended later
}

// StreamServerMetrics defines gRPC stream server metrics bundle.
type StreamServerMetrics interface {
	ServerMetrics

	// can be extended later
}

// ----------------------------------------------------------------------------

var _ UnaryServerMetrics = (*DefaultServerMetrics)(nil)
var _ StreamServerMetrics = (*DefaultServerMetrics)(nil)

// DefaultServerMetrics provides both UnaryServerMetrics and StreamServerMetrics.
type DefaultServerMetrics struct {
	requestCount    openmetrics.CounterFamily
	requestDuration openmetrics.HistogramFamily
}

// NewDefaultServerMetrics builds metrics bundle with the following metrics/labels:
//   "grpc_request" counter with labels "full_method" and "status"
//   "grpc_request" histogram (request duration) with labels "full_method" and "status" and bounds: .1, .2, 0.5, 1
func NewDefaultServerMetrics(reg openmetrics.Registry) *DefaultServerMetrics {
	return &DefaultServerMetrics{
		requestCount: reg.Counter(openmetrics.Desc{
			Name:   "grpc_requests",
			Labels: []string{"full_method", "status"},
		}),
		requestDuration: reg.Histogram(openmetrics.Desc{
			Name:   "grpc_requests",
			Unit:   "seconds",
			Labels: []string{"full_method", "status"},
		}, []float64{.1, .2, 0.5, 1}),
	}
}

func (m *DefaultServerMetrics) RequestCount() openmetrics.CounterFamily {
	return m.requestCount
}

func (m *DefaultServerMetrics) RequestDuration() openmetrics.HistogramFamily {
	return m.requestDuration
}

// ----------------------------------------------------------------------------

// UnaryServer builds an unary server interceptor.
func UnaryServer(metrics UnaryServerMetrics) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		started := time.Now()
		resp, err := handler(ctx, req) // TODO: check if gRPC will recover from panics or we should?
		statusLabel := status.Code(err).String()

		metrics.RequestDuration().With(info.FullMethod, statusLabel).Observe(time.Since(started).Seconds())
		metrics.RequestCount().With(info.FullMethod, statusLabel).Add(1)

		return resp, err
	}
}

// StreamServer builds a streaming server interceptor.
func StreamServer(metrics StreamServerMetrics) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		started := time.Now()
		// Note: stream can be monitored for msgs recved/sent as well, see:
		// https://github.com/piotrkowalczuk/promgrpc/blob/d34dd04b874a678ba14e884abb6b1b1b1701070b/prometheus.go#L533-L553
		err := handler(srv, ss) // TODO: check if gRPC will recover from panics or we should?
		statusLabel := status.Code(err).String()

		metrics.RequestDuration().With(info.FullMethod, statusLabel).Observe(time.Since(started).Seconds())
		metrics.RequestCount().With(info.FullMethod, statusLabel).Add(1)

		return err
	}
}
