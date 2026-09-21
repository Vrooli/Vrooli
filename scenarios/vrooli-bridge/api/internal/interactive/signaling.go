package interactive

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	channelv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/channel"
	interactivev1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/interactive"
)

var (
	ErrSignalTransportUnavailable = errors.New("interactive signal transport is unavailable")
	ErrSignalResponseTimeout      = errors.New("interactive signal response timed out")
	ErrSignalResponseMismatch     = errors.New("interactive signal response did not match request")
)

// SignalPusher is the signed node-channel boundary. It carries opaque signal
// bytes; the Bridge API never parses SDP, ICE, or media payloads.
type SignalPusher interface {
	PushInteractiveSignal(context.Context, string, *channelv1.InteractiveSignal) error
}

type RevokePusher interface {
	PushInteractiveRevoke(context.Context, *interactivev1.ChannelGrant, string) error
}

type SignalBroker struct {
	mu      sync.Mutex
	push    SignalPusher
	waiters map[string]chan *channelv1.InteractiveSignalResponse
	timeout time.Duration
}

func NewSignalBroker(push SignalPusher) *SignalBroker {
	return &SignalBroker{push: push, waiters: make(map[string]chan *channelv1.InteractiveSignalResponse), timeout: 15 * time.Second}
}

func (b *SignalBroker) Submit(ctx context.Context, req *interactivev1.SignalRequest) (*interactivev1.SignalResponse, error) {
	if b == nil || b.push == nil {
		return nil, ErrSignalTransportUnavailable
	}
	if req == nil || req.GetGrant() == nil || req.GetGrant().GetChannelId() == "" || req.GetRequestId() == "" {
		return nil, ErrSignalResponseMismatch
	}
	key := req.GetGrant().GetChannelId() + ":" + req.GetRequestId()
	waiter := make(chan *channelv1.InteractiveSignalResponse, 1)
	b.mu.Lock()
	if _, exists := b.waiters[key]; exists {
		b.mu.Unlock()
		return nil, fmt.Errorf("%w: duplicate request", ErrSignalResponseMismatch)
	}
	b.waiters[key] = waiter
	b.mu.Unlock()
	defer func() {
		b.mu.Lock()
		delete(b.waiters, key)
		b.mu.Unlock()
	}()
	grant := req.GetGrant()
	if err := b.push.PushInteractiveSignal(ctx, grant.GetNodeId(), &channelv1.InteractiveSignal{
		ChannelId: grant.GetChannelId(), NodeId: grant.GetNodeId(), SessionId: grant.GetSessionId(), LeaseId: grant.GetLeaseId(), LeaseEpoch: grant.GetLeaseEpoch(), Kind: uint32(req.GetKind()), Generation: req.GetGeneration(), Payload: append([]byte(nil), req.GetPayload()...), RequestId: req.GetRequestId(), Routes: cloneRoutes(grant.GetRoutes()),
	}); err != nil {
		return nil, err
	}
	timer := time.NewTimer(b.timeout)
	defer timer.Stop()
	select {
	case response := <-waiter:
		expectedKind := uint32(req.GetKind())
		if req.GetKind() == interactivev1.SignalKind_SIGNAL_KIND_OFFER {
			expectedKind = uint32(interactivev1.SignalKind_SIGNAL_KIND_ANSWER)
		}
		if response == nil || response.GetChannelId() != grant.GetChannelId() || response.GetNodeId() != grant.GetNodeId() || response.GetRequestId() != req.GetRequestId() || response.GetLeaseEpoch() != grant.GetLeaseEpoch() || response.GetGeneration() != req.GetGeneration() || response.GetKind() != expectedKind {
			return nil, ErrSignalResponseMismatch
		}
		return &interactivev1.SignalResponse{Accepted: response.GetAccepted(), Generation: response.GetGeneration(), ReasonCode: response.GetReasonCode(), Payload: append([]byte(nil), response.GetPayload()...), RequestId: response.GetRequestId()}, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, ErrSignalResponseTimeout
	}
}

func (b *SignalBroker) Deliver(response *channelv1.InteractiveSignalResponse) bool {
	if b == nil || response == nil || response.GetChannelId() == "" || response.GetRequestId() == "" {
		return false
	}
	key := response.GetChannelId() + ":" + response.GetRequestId()
	b.mu.Lock()
	waiter := b.waiters[key]
	b.mu.Unlock()
	if waiter == nil {
		return false
	}
	select {
	case waiter <- response:
		return true
	default:
		return false
	}
}
