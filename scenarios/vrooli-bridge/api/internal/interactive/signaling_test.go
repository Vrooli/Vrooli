package interactive

import (
	"context"
	"testing"
	"time"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	interactivev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/interactive"
)

type signalPusherFunc func(context.Context, string, *channelv1.InteractiveSignal) error

func (f signalPusherFunc) PushInteractiveSignal(ctx context.Context, node string, signal *channelv1.InteractiveSignal) error {
	return f(ctx, node, signal)
}

func TestSignalBrokerCorrelatesOpaqueCompanionResponse(t *testing.T) {
	grant := &interactivev1.ChannelGrant{ChannelId: "channel-1", NodeId: "node-1", LeaseEpoch: 3, Routes: []*interactivev1.RouteCandidate{{Kind: interactivev1.RouteKind_ROUTE_KIND_TURN, Url: "turn:relay.example", Username: "user", Credential: "credential"}}}
	request := &interactivev1.SignalRequest{Grant: grant, Kind: interactivev1.SignalKind_SIGNAL_KIND_OFFER, RequestId: "request-1", Generation: "generation-1", Payload: []byte("opaque-offer")}
	var broker *SignalBroker
	broker = NewSignalBroker(signalPusherFunc(func(_ context.Context, node string, signal *channelv1.InteractiveSignal) error {
		if node != "node-1" || string(signal.GetPayload()) != "opaque-offer" || signal.GetKind() != uint32(request.GetKind()) || len(signal.GetRoutes()) != 1 || signal.GetRoutes()[0].GetCredential() != "credential" {
			t.Fatalf("unexpected pushed signal: %v", signal)
		}
		go func() {
			broker.Deliver(&channelv1.InteractiveSignalResponse{ChannelId: "channel-1", NodeId: "node-1", LeaseEpoch: 3, RequestId: "request-1", Kind: uint32(interactivev1.SignalKind_SIGNAL_KIND_ANSWER), Generation: "generation-1", Accepted: true, Payload: []byte("opaque-answer")})
		}()
		return nil
	}))
	response, err := broker.Submit(context.Background(), request)
	if err != nil || !response.GetAccepted() || string(response.GetPayload()) != "opaque-answer" || response.GetRequestId() != "request-1" {
		t.Fatalf("response=%v err=%v", response, err)
	}
}

func TestSignalBrokerTimesOutAndRejectsLateResponse(t *testing.T) {
	broker := NewSignalBroker(signalPusherFunc(func(context.Context, string, *channelv1.InteractiveSignal) error { return nil }))
	broker.timeout = 10 * time.Millisecond
	request := &interactivev1.SignalRequest{Grant: &interactivev1.ChannelGrant{ChannelId: "channel-1", NodeId: "node-1", LeaseEpoch: 1}, RequestId: "request-1", Payload: []byte("offer")}
	if _, err := broker.Submit(context.Background(), request); err != ErrSignalResponseTimeout {
		t.Fatalf("timeout error=%v", err)
	}
	if broker.Deliver(&channelv1.InteractiveSignalResponse{ChannelId: "channel-1", NodeId: "node-1", LeaseEpoch: 1, RequestId: "request-1"}) {
		t.Fatal("late response should not be delivered")
	}
}

func TestSignalBrokerRejectsStaleGenerationAndWrongResponseKind(t *testing.T) {
	grant := &interactivev1.ChannelGrant{ChannelId: "channel-1", NodeId: "node-1", LeaseEpoch: 1}
	cases := []struct {
		name       string
		generation string
		kind       interactivev1.SignalKind
	}{
		{name: "stale generation", generation: "old-generation", kind: interactivev1.SignalKind_SIGNAL_KIND_ANSWER},
		{name: "wrong response kind", generation: "generation-1", kind: interactivev1.SignalKind_SIGNAL_KIND_ICE},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			var broker *SignalBroker
			broker = NewSignalBroker(signalPusherFunc(func(_ context.Context, _ string, _ *channelv1.InteractiveSignal) error {
				go func() {
					broker.Deliver(&channelv1.InteractiveSignalResponse{ChannelId: "channel-1", NodeId: "node-1", LeaseEpoch: 1, Kind: uint32(tt.kind), Generation: tt.generation, RequestId: "request-1", Accepted: true})
				}()
				return nil
			}))
			request := &interactivev1.SignalRequest{Grant: grant, Kind: interactivev1.SignalKind_SIGNAL_KIND_OFFER, RequestId: "request-1", Generation: "generation-1", Payload: []byte("offer")}
			if _, err := broker.Submit(context.Background(), request); err != ErrSignalResponseMismatch {
				t.Fatalf("error=%v, want=%v", err, ErrSignalResponseMismatch)
			}
		})
	}
}
