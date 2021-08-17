# omgrpc - [OpenMetrics](https://github.com/bsm/openmetrics) [gRPC](https://github.com/grpc/grpc-go) middleware

[![Lint](https://github.com/bsm/omgrpc/actions/workflows/lint.yml/badge.svg)](https://github.com/bsm/omgrpc/actions/workflows/lint.yml)
[![Test](https://github.com/bsm/omgrpc/actions/workflows/test.yml/badge.svg)](https://github.com/bsm/omgrpc/actions/workflows/test.yml)

## Quickstart

```go
import (
  "github.com/bsm/omgrpc"
  "github.com/bsm/openmetrics"
  "google.golang.org/grpc"
)

func initGRPCServer(
  fooServer yourproto.FooServer,

  callCount openmetrics.CounterFamily,      // with tags: "method", "status"
  callDuration openmetrics.HistogramFamily, // with tags: "method", "status"
  activeConns openmetrics.GaugeFamily,      // no tags
) *grpc.Server {
  server := grpc.NewServer(
    grpc.WithStats(omgrpc.InstrumentCallCount(callCount)),
    grpc.WithStats(omgrpc.InstrumentCallDuration(callDuration)),
    grpc.WithStats(omgrpc.InstrumentActiveConns(activeConns)),
  )
  yourproto.RegisterFooServer(server, fooServer)
  return server
}

func initGRPCClient(
  ctx context.Context,
  target string,

  callCount openmetrics.CounterFamily,      // with tags: "method", "status"
  callDuration openmetrics.HistogramFamily, // with tags: "method", "status"
  activeConns openmetrics.GaugeFamily,      // no tags
) (*grpc.Conn, error) {
  return grpc.DialContext(
    ctx,
    target,

    grpc.WithStatsHandler(omgrpc.InstrumentCallCount(callCount)),
    grpc.WithStatsHandler(omgrpc.InstrumentCallDuration(callDuration)),
    grpc.WithStatsHandler(omgrpc.InstrumentActiveConns(activeConns)),
  )
}
```
