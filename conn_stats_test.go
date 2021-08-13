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

		subject         ConnStatsHandler
		clientConnStats []ConnStats
		serverConnStats []ConnStats
		client          testpb.TestClient
		clientClose     func()
		teardown        func()
	)

	BeforeEach(func() {
		clientConnStats = clientConnStats[:0]
		serverConnStats = serverConnStats[:0]

		subject = ConnStatsHandler(func(conn *ConnStats) {
			if conn.Client {
				clientConnStats = append(clientConnStats, *conn)
			} else {
				serverConnStats = append(serverConnStats, *conn)
			}
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
		var err error
		_, err = client.Unary(ctx, &testpb.Message{Payload: "1"})
		Expect(err).NotTo(HaveOccurred())
		_, err = client.Unary(ctx, &testpb.Message{Payload: "2"})
		Expect(err).NotTo(HaveOccurred())

		clientClose() // and disconnect right away

		Expect(clientConnStats).To(HaveLen(2))
		Expect(serverConnStats).To(HaveLen(2))

		var s ConnStats

		// Client connect:
		s = clientConnStats[0]
		Expect(s.Client).To(BeTrue())
		Expect(s.Status).To(Equal(Connected))
		Expect(s.BytesRecv).To(BeZero()) // supported only server-side
		Expect(s.BytesSent).To(BeZero()) // supported only server-side

		// assert once that these fields are populated:
		Expect(s.LocalAddr).NotTo(BeNil())
		Expect(s.RemoteAddr).NotTo(BeNil())

		// Client disconnect:
		s = clientConnStats[1]
		Expect(s.Client).To(BeTrue())
		Expect(s.Status).To(Equal(Disconnected))
		Expect(s.BytesRecv).To(BeZero()) // supported only server-side
		Expect(s.BytesSent).To(BeZero()) // supported only server-side

		// Server connect:
		s = serverConnStats[0]
		Expect(s.Client).To(BeFalse())
		Expect(s.Status).To(Equal(Connected))
		Expect(s.BytesRecv).To(BeZero()) // obviously, basically just checking that it's cleaned on pooling
		Expect(s.BytesSent).To(BeZero()) // obviously, basically just checking that it's cleaned on pooling

		// Server disconnect:
		s = serverConnStats[1]
		Expect(s.Client).To(BeFalse())
		Expect(s.Status).To(Equal(Disconnected))
		Expect(s.BytesRecv).To(Equal(109))
		Expect(s.BytesSent).To(Equal(30))
	})
})
