package omgrpc_test

import (
	"context"
	"encoding/json"
	"net"
	"testing"
	"time"

	"github.com/bsm/omgrpc"
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

func initClientServerSystem(clientOptions []grpc.DialOption, serverOptions []grpc.ServerOption) (testpb.TestClient, func()) {
	server := grpc.NewServer(serverOptions...)
	testpb.RegisterTestServer(server, new(testpb.TestServerImpl))

	listener := bufconn.Listen(10 * 1024 * 1024 /* 10 MB buf */)
	go func() {
		defer GinkgoRecover()
		_ = server.Serve(listener)
	}()
	time.Sleep(100 * time.Millisecond) // give it a bit of time to start serving in background

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

	return testpb.NewTestClient(client), func() {
		_ = client.Close()
		server.Stop()
		_ = listener.Close()
	}
}

// TODO: kill
var _ = Describe("StatsHandler", func() {
	var subject *omgrpc.StatsHandler
	var calls [][]interface{}
	var client testpb.TestClient
	var teardown func()

	ctx := context.Background()

	BeforeEach(func() {
		calls = calls[:0]
		subject = &omgrpc.StatsHandler{
			OnData: func(fullMethod string, phase omgrpc.DataPhase, dir omgrpc.DataDirection, size int) {
				calls = append(calls, []interface{}{"OnData", fullMethod, phase, dir, size})
			},
			OnCall: func(fullMethod string, err error, elapsed time.Duration) {
				if elapsed != 0 {
					elapsed = time.Second // to simplify assertions
				}
				calls = append(calls, []interface{}{"OnCall", fullMethod, err, elapsed.String()})
			},
			OnConn: func(increment int) {
				calls = append(calls, []interface{}{"OnConn", increment})
			},
		}

		client, teardown = initClientServerSystem(
			[]grpc.DialOption(nil),
			[]grpc.ServerOption{
				grpc.StatsHandler(subject),
			},
		)
	})

	AfterEach(func() {
		teardown()
	})

	It("runs callbacks", func() {
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

		// to simplify and make it more readable:
		callsJSON, err := json.MarshalIndent(calls, "", "\t")
		Expect(err).NotTo(HaveOccurred())
		Expect(callsJSON).To(MatchJSON(`[
			["OnConn", 1],

			["OnData", "/com.blacksquaremedia.omgrpc.internal.testpb.Test/Unary", "header",  "in",  86],
			["OnData", "/com.blacksquaremedia.omgrpc.internal.testpb.Test/Unary", "payload", "in",   8],
			["OnData", "/com.blacksquaremedia.omgrpc.internal.testpb.Test/Unary", "payload", "out", 15],
			["OnCall", "/com.blacksquaremedia.omgrpc.internal.testpb.Test/Unary", null, "1s"],

			["OnData", "/com.blacksquaremedia.omgrpc.internal.testpb.Test/Stream", "header",  "in",  48],
			["OnData", "/com.blacksquaremedia.omgrpc.internal.testpb.Test/Stream", "payload", "in",   8],
			["OnData", "/com.blacksquaremedia.omgrpc.internal.testpb.Test/Stream", "payload", "out", 16],
			["OnData", "/com.blacksquaremedia.omgrpc.internal.testpb.Test/Stream", "payload", "in",   8],
			["OnData", "/com.blacksquaremedia.omgrpc.internal.testpb.Test/Stream", "payload", "out", 16],
			["OnCall", "/com.blacksquaremedia.omgrpc.internal.testpb.Test/Stream", null, "1s"]
		]`))
	})
})
