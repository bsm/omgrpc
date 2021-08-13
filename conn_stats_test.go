package omgrpc_test

import (
	"context"

	"github.com/bsm/omgrpc/internal/testpb"
	"google.golang.org/grpc"

	. "github.com/bsm/omgrpc"

	. "github.com/bsm/ginkgo"
	. "github.com/bsm/gomega"
)

var _ = Describe("ConnStatsHandler", func() {
	var (
		ctx = context.Background()

		connStats   []ConnStats
		client      testpb.TestClient
		clientClose func()
		teardown    func()
	)

	subject := ConnStatsHandler(func(conn *ConnStats) {
		connStats = append(connStats, *conn)
	})

	BeforeEach(func() {
		connStats = connStats[:0]

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
		_, _ = client.Unary(ctx, &testpb.Message{Payload: "trigger connection"}) // grpc.Dial can be lazy and can connect on first gRPC call
		clientClose()                                                            // and disconnect right away

		Expect(connStats).To(HaveLen(4))

		var s ConnStats

		// Client connect:
		s = connStats[0]
		Expect(s.Client).To(BeTrue())
		Expect(s.Connected).To(BeTrue())

		// assert once that these fields are populated:
		Expect(s.LocalAddr).NotTo(BeNil())
		Expect(s.RemoteAddr).NotTo(BeNil())

		// Server connect:
		s = connStats[1]
		Expect(s.Client).To(BeFalse())
		Expect(s.Connected).To(BeTrue())

		// Server disconnect:
		s = connStats[2]
		Expect(s.Client).To(BeFalse())
		Expect(s.Connected).To(BeFalse())

		// Client disconnect:
		s = connStats[3]
		Expect(s.Client).To(BeTrue())
		Expect(s.Connected).To(BeFalse())
	})
})
