package phases

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	routingconnect "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing/routing_v1connect"
	"test-genie/internal/orchestrator/phases/isolation"
	"test-genie/internal/orchestrator/workspace"
)

type routedLeaseFixture struct {
	installed bool
	cleared   bool
	dsn       string
}

func (f *routedLeaseFixture) InstallTestPool(_ context.Context, req *connect.Request[routingv1.InstallTestPoolRequest]) (*connect.Response[routingv1.InstallTestPoolResponse], error) {
	f.installed = true
	f.dsn = req.Msg.GetDsn()
	return connect.NewResponse(&routingv1.InstallTestPoolResponse{ActiveLeaseId: req.Msg.GetLeaseId(), FileRootsInstalled: true}), nil
}

func (f *routedLeaseFixture) ClearTestPool(_ context.Context, _ *connect.Request[routingv1.ClearTestPoolRequest]) (*connect.Response[routingv1.ClearTestPoolResponse], error) {
	f.cleared = true
	return connect.NewResponse(&routingv1.ClearTestPoolResponse{Stats: &routingv1.LeaseStats{TestPoolRequests: 3, TestRootWrites: 2}}), nil
}

func (*routedLeaseFixture) HeartbeatTestPool(context.Context, *connect.Request[routingv1.HeartbeatTestPoolRequest]) (*connect.Response[routingv1.HeartbeatTestPoolResponse], error) {
	return connect.NewResponse(&routingv1.HeartbeatTestPoolResponse{}), nil
}

func TestRoutedQualificationLeaseInstallsAndClearsPairedOwnerRoute(t *testing.T) {
	fixture := &routedLeaseFixture{}
	path, handler := routingconnect.NewRoutingServiceHandler(fixture)
	mux := http.NewServeMux()
	mux.Handle(path, handler)
	server := httptest.NewServer(mux)
	defer server.Close()

	lease, err := installRoutedLease(context.Background(), workspace.Environment{APIURL: server.URL}, resourceNeeds{PrimaryDriver: "sqlite"}, &isolation.Result{RunID: "qualification-1", Env: map[string]string{"PLAYBOOKS_SQLITE_DSN": "file:test-qualification"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !fixture.installed || fixture.dsn != "file:test-qualification" || lease.leaseID != "qualification-1" {
		t.Fatalf("install fixture=%+v lease=%+v", fixture, lease)
	}
	if err := clearRoutedLease(context.Background(), lease, nil); err != nil {
		t.Fatal(err)
	}
	if !fixture.cleared {
		t.Fatal("routed lease was not cleared")
	}
}

func TestRoutedQualificationRefusesWithoutLiveTargetAPI(t *testing.T) {
	_, err := installRoutedLease(context.Background(), workspace.Environment{}, resourceNeeds{PrimaryDriver: "sqlite"}, &isolation.Result{RunID: "qualification-2", Env: map[string]string{"PLAYBOOKS_SQLITE_DSN": "file:test-qualification"}}, nil)
	if err == nil {
		t.Fatal("qualification accepted without a live target API")
	}
}
