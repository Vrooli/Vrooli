package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/targetmodel"
	"google.golang.org/protobuf/encoding/protojson"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	bridgeSharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/shared"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/shared"
	targetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/targets"
)

func readyRemoteTarget() targetConnection {
	return targetConnection{
		Target: targetmodel.Target{
			ID: "bridge-node:node-a", DeviceKind: "bridge-node", Label: "Build node A",
			OS: "linux", Architecture: "amd64", NodeID: "node-a", Revision: "r1", Health: targetmodel.TargetHealth{Status: "ONLINE"},
			BridgeTrust: &targetmodel.BridgeTrust{Online: true}, Available: true,
			Readiness: []targetmodel.ReadinessCheck{{Identity: "dispatch", Label: "Dispatchable", Passed: true, Detail: "ready"}},
		},
		BaseURL: "https://bridge.internal", OwnerToken: "LocalSession secret-owner", ReauthToken: "secret-reauth",
	}
}

func TestTargetCatalogListProjectsSafeRemoteMetadata(t *testing.T) {
	remote := readyRemoteTarget()
	srv := &Server{remoteTargetCatalog: func() []targetConnection { return []targetConnection{remote} }}

	response, err := (&targetCatalogRPC{server: srv}).List(context.Background(), connect.NewRequest(&targetsv1.ListRequest{}))
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if got := response.Msg.GetState(); got != targetsv1.CatalogState_CATALOG_STATE_READY {
		t.Fatalf("catalog state = %s, want READY", got)
	}
	if len(response.Msg.GetTargets()) != 2 {
		t.Fatalf("target count = %d, want local plus remote", len(response.Msg.GetTargets()))
	}
	projected := response.Msg.GetTargets()[1]
	if projected.GetId() != remote.ID || !projected.GetDispatchable() || projected.GetOs() != "linux" {
		t.Fatalf("unexpected projected target: %v", projected)
	}

	wire, err := protojson.Marshal(response.Msg)
	if err != nil {
		t.Fatalf("marshal catalog: %v", err)
	}
	for _, secret := range []string{remote.OwnerToken, remote.ReauthToken, remote.BaseURL} {
		if strings.Contains(string(wire), secret) {
			t.Fatalf("catalog leaked credential or endpoint %q: %s", secret, wire)
		}
	}
}

func TestRemoteCatalogStateDistinguishesConfigurationFailures(t *testing.T) {
	tests := []struct {
		name   string
		remote []targetConnection
		want   targetsv1.CatalogState
	}{
		{name: "configured empty", want: targetsv1.CatalogState_CATALOG_STATE_CONFIGURED_EMPTY},
		{name: "unconfigured", remote: []targetConnection{{Target: targetmodel.Target{Reason: "bridge credentials not configured"}}}, want: targetsv1.CatalogState_CATALOG_STATE_UNCONFIGURED},
		{name: "registry error", remote: []targetConnection{{Target: targetmodel.Target{Reason: "Bridge registry unavailable"}}}, want: targetsv1.CatalogState_CATALOG_STATE_REGISTRY_ERROR},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			state, _, _ := remoteCatalogState(tt.remote)
			if state != tt.want {
				t.Fatalf("remoteCatalogState() = %s, want %s", state, tt.want)
			}
		})
	}
}

func TestRemoteTargetExistsAndOffersBridgeActionWithoutConfiguration(t *testing.T) {
	t.Setenv("VROOLI_OPERATOR_SESSION_DIR", t.TempDir())
	t.Setenv("VROOLI_BRIDGE_NODE_ID", "")
	t.Setenv("VROOLI_BRIDGE_API_TOKEN", "")
	t.Setenv("VROOLI_BRIDGE_REAUTH_TOKEN", "")

	srv := &Server{}
	target, ok := srv.targetByID("bridge-node:")
	if !ok {
		t.Fatal("remote target disappeared when Bridge is unconfigured")
	}
	if target.Available || target.Reason != "Bridge credentials not configured" {
		t.Fatalf("target status = %+v", target)
	}
	if target.NextAction != "Enroll this machine with Bridge, then refresh the catalog" {
		t.Fatalf("target recovery action = %q", target.NextAction)
	}
}

