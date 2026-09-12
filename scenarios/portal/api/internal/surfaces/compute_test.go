package surfaces

import (
	"context"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	instancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance"
	instanceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/compute-manager/v1/instance/instance_v1connect"
)

type computeFixture struct {
	instanceconnect.UnimplementedInstanceServiceHandler
	instances []*instancev1.Instance
}

func (f computeFixture) ListInstances(context.Context, *connect.Request[instancev1.ListInstancesRequest]) (*connect.Response[instancev1.ListInstancesResponse], error) {
	return connect.NewResponse(&instancev1.ListInstancesResponse{Instances: f.instances}), nil
}

func TestComputeProviderProjectsScreenlessNodesWithoutDesktop(t *testing.T) {
	now := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	_, handler := instanceconnect.NewInstanceServiceHandler(computeFixture{instances: []*instancev1.Instance{{Id: "node-1", Provider: "local", Region: "east", Size: "small", State: instancev1.InstanceState_INSTANCE_STATE_RUNNING, BridgeMachineId: "bridge-1"}}})
	server := httptest.NewServer(handler)
	defer server.Close()
	provider := ComputeProvider{ResolveURL: func(context.Context, string) (string, error) { return server.URL, nil }, Client: server.Client(), Now: func() time.Time { return now }}
	result, err := provider.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Surfaces) != 1 {
		t.Fatalf("expected one compute surface, got %d", len(result.Surfaces))
	}
	surface := result.Surfaces[0]
	if surface.Kind != "device-panel" || surface.Ref.Target.HostNodeID != "bridge-1" || surface.Ref.OwnerScenario != "compute-manager" {
		t.Fatalf("unexpected compute identity: %+v", surface.Ref)
	}
	for _, capability := range surface.Capabilities {
		if capability.Capability == "desktop.session" {
			t.Fatal("compute node must never advertise a desktop session")
		}
	}
	if err := surface.Validate(); err != nil {
		t.Fatalf("invalid projected surface: %v", err)
	}
}
