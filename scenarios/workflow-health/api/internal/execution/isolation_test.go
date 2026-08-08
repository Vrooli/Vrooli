package execution

import (
	"context"
	"errors"
	"testing"
	"time"

	"connectrpc.com/connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
)

type heartbeatRoutingClient struct {
	heartbeatErr error
}

func (c heartbeatRoutingClient) InstallTestPool(context.Context, *connect.Request[routingv1.InstallTestPoolRequest]) (*connect.Response[routingv1.InstallTestPoolResponse], error) {
	return connect.NewResponse(&routingv1.InstallTestPoolResponse{}), nil
}

func (c heartbeatRoutingClient) ClearTestPool(context.Context, *connect.Request[routingv1.ClearTestPoolRequest]) (*connect.Response[routingv1.ClearTestPoolResponse], error) {
	return connect.NewResponse(&routingv1.ClearTestPoolResponse{}), nil
}

func (c heartbeatRoutingClient) HeartbeatTestPool(context.Context, *connect.Request[routingv1.HeartbeatTestPoolRequest]) (*connect.Response[routingv1.HeartbeatTestPoolResponse], error) {
	if c.heartbeatErr != nil {
		return nil, c.heartbeatErr
	}
	return connect.NewResponse(&routingv1.HeartbeatTestPoolResponse{}), nil
}

func TestRoutingLeaseHeartbeatFailureSignalsHealthAndEvidence(t *testing.T) {
	heartbeatErr := errors.New("routing service unavailable")
	lease := &routingLease{
		client:     heartbeatRoutingClient{heartbeatErr: heartbeatErr},
		leaseID:    "lease-1",
		healthDone: make(chan struct{}),
		evidence:   IsolationEvidence{Installed: true, LeaseID: "lease-1"},
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go lease.heartbeat(ctx, 3*time.Millisecond)
	select {
	case <-lease.Done():
	case <-time.After(250 * time.Millisecond):
		t.Fatal("lease heartbeat failure did not signal health loss")
	}

	if !errors.Is(lease.Err(), heartbeatErr) {
		t.Fatalf("health error = %v, want %v", lease.Err(), heartbeatErr)
	}
	if got := lease.Evidence().HeartbeatError; got != heartbeatErr.Error() {
		t.Fatalf("heartbeat evidence = %q, want %q", got, heartbeatErr.Error())
	}
}
