package omgrpc

import (
	"strings"
	"time"

	"github.com/bsm/openmetrics"
	"google.golang.org/grpc/stats"
)

// InstrumentCallCount returns default stats.Handler to instrument RPC call count.
// It populates labels it can recognize and leaves others empty:
//
//   - "method" - full method name like "/com.package/MethodName"
//   - "status" or "code" - populated with gRPC code string: https://pkg.go.dev/google.golang.org/grpc/codes#Code
//
func InstrumentCallCount(m openmetrics.CounterFamily) stats.Handler {
	extractors := buildCallExtractors(m.Desc().Labels)

	return CallStatsHandler(func(call *CallStats) {
		labels := extractCallLabels(extractors, call)
		m.With(labels...).Add(1)
	})
}

// InstrumentCallDuration returns default stats.Handler to instrument RPC call duration in units configured for metric.
// It populates labels it can recognize and leaves others empty:
//
//   - "method" - full method name like "/com.package/MethodName"
//   - "status" or "code" - populated with gRPC code string: https://pkg.go.dev/google.golang.org/grpc/codes#Code
//
func InstrumentCallDuration(m openmetrics.HistogramFamily) stats.Handler {
	desc := m.Desc()
	extractors := buildCallExtractors(desc.Labels)
	convertDuration := makeDurationConverter(desc.Unit)

	return CallStatsHandler(func(call *CallStats) {
		labels := extractCallLabels(extractors, call)
		m.With(labels...).Observe(convertDuration(call.Duration()))
	})
}

// InstrumentActiveConns returns default stats.Handler to instrument number of active gRPC connections.
// It populates no labels.
func InstrumentActiveConns(m openmetrics.GaugeFamily) stats.Handler {
	return ConnStatsHandler(func(conn *ConnStats) {
		switch conn.Status {
		case Connected:
			m.With().Add(1)
		case Disconnected:
			m.With().Add(-1)
		}
	})
}

// ----------------------------------------------------------------------------

func buildCallExtractors(labels []string) []func(*CallStats) string {
	if len(labels) == 0 {
		return nil
	}

	extractors := make([]func(*CallStats) string, 0, len(labels))
	for _, l := range labels {
		if strings.EqualFold(l, "method") {
			extractors = append(extractors, extractCallMethod)
		} else if strings.EqualFold(l, "status") || strings.EqualFold(l, "code") {
			extractors = append(extractors, extractCallStatus)
		} else {
			extractors = append(extractors, returnEmptyString)
		}
	}
	return extractors
}

func extractCallLabels(extractors []func(*CallStats) string, call *CallStats) []string {
	if len(extractors) == 0 {
		return nil
	}

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
	return call.Code().String()
}

func returnEmptyString(*CallStats) string {
	return ""
}

func makeDurationConverter(unit string) func(time.Duration) float64 {
	switch unit {
	case "nanoseconds":
		return func(d time.Duration) float64 { return float64(d.Nanoseconds()) }
	case "microseconds":
		return func(d time.Duration) float64 { return float64(d.Microseconds()) }
	case "milliseconds":
		return func(d time.Duration) float64 { return float64(d.Milliseconds()) }
	default:
		return func(d time.Duration) float64 { return d.Seconds() }
	}
}
