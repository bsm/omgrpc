package omgrpc_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/bsm/omgrpc/internal/testpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"

	. "github.com/bsm/ginkgo"
	. "github.com/bsm/gomega"
)

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "omgrpc")
}

func initClientServerSystem(
	clientOptions []grpc.DialOption,
	serverOptions []grpc.ServerOption,
) (
	testClient testpb.TestClient,
	clientClose func(),
	teardown func(),
) {
	const serverDelay = 100 * time.Millisecond // allow server to lag behind a bit - to start in background, to process data etc

	server := grpc.NewServer(serverOptions...)
	testpb.RegisterTestServer(server, new(testpb.TestServerImpl))

	listener := bufconn.Listen(10 * 1024 * 1024 /* 10 MB buf */)
	go func() {
		defer GinkgoRecover()
		_ = server.Serve(listener)
	}()
	time.Sleep(serverDelay) // give it a bit of time to start serving in background

	dialer := func(context.Context, string) (net.Conn, error) {
		return listener.Dial()
	}

	client, err := grpc.Dial(
		"bufconn",
		append(
			[]grpc.DialOption{
				grpc.WithContextDialer(dialer),
				grpc.WithInsecure(),
			},
			clientOptions..., // overrides defaults if needed
		)...,
	)
	Expect(err).NotTo(HaveOccurred())

	testClient = testpb.NewTestClient(client)
	clientClose = func() {
		client.Close()
		time.Sleep(serverDelay) // and let server "digest" / submit stats for client disconnect
	}
	teardown = func() {
		_ = client.Close()
		server.Stop()
		_ = listener.Close()
	}

	return testClient, clientClose, teardown
}
