package omgrpc

import (
	"strings"

	"github.com/bsm/openmetrics"
	"google.golang.org/grpc/stats"
	"google.golang.org/grpc/status"
)

// InstrumentCallCount returns default stats.Handler to instrument RPC call count.
// It populates labels it can recognize and leaves others empty:
//
//   - /*method*/ - full method name like "/com.package/MethodName"
//   - /*(status|code)*/ - populated with gRPC code string: https://pkg.go.dev/google.golang.org/grpc/codes#Code
//
func InstrumentCallCount(m openmetrics.CounterFamily) stats.Handler {
	extractors := buildCallExtractors(m.Desc().Labels)

	return CallStatsHandler(func(call *CallStats) {
		labels := extractCallLabels(extractors, call)
		m.With(labels...).Add(1)
	})
}

// InstrumentCallDuration returns default stats.Handler to instrument RPC call duration.
// It populates labels it can recognize and leaves others empty:
//
//   - /*method*/ - full method name like "/com.package/MethodName"
//   - /*(status|code)*/ - populated with gRPC code string: https://pkg.go.dev/google.golang.org/grpc/codes#Code
//
func InstrumentCallDuration(m openmetrics.HistogramFamily) stats.Handler {
	extractors := buildCallExtractors(m.Desc().Labels)

	return CallStatsHandler(func(call *CallStats) {
		labels := extractCallLabels(extractors, call)
		elapsed := call.EndTime.Sub(call.BeginTime)
		m.With(labels...).Observe(float64(elapsed))
	})
}

// InstrumentActiveConns returns default stats.Handler to instrument number of active gRPC connections.
// It populates all the labels empty (if any).
func InstrumentActiveConns(m openmetrics.GaugeFamily) stats.Handler {
	numLabel := len(m.Desc().Labels)

	return ConnStatsHandler(func(conn *ConnStats) {
		labels := make([]string, numLabel) // all empty, nothing to populate here
		switch conn.Status {
		case Connected:
			m.With(labels...).Add(1)
		case Disconnected:
			m.With(labels...).Add(-1)
		}
	})
}

// ----------------------------------------------------------------------------

func buildCallExtractors(labels []string) []func(*CallStats) string {
	extractors := make([]func(*CallStats) string, 0, len(labels))
	for _, l := range labels {
		name := strings.ToLower(l)
		if strings.Contains(name, "method") {
			extractors = append(extractors, extractCallMethod)
		} else if strings.Contains(name, "status") || strings.Contains(name, "code") {
			extractors = append(extractors, extractCallStatus)
		} else {
			extractors = append(extractors, returnEmptyString)
		}
	}
	return extractors
}

func extractCallLabels(extractors []func(*CallStats) string, call *CallStats) []string {
	values := make([]string, 0, len(extractors))
	for _, extract := range extractors {
		values = append(values, extract(call))
	}
	return values
}

func extractCallMethod(call *CallStats) string {
	return call.FullMethodName
}

func extractCallStatus(call *CallStats) string {
	s, _ := status.FromError(call.Error) // returns Unknown status instead of nil
	return s.Code().String()
}

func returnEmptyString(*CallStats) string {
	return ""
}
