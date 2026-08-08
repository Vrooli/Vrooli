package bridgevalidation

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	domainv1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-to-desktop/v1/domain"
	dispatchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/dispatch"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	runsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/runs"
	"scenario-to-desktop-api/validationmatrix"
)

type fakeRegistry struct {
	nodes []*registryv1.Node
}

func (f fakeRegistry) ListNodes(context.Context, *connect.Request[registryv1.ListNodesRequest]) (*connect.Response[registryv1.ListNodesResponse], error) {
	return connect.NewResponse(&registryv1.ListNodesResponse{Nodes: f.nodes}), nil
}

type fakeDispatcher struct {
	request *dispatchv1.DispatchJobRequest
}

func (f *fakeDispatcher) DispatchJob(_ context.Context, request *connect.Request[dispatchv1.DispatchJobRequest]) (*connect.Response[dispatchv1.DispatchJobResponse], error) {
	f.request = request.Msg
	return connect.NewResponse(&dispatchv1.DispatchJobResponse{RunId: "run-1", NodeId: request.Msg.NodeId, Scenario: request.Msg.Scenario, Verb: request.Msg.Verb}), nil
}

type fakeRuns struct {
	status runsv1.RunStatus
}

func (f fakeRuns) WaitRun(_ context.Context, request *connect.Request[runsv1.WaitRunRequest]) (*connect.Response[runsv1.WaitRunResponse], error) {
	return connect.NewResponse(&runsv1.WaitRunResponse{Run: &runsv1.Run{Id: request.Msg.Id, Status: f.status}}), nil
}

func TestDiscoverMapsBridgeIdentityAndCapabilities(t *testing.T) {
	client := NewClientForTesting(fakeRegistry{nodes: []*registryv1.Node{
		{Id: "node-1", Name: "Linux runner", Os: "linux", Arch: "amd64", Online: true, Status: registryv1.NodeStatus_NODE_STATUS_ONLINE, Capabilities: []string{"electron-cdp", "native-window", "process-metrics"}, Scopes: []string{"scenario test*"}},
		{Id: "node-2", Name: "Offline runner", Online: false, Status: registryv1.NodeStatus_NODE_STATUS_OFFLINE},
	}}, nil, nil)

	targets, err := client.Discover(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 2 {
		t.Fatalf("targets = %d, want 2", len(targets))
	}
	if got := targets[0].Descriptor.GetTargetId(); got != "bridge:node-1" {
		t.Fatalf("target id = %q", got)
	}
	if !targets[0].Descriptor.GetAvailable() || targets[0].NodeID != "node-1" {
		t.Fatalf("online node was not available: %#v", targets[0])
	}
	if targets[0].Health.Status != "healthy" || targets[0].BridgeTrust == nil || !targets[0].BridgeTrust.DispatchAuthorized {
		t.Fatalf("online node health/trust = %#v/%#v", targets[0].Health, targets[0].BridgeTrust)
	}
	if targets[1].Descriptor.GetAvailable() {
		t.Fatal("offline node was reported available")
	}
	if targets[1].Health.Status != "offline" || targets[1].BridgeTrust == nil || targets[1].BridgeTrust.DispatchAuthorized {
		t.Fatalf("offline node health/trust = %#v/%#v", targets[1].Health, targets[1].BridgeTrust)
	}
}

func TestExecuteBridgeNeverClaimsDesktopPassWithoutTargetEvidence(t *testing.T) {
	dispatcher := &fakeDispatcher{}
	client := NewClientForTesting(nil, dispatcher, fakeRuns{status: runsv1.RunStatus_RUN_STATUS_PASSED})
	result := client.ExecuteBridge(context.Background(), validationmatrix.CellRequest{RunID: "matrix-run", Cell: &domainv1.ValidationCell{ScenarioName: "demo", TargetId: "bridge:node-1"}})

	if result.Disposition != domainv1.ValidationDisposition_VALIDATION_DISPOSITION_DEGRADED {
		t.Fatalf("disposition = %s, want degraded", result.Disposition)
	}
	if len(result.Evidence) != 1 || result.Evidence[0].GetKind() != domainv1.LayeredEvidence_KIND_TARGET {
		t.Fatalf("bridge evidence = %#v", result.Evidence)
	}
	if dispatcher.request.GetVerb() != "scenario test" || dispatcher.request.GetScenario() != "demo" {
		t.Fatalf("typed dispatch request = %#v", dispatcher.request)
	}
}

func TestExecuteBridgeMapsFailedAndAbortedRuns(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status runsv1.RunStatus
		want   domainv1.ValidationDisposition
	}{
		{name: "failed", status: runsv1.RunStatus_RUN_STATUS_FAILED, want: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_FAILED},
		{name: "aborted", status: runsv1.RunStatus_RUN_STATUS_ABORTED, want: domainv1.ValidationDisposition_VALIDATION_DISPOSITION_NOT_RUN},
	} {
		t.Run(tc.name, func(t *testing.T) {
			client := NewClientForTesting(nil, &fakeDispatcher{}, fakeRuns{status: tc.status})
			result := client.ExecuteBridge(context.Background(), validationmatrix.CellRequest{Cell: &domainv1.ValidationCell{ScenarioName: "demo", TargetId: "bridge:node-1"}})
			if result.Disposition != tc.want {
				t.Fatalf("disposition = %s, want %s", result.Disposition, tc.want)
			}
		})
	}
}