func TestTargetStateForNodeSurfacesNeedsUpdate(t *testing.T) {
	state := targetStateForNode(&registryv1.Node{
		Status: registryv1.NodeStatus_NODE_STATUS_NEEDS_UPDATE,
		Online: true, HeartbeatFresh: true,
	}, false)
	if state != "needs-update" {
		t.Fatalf("targetStateForNode() = %q, want needs-update", state)
	}
}

func TestTargetCatalogProjectsReadinessAndRecoveryText(t *testing.T) {
	node := &registryv1.Node{
		Kind:                  registryv1.NodeKind_NODE_KIND_AGENT,
		RegistryRecordPresent: true,
		Online:                true,
		HeartbeatFresh:        true,
		ChannelHeld:           true,
		ProtocolCompatible:    true,
		Dispatchable:          false,
		Scopes:                []string{"system-monitor:read"},
	}
	facts := readinessFactsForNode(node)
	if len(facts) != 7 || facts[0].Passed != true || facts[4].Passed != false || facts[5].Passed || facts[6].Passed {
		t.Fatalf("readiness facts = %+v", facts)
	}
	if facts[5].Detail != "Read only; changes are not permitted. Granted scopes: system-monitor:read" {
		t.Fatalf("grant summary = %q", facts[5].Detail)
	}
	if got := recoveryActionForNode(node, "protocol compatibility"); !strings.Contains(got, "Update") {
		t.Fatalf("protocol recovery action = %q", got)
	}
	if got := recoveryActionForNode(&registryv1.Node{Kind: registryv1.NodeKind_NODE_KIND_CONTROL_PLANE}, "other"); !strings.Contains(got, "does not host") {
		t.Fatalf("controller recovery action = %q", got)
	}

	for _, tc := range []struct {
		state string
		want  sharedv1.TargetState
	}{
		{"dispatchable", sharedv1.TargetState_TARGET_STATE_DISPATCHABLE},
		{"offline", sharedv1.TargetState_TARGET_STATE_OFFLINE},
		{"needs-update", sharedv1.TargetState_TARGET_STATE_NEEDS_UPDATE},
		{"unavailable", sharedv1.TargetState_TARGET_STATE_UNAVAILABLE},
	} {
		if got := targetProtoState(tc.state); got != tc.want {
			t.Errorf("targetProtoState(%q) = %v, want %v", tc.state, got, tc.want)
		}
	}
	projected := targetToProto(targetConnection{Target: targetmodel.Target{
		ID: "node", Available: false, Reason: "heartbeat freshness", NextAction: "reconnect",
		LastSeenAt: time.Now(), Readiness: facts,
	}})
	if targetText(projected, "failure_rung") != "heartbeat freshness" || targetText(projected, "recovery_action") != "reconnect" {
		t.Fatalf("target text projection = %q/%q", targetText(projected, "failure_rung"), targetText(projected, "recovery_action"))
	}
	if targetText(nil, "failure_rung") != "" {
		t.Fatal("nil target text should be empty")
	}
}

func TestTargetCatalogProjectsCapabilityInventory(t *testing.T) {
	node := &registryv1.Node{
		Kind:                  registryv1.NodeKind_NODE_KIND_AGENT,
		RegistryRecordPresent: true,
		CapabilityInventory: []*bridgeSharedv1.CapabilityObservation{{
			Capability: "ai-cli", Id: "codex", Label: "Codex",
			State:  bridgeSharedv1.CapabilityObservationState_CAPABILITY_OBSERVATION_STATE_MISSING,
			Version: "1.2.3", Detail: "codex is not installed",
		}},
	}
	facts := readinessFactsForNode(node)
	var found *targetmodel.ReadinessCheck
	for index := range facts {
		if facts[index].Identity == "capability:codex" {
			found = &facts[index]
			break
		}
	}
	if found == nil || found.State != targetmodel.ReadinessMissing || found.Passed || found.Version != "1.2.3" {
		t.Fatalf("capability readiness = %+v", found)
	}
	projected := targetToProto(targetConnection{Target: targetmodel.Target{Readiness: facts}})
	for _, fact := range projected.GetReadiness() {
		if fact.GetKey() == "capability:codex" && fact.GetState() != "missing" {
			t.Fatalf("projected capability state = %q", fact.GetState())
		}
	}
}

