package capabilityapp

import (
	"context"
	"testing"
	"time"

	"github.com/vrooli/api-core/demand"
)

type capabilityDemandStub struct {
	acquired []demand.AcquireRequest
	released []string
}

func (s *capabilityDemandStub) Acquire(_ context.Context, req demand.AcquireRequest) (demand.Lease, error) {
	s.acquired = append(s.acquired, req)
	return demand.Lease{LeaseID: req.LeaseID}, nil
}

func (s *capabilityDemandStub) Renew(context.Context, string, time.Duration) (demand.Lease, error) {
	return demand.Lease{}, nil
}

func (s *capabilityDemandStub) Release(_ context.Context, leaseID, _ string) (demand.Lease, error) {
	s.released = append(s.released, leaseID)
	return demand.Lease{LeaseID: leaseID}, nil
}

func TestCapabilityClientOwnsAndReleasesDemandLease(t *testing.T) {
	oldResolver, oldDemand := capabilityResolveURL, capabilityDemandClient
	t.Cleanup(func() {
		capabilityResolveURL, capabilityDemandClient = oldResolver, oldDemand
	})
	capabilityResolveURL = func(context.Context, string) (string, error) { return "http://127.0.0.1:12345", nil }
	stub := &capabilityDemandStub{}
	capabilityDemandClient = stub

	_, release, err := capabilityClientWithDemand(context.Background(), "ledger")
	if err != nil {
		t.Fatalf("capabilityClientWithDemand: %v", err)
	}
	if release == nil {
		t.Fatal("expected release function")
	}
	if len(stub.acquired) != 1 {
		t.Fatalf("acquire count = %d", len(stub.acquired))
	}
	req := stub.acquired[0]
	if req.Kind != demand.KindDependency || req.ConsumerID != "control-plane:capability-client" {
		t.Fatalf("unexpected demand request: %+v", req)
	}
	release()
	if len(stub.released) != 1 || stub.released[0] != req.LeaseID {
		t.Fatalf("release lifecycle = %+v", stub.released)
	}
}
