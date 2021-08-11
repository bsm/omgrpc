package omgrpc_test

import (
	"context"
	"net"
	"testing"
	"time"

	. "github.com/bsm/ginkgo"
	. "github.com/bsm/gomega"
	"github.com/bsm/omgrpc"
	"github.com/bsm/omgrpc/internal/testpb"
	"google.golang.org/grpc"
	"google.golang.org/grpc/test/bufconn"
)

func TestSuite(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "omgrpc")
}

var _ = Describe("StatsHandler", func() {
	var subject *omgrpc.StatsHandler
	var client testpb.TestClient
	var listener *bufconn.Listener
	var grpcServer *grpc.Server
	var grpcClient *grpc.ClientConn

	ctx := context.Background()

	BeforeEach(func() {
		subject = new(omgrpc.StatsHandler)
		service := &testpb.TestServerImpl{}

		grpcServer = grpc.NewServer(grpc.StatsHandler(subject))
		testpb.RegisterTestServer(grpcServer, service)

		listener = bufconn.Listen(10 * 1024 * 1024 /* 10 MB buf */)
		go func() {
			defer GinkgoRecover()
			_ = grpcServer.Serve(listener)
		}()
		time.Sleep(100 * time.Millisecond) // give it a bit of time to start serving in background

		dialer := func(context.Context, string) (net.Conn, error) { return listener.Dial() }

		var err error
		grpcClient, err = grpc.Dial(
			"bufconn",
			grpc.WithContextDialer(dialer),
			grpc.WithInsecure(),
			// grpc.WithStatsHandler(subject) // TODO: implement/test client stats handler as well!
		)
		Expect(err).NotTo(HaveOccurred())

		client = testpb.NewTestClient(grpcClient)
	})

	AfterEach(func() {
		_ = grpcClient.Close()
		grpcServer.Stop()
		listener.Close()
	})

	It("prints debug info temporarily", func() {
		var (
			msg *testpb.Message
			err error
		)

		msg, err = client.Unary(ctx, &testpb.Message{Payload: "1"})
		Expect(err).NotTo(HaveOccurred())
		Expect(msg.Payload).To(Equal("Unary: 1"))

		stream, err := client.Stream(ctx)
		Expect(err).NotTo(HaveOccurred())
		defer stream.CloseSend()

		Expect(stream.SendMsg(&testpb.Message{Payload: "2"})).To(Succeed())
		Expect(stream.SendMsg(&testpb.Message{Payload: "3"})).To(Succeed())
		Expect(stream.CloseSend()).To(Succeed())

		msg, err = stream.Recv()
		Expect(err).NotTo(HaveOccurred())
		Expect(msg.Payload).To(Equal("Stream: 2"))

		msg, err = stream.Recv()
		Expect(err).NotTo(HaveOccurred())
		Expect(msg.Payload).To(Equal("Stream: 3"))

		// TODO: assert metrics

		/*
			- connection: +1
			- transfer size: method=/com.blacksquaremedia.omgrpc.internal.testpb.Test/Unary, wire_length=86, when=*stats.InHeader
			- transfer size: method=/com.blacksquaremedia.omgrpc.internal.testpb.Test/Unary, wire_length=8, when=*stats.InPayload
			- transfer size: method=/com.blacksquaremedia.omgrpc.internal.testpb.Test/Unary, wire_length=15, when=*stats.OutPayload
			- request timing: method=/com.blacksquaremedia.omgrpc.internal.testpb.Test/Unary, elapsed=0.000074, status=OK
			- transfer size: method=/com.blacksquaremedia.omgrpc.internal.testpb.Test/Stream, wire_length=48, when=*stats.InHeader
			- transfer size: method=/com.blacksquaremedia.omgrpc.internal.testpb.Test/Stream, wire_length=8, when=*stats.InPayload
			- transfer size: method=/com.blacksquaremedia.omgrpc.internal.testpb.Test/Stream, wire_length=16, when=*stats.OutPayload
			- transfer size: method=/com.blacksquaremedia.omgrpc.internal.testpb.Test/Stream, wire_length=8, when=*stats.InPayload
			- transfer size: method=/com.blacksquaremedia.omgrpc.internal.testpb.Test/Stream, wire_length=16, when=*stats.OutPayload
			- request timing: method=/com.blacksquaremedia.omgrpc.internal.testpb.Test/Stream, elapsed=0.000114, status=OK
			- connection: -1
		*/
	})
})