func TestTargetStateForNodeReportsDispatchAndAvailabilityStates(t *testing.T) {
	if got := targetStateForNode(&registryv1.Node{}, true); got != "dispatchable" {
		t.Fatalf("dispatchable state = %q", got)
	}
	if got := targetStateForNode(&registryv1.Node{Status: registryv1.NodeStatus_NODE_STATUS_OFFLINE}, false); got != "offline" {
		t.Fatalf("offline status state = %q", got)
	}
	if got := targetStateForNode(&registryv1.Node{Online: false, HeartbeatFresh: true}, false); got != "offline" {
		t.Fatalf("offline node state = %q", got)
	}
	if got := targetStateForNode(&registryv1.Node{Online: true, HeartbeatFresh: false}, false); got != "offline" {
		t.Fatalf("stale heartbeat state = %q", got)
	}
	if got := targetStateForNode(&registryv1.Node{Online: true, HeartbeatFresh: true}, false); got != "unavailable" {
		t.Fatalf("unavailable state = %q", got)
	}
}

func TestTargetCatalogGetAndDoctorExplainFailures(t *testing.T) {
	srv := &Server{remoteTargetCatalog: func() []targetConnection {
		return []targetConnection{{Target: targetmodel.Target{ID: "node", Label: "Node", Reason: "heartbeat freshness"}}}
	}}
	h := &targetCatalogRPC{server: srv}
	if response, err := h.Get(context.Background(), connect.NewRequest(&targetsv1.GetRequest{Id: "local"})); err != nil || response.Msg.GetTarget().GetId() != "local" {
		t.Fatalf("Get local = %v/%v", response, err)
	}
	if _, err := h.Get(context.Background(), connect.NewRequest(&targetsv1.GetRequest{Id: "missing"})); connect.CodeOf(err) != connect.CodeNotFound {
		t.Fatalf("Get missing error = %v", err)
	}
	response, err := h.Doctor(context.Background(), connect.NewRequest(&targetsv1.DoctorRequest{Id: "node"}))
	if err != nil || response.Msg.GetSummary() != "heartbeat freshness" {
		t.Fatalf("Doctor unavailable = %v/%v", response, err)
	}
	if response, err := h.Doctor(context.Background(), connect.NewRequest(&targetsv1.DoctorRequest{Id: "local"})); err != nil || response.Msg.GetSummary() != "target is dispatchable" {
		t.Fatalf("Doctor local = %v/%v", response, err)
	}
}

func TestTargetCatalogTextHelpersIgnoreNilAndUnknownFields(t *testing.T) {
	setTargetText(nil, "failure_rung", "ignored")
	setCatalogText(nil, "recovery_action", "ignored")
	target := &sharedv1.Target{}
	setTargetText(target, "unknown_field", "ignored")
	setTargetText(target, "failure_rung", "failure")
	if targetText(target, "failure_rung") != "failure" || targetText(target, "unknown_field") != "" {
		t.Fatalf("target text = %q/%q", targetText(target, "failure_rung"), targetText(target, "unknown_field"))
	}
	response := &targetsv1.ListResponse{}
	setCatalogText(response, "unknown_field", "ignored")
	setCatalogText(response, "recovery_action", "recover")
	wire, err := protojson.Marshal(response)
	if err != nil || !strings.Contains(string(wire), `"recoveryAction":"recover"`) {
		t.Fatalf("catalog recovery action wire = %s (err=%v)", wire, err)
	}
}
