package plans

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/vrooli/api-core/demand"
)

type planDemandStub struct {
	acquired []demand.AcquireRequest
	released []string
}

func (s *planDemandStub) Acquire(_ context.Context, req demand.AcquireRequest) (demand.Lease, error) {
	s.acquired = append(s.acquired, req)
	return demand.Lease{LeaseID: req.LeaseID}, nil
}

func (s *planDemandStub) Renew(context.Context, string, time.Duration) (demand.Lease, error) {
	return demand.Lease{}, nil
}

func (s *planDemandStub) Release(_ context.Context, leaseID, _ string) (demand.Lease, error) {
	s.released = append(s.released, leaseID)
	return demand.Lease{LeaseID: leaseID}, nil
}

func TestPlanManagerRequestOwnsAndReleasesDemandLease(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vrooli.plan_manager.v1.plans.PlansService/ListPlans" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"plans":[]}`))
	}))
	defer server.Close()

	stub := &planDemandStub{}
	client := HTTPPlanManagerClient{
		BaseURL:          server.URL,
		Demand:           stub,
		DemandConsumerID: "test:plan-client",
	}
	if _, err := client.ListPlans(context.Background(), WorkspaceScope{}, false); err != nil {
		t.Fatalf("ListPlans: %v", err)
	}
	if len(stub.acquired) != 1 || len(stub.released) != 1 {
		t.Fatalf("demand lifecycle = acquired %d released %d", len(stub.acquired), len(stub.released))
	}
	req := stub.acquired[0]
	if req.Kind != demand.KindDependency || req.ConsumerID != "test:plan-client" {
		t.Fatalf("unexpected demand request: %+v", req)
	}
	if stub.released[0] != req.LeaseID {
		t.Fatalf("released lease %q, acquired %q", stub.released[0], req.LeaseID)
	}
}
