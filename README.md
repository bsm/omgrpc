# omgrpc - [OpenMetrics](https://github.com/bsm/openmetrics) [gRPC](https://github.com/grpc/grpc-go) middleware

[![Lint](https://github.com/bsm/omgrpc/actions/workflows/lint.yml/badge.svg)](https://github.com/bsm/omgrpc/actions/workflows/lint.yml)
[![Test](https://github.com/bsm/omgrpc/actions/workflows/test.yml/badge.svg)](https://github.com/bsm/omgrpc/actions/workflows/test.yml)

## Quickstart

```go
import (
  "github.com/bsm/omgrpc"
  "google.golang.org/grpc"
)

func initGRPCServer(fooServer yourproto.FooServer) *grpc.Server {
  server := grpc.NewServer(
    grpc.StatsHandler(omgrpc.DefaultCallStatsHandler(openmetrics.DefaultRegistry())), // to track grpc_call count + timings
    grpc.StatsHandler(omgrpc.DefaultConnStatsHandler(openmetrics.DefaultRegistry())), // to track grpc_active_conns
  )
  yourproto.RegisterFooServer(server, fooServer)
  return server
}

func initGRPCClient(ctx context.Context, target string) (*grpc.Conn, error) {
  conn, err := grpc.DialContext(
    ctx,
    target,

    // same registry/same handler can be used for client connection
    // as long as you run them in different processes
    // so metrics do not overlap:
    grpc.StatsHandler(omgrpc.DefaultCallStatsHandler(openmetrics.DefaultRegistry())), // to track grpc_call count + timings
    grpc.StatsHandler(omgrpc.DefaultConnStatsHandler(openmetrics.DefaultRegistry())), // to track grpc_active_conns
  )
  return conn
}
```
