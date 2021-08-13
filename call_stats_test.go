package omgrpc_test

import (
	"context"
	"time"

	"github.com/bsm/omgrpc/internal/testpb"
	"google.golang.org/grpc"

	. "github.com/bsm/omgrpc"

	. "github.com/bsm/ginkgo"
	. "github.com/bsm/gomega"
)

var _ = Describe("CallStatsHandler", func() {
	var (
		ctx = context.Background()

		subject     CallStatsHandler
		callStats   []CallStats
		client      testpb.TestClient
		clientClose func()
		teardown    func()
	)

	BeforeEach(func() {
		callStats = callStats[:0]

		subject = CallStatsHandler(func(call *CallStats) {
			callStats = append(callStats, *call)
		})

		client, clientClose, teardown = initClientServerSystem(
			[]grpc.DialOption{
				grpc.WithStatsHandler(subject),
			},
			[]grpc.ServerOption{
				grpc.StatsHandler(subject),
			},
		)
	})

	AfterEach(func() {
		teardown()
	})

	It("tracks call stats", func() {
		var (
			msg *testpb.Message
			err error
		)

		msg, err = client.Unary(ctx, &testpb.Message{Payload: "1"})
		Expect(err).NotTo(HaveOccurred())
		Expect(msg.Payload).To(Equal("Unary: 1")) // payload assertions are to illustrate/justify transfer size checks

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

		clientClose() // initiate Stream call to be closed by closing Client

		var s CallStats

		// server Unary:
		s = callStats[0]
		Expect(s.Client).To(BeFalse())
		Expect(s.FullMethodName).To(Equal("/com.blacksquaremedia.omgrpc.internal.testpb.Test/Unary"))
		Expect(s.IsClientStream).To(BeFalse())
		Expect(s.IsServerStream).To(BeFalse())
		Expect(s.BytesRecv).To(Equal(94))
		Expect(s.BytesSent).To(Equal(15))
		Expect(s.Error).To(BeNil())

		// assert once that these fields are populated:
		Expect(s.BeginTime).To(BeTemporally("~", time.Now(), time.Second))
		Expect(s.EndTime).To(BeTemporally("~", time.Now(), time.Second))
		Expect(s.EndTime).To(BeTemporally(">", s.BeginTime))
		Expect(s.InHeader).NotTo(BeNil())   // just check that this is set; no assertions for InTrailer as they're rare and not used in this test
		Expect(s.OutHeader).NotTo(BeNil())  // just check that this is set; no assertions for InTrailer as they're rare and not used in this test
		Expect(s.LocalAddr).NotTo(BeNil())  // just check that this is set
		Expect(s.RemoteAddr).NotTo(BeNil()) // just check that this is set

		// client Unary:
		s = callStats[1]
		Expect(s.Client).To(BeTrue())
		Expect(s.FullMethodName).To(Equal("/com.blacksquaremedia.omgrpc.internal.testpb.Test/Unary"))
		Expect(s.IsClientStream).To(BeFalse())
		Expect(s.IsServerStream).To(BeFalse())
		Expect(s.BytesRecv).To(Equal(53))
		Expect(s.BytesSent).To(Equal(8))
		Expect(s.Error).To(BeNil())

		// server Stream:
		s = callStats[2]
		Expect(s.Client).To(BeFalse())
		Expect(s.FullMethodName).To(Equal("/com.blacksquaremedia.omgrpc.internal.testpb.Test/Stream"))
		Expect(s.IsClientStream).To(BeTrue())
		Expect(s.IsServerStream).To(BeTrue())
		Expect(s.BytesRecv).To(Equal(64))
		Expect(s.BytesSent).To(Equal(32))
		Expect(s.Error).To(BeNil())

		// client Stream:
		s = callStats[3]
		Expect(s.Client).To(BeTrue())
		Expect(s.FullMethodName).To(Equal("/com.blacksquaremedia.omgrpc.internal.testpb.Test/Stream"))
		Expect(s.IsClientStream).To(BeTrue())
		Expect(s.IsServerStream).To(BeTrue())
		Expect(s.BytesRecv).To(Equal(36))
		Expect(s.BytesSent).To(Equal(16))
		// basically, clientClose affects Client first, and only then Server, so we get this:
		Expect(s.Error).To(MatchError(ContainSubstring("grpc: the client connection is closing")))
	})
})
