package omgrpc_test

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/bsm/omgrpc/internal/testpb"
	"google.golang.org/grpc"

	. "github.com/bsm/omgrpc"

	. "github.com/bsm/ginkgo"
	. "github.com/bsm/gomega"
)

var _ = Describe("CallStatsHandler", func() {
	ctx := context.Background()
	out := bytes.NewBuffer(nil)
	outJSON := json.NewEncoder(out)
	subject := CallStatsHandler(func(call *CallStats) {
		// simplify assertions:
		call.BeginTime = time.Date(2021, time.August, 12, 19, 8, 52, 0, time.UTC)
		call.EndTime = time.Date(2021, time.August, 12, 19, 8, 53, 0, time.UTC)
		if _, ok := call.InHeader["user-agent"]; ok {
			call.InHeader["user-agent"] = []string{"IS_SET"} // usually contains grpc version
		}
		if _, ok := call.OutHeader["user-agent"]; ok {
			call.OutHeader["user-agent"] = []string{"IS_SET"} // usually contains grpc version
		}
		Expect(outJSON.Encode(call)).To(Succeed())
	})

	var (
		client      testpb.TestClient
		clientClose func()
		teardown    func()
	)

	BeforeEach(func() {
		out.Reset()

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

		clientClose()

		jsonLines := strings.Split(strings.TrimSpace(out.String()), "\n")
		Expect(jsonLines).To(ConsistOf(
			MatchJSON(`{
				"FullMethodName": "/com.blacksquaremedia.omgrpc.internal.testpb.Test/Unary",
				"FailFast": false,
				"Client": false,
				"BeginTime": "2021-08-12T19:08:52Z",
				"IsClientStream": false,
				"IsServerStream": false,
				"InHeader": {
					":authority": [
						"bufconn"
					],
					"content-type": [
						"application/grpc"
					],
					"user-agent": [
						"IS_SET"
					]
				},
				"InTrailer": null,
				"OutHeader": {},
				"OutTrailer": {},
				"EndTime": "2021-08-12T19:08:53Z",
				"Error": null,
				"RemoteAddr": {},
				"LocalAddr": {},
				"InWireLength": 109,
				"OutWireLength": 15
			}`),
			MatchJSON(`{
				"FullMethodName": "/com.blacksquaremedia.omgrpc.internal.testpb.Test/Unary",
				"FailFast": true,
				"Client": true,
				"BeginTime": "2021-08-12T19:08:52Z",
				"IsClientStream": false,
				"IsServerStream": false,
				"InHeader": {
					"content-type": [
						"application/grpc"
					]
				},
				"InTrailer": {},
				"OutHeader": {
					"user-agent": [
						"IS_SET"
					]
				},
				"OutTrailer": null,
				"EndTime": "2021-08-12T19:08:53Z",
				"Error": null,
				"RemoteAddr": {},
				"LocalAddr": {},
				"InWireLength": 61,
				"OutWireLength": 8
			}`),
			MatchJSON(`{
				"FullMethodName": "/com.blacksquaremedia.omgrpc.internal.testpb.Test/Stream",
				"FailFast": false,
				"Client": false,
				"BeginTime": "2021-08-12T19:08:52Z",
				"IsClientStream": true,
				"IsServerStream": true,
				"InHeader": {
					":authority": [
						"bufconn"
					],
					"content-type": [
						"application/grpc"
					],
					"user-agent": [
						"IS_SET"
					]
				},
				"InTrailer": null,
				"OutHeader": {},
				"OutTrailer": {},
				"EndTime": "2021-08-12T19:08:53Z",
				"Error": null,
				"RemoteAddr": {},
				"LocalAddr": {},
				"InWireLength": 96,
				"OutWireLength": 32
			}`),

			// Note "Error" here - it's context.Cancelled (because of server closed conn)
			MatchJSON(`{
				"FullMethodName": "/com.blacksquaremedia.omgrpc.internal.testpb.Test/Stream",
				"FailFast": true,
				"Client": true,
				"BeginTime": "2021-08-12T19:08:52Z",
				"IsClientStream": true,
				"IsServerStream": true,
				"InHeader": {
					"content-type": [
						"application/grpc"
					]
				},
				"InTrailer": {},
				"OutHeader": {
					"user-agent": [
						"IS_SET"
					]
				},
				"OutTrailer": null,
				"EndTime": "2021-08-12T19:08:53Z",
				"Error": {},
				"RemoteAddr": {},
				"LocalAddr": {},
				"InWireLength": 52,
				"OutWireLength": 16
			}`),
		))
	})

	PIt("registers errors")

})
