package main

import (
	"context"
	"errors"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	sessionsH "web-console/handlers/sessions"
	intsessions "web-console/internal/sessions"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
	targetsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/web-console/v1/targets"
)

func readyRemoteTarget() remoteTerminalTarget {
	return remoteTerminalTarget{
		ID: "bridge-node:node-a", Kind: "bridge-node", Label: "Build node A",
		OS: "linux", Arch: "amd64", NodeID: "node-a", Revision: "r1",
		Status: "ONLINE", Online: true, Available: true, State: "dispatchable",
		BaseURL: "https://bridge.internal", OwnerToken: "LocalSession secret-owner", ReauthToken: "secret-reauth",
		ReadinessFacts: []remoteReadinessFact{{Key: "dispatch", Label: "Dispatchable", Passed: true, Detail: "ready"}},
	}
}

func TestTargetCatalogListProjectsSafeRemoteMetadata(t *testing.T) {
	remote := readyRemoteTarget()
	srv := &Server{remoteTargetCatalog: func() []remoteTerminalTarget { return []remoteTerminalTarget{remote} }}

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
		remote []remoteTerminalTarget
		want   targetsv1.CatalogState
	}{
		{name: "configured empty", want: targetsv1.CatalogState_CATALOG_STATE_CONFIGURED_EMPTY},
		{name: "unconfigured", remote: []remoteTerminalTarget{{FailureRung: "bridge credentials not configured"}}, want: targetsv1.CatalogState_CATALOG_STATE_UNCONFIGURED},
		{name: "registry error", remote: []remoteTerminalTarget{{FailureRung: "Bridge registry unavailable"}}, want: targetsv1.CatalogState_CATALOG_STATE_REGISTRY_ERROR},
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

func TestTargetStateForNodeSurfacesNeedsUpdate(t *testing.T) {
	state := targetStateForNode(&registryv1.Node{
		Status: registryv1.NodeStatus_NODE_STATUS_NEEDS_UPDATE,
		Online: true, HeartbeatFresh: true,
	}, false)
	if state != "needs-update" {
		t.Fatalf("targetStateForNode() = %q, want needs-update", state)
	}
}

func TestTypedRemoteSessionLifecycleAndUnavailableTarget(t *testing.T) {
	remote := readyRemoteTarget()
	unavailable := remote
	unavailable.ID = "bridge-node:offline"
	unavailable.Label = "Offline node"
	unavailable.Available = false
	unavailable.State = "offline"
	unavailable.FailureRung = "heartbeat freshness"
	srv := &Server{
		remoteSessions:      &remoteTerminalRegistry{sessions: make(map[string]remoteTerminalSession)},
		remoteTargetCatalog: func() []remoteTerminalTarget { return []remoteTerminalTarget{remote, unavailable} },
	}

	created, err := srv.Create(context.Background(), sessionsH.CreateInput{
		TargetID:             remote.ID,
		LaunchCommand:        "codex login --device-auth",
		ExecuteLaunchCommand: true,
		WorkingDir:           "/workspaces/demo",
	})
	if err != nil {
		t.Fatalf("remote Create() error = %v", err)
	}
	if !strings.HasPrefix(created.ID, "remote:") || created.Target == nil || created.Target.GetId() != remote.ID {
		t.Fatalf("created session missing target metadata: %+v", created)
	}
	if created.SurvivesRestart {
		t.Fatal("remote session incorrectly promises restart durability")
	}

	listed, err := srv.List(context.Background())
	if err != nil || len(listed) != 1 || listed[0].ID != created.ID {
		t.Fatalf("List() = %#v, error = %v", listed, err)
	}
	got, err := srv.Get(context.Background(), created.ID)
	if err != nil || got.Target == nil || got.Target.GetLabel() != remote.Label {
		t.Fatalf("Get() = %#v, error = %v", got, err)
	}
	if err := srv.Delete(context.Background(), created.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := srv.Get(context.Background(), created.ID); !errors.Is(err, sessionsH.ErrNotFound) {
		t.Fatalf("Get() after Delete() error = %v, want ErrNotFound", err)
	}

	_, err = srv.Create(context.Background(), sessionsH.CreateInput{TargetID: unavailable.ID})
	if !errors.Is(err, sessionsH.ErrTargetUnavailable) {
		t.Fatalf("unavailable Create() error = %v, want ErrTargetUnavailable", err)
	}
	if listed, listErr := srv.List(context.Background()); listErr != nil || len(listed) != 0 {
		t.Fatalf("unavailable create changed registry: %#v, error = %v", listed, listErr)
	}
}

func TestTypedRemoteCreateIsReplaySafeAndPreservesTargetMetadata(t *testing.T) {
	remote := readyRemoteTarget()
	srv := &Server{
		remoteSessions:      &remoteTerminalRegistry{sessions: make(map[string]remoteTerminalSession)},
		remoteTargetCatalog: func() []remoteTerminalTarget { return []remoteTerminalTarget{remote} },
	}
	adapter := &sessionsH.Adapter{Remote: srv, Idempotency: intsessions.NewIdempotencyCache()}
	in := sessionsH.CreateInput{TargetID: remote.ID, IdempotencyKey: "remote-create-replay", WorkingDir: "/workspaces/demo"}
	first, err := adapter.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("first remote create: %v", err)
	}
	second, err := adapter.Create(context.Background(), in)
	if err != nil {
		t.Fatalf("replayed remote create: %v", err)
	}
	if second.ID != first.ID {
		t.Fatalf("replayed remote create returned %q, want original %q", second.ID, first.ID)
	}
	if second.Target == nil || second.Target.GetId() != remote.ID {
		t.Fatalf("replayed remote create lost target metadata: %+v", second.Target)
	}
	if _, err := adapter.Create(context.Background(), sessionsH.CreateInput{
		TargetID: remote.ID, IdempotencyKey: in.IdempotencyKey, WorkingDir: "/workspaces/other",
	}); !errors.Is(err, sessionsH.ErrIdempotencyConflict) {
		t.Fatalf("reused key with different request error = %v, want ErrIdempotencyConflict", err)
	}
	listed, err := srv.List(context.Background())
	if err != nil || len(listed) != 1 {
		t.Fatalf("remote registry after replay = %#v, error = %v; want one session", listed, err)
	}
}
