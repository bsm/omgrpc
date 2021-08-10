# omgrpc - [OpenMetrics](https://github.com/bsm/openmetrics) [gRPC](https://github.com/grpc/grpc-go) middleware

## Usage

```go
import (
  "github.com/bsm/omgrpc"
  "github.com/bsm/openmetrics"
  "github.com/grpc-ecosystem/go-grpc-middleware/recovery" // optional, omgrpc does not handle panics
  "google.golang.org/grpc"
)

func buildServer() *grpc.Server {
  reg := openmetrics.DefaultRegistry             // or other registry impl
  metrics := omgrpc.NewDefaultServerMetrics(reg) // provides "grpc_requests" metrics (counter + histogram for request duration) with "full_method" and "status" labels

  server, err := grpc.NewServer(
    grpc.ChainStreamInterceptor(
      omgrpc.NewStreamServerInterceptor(metrics),
      recovery.UnaryServerInterceptor(), // omgrpc does not recover from panics, use own or third-party recoverer
      // your very own interceptors
    ),
    grpc.ChainUnaryInterceptor(
      omgrpc.NewStreamUnaryInterceptor(metrics),
      recovery.StreamServerInterceptor(), // omgrpc does not recover from panics, use own or third-party recoverer
      // your very own interceptors
    ),
    // any other options
  )
  if err != nil {
    return nil, err
  }

  // register your services:
  server.RegisterService(...)

  return server, nil
}
```
