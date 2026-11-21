package main

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/api-core/nodereach"
	"github.com/vrooli/api-core/operatorsession"
	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/vrooli-bridge/v1/registry"
)

type targetProbeReacher struct {
	request nodereach.ScenarioRequest
	err     error
}

func (f *targetProbeReacher) CallScenario(_ context.Context, request nodereach.ScenarioRequest) ([]byte, error) {
	f.request = request
	return nil, f.err
}

func TestBridgeClientConfigUsesTheEnrolledLocalOperatorSession(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("VROOLI_OPERATOR_SESSION_DIR", dir)
	private, err := operatorsession.GenerateKey()
	if err != nil {
		t.Fatal(err)
	}
	store, err := operatorsession.NewFileStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(private, operatorsession.Enrollment{
		OperatorID:       "operator-1",
		IdentityProvider: "scenario-authenticator",
		Mode:             operatorsession.ModePersonal,
		Reference:        "enrollment-1",
		EnrolledAt:       time.Now().UTC(),
		ScopeCeiling:     []string{"vrooli-bridge:read"},
	}); err != nil {
		t.Fatal(err)
	}

	config := bridgeClientConfig()
	if config.TokenProvider == nil {
		t.Fatal("bridge target discovery must provide the enrolled local operator session")
	}
	token, err := config.TokenProvider(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(token, operatorsession.LocalSessionScheme+" ") {
		t.Fatalf("token scheme = %q, want %q", token, operatorsession.LocalSessionScheme)
	}
}

func TestNodeToTargetOnlyProvidesRecoveryActionWhenUnavailable(t *testing.T) {
	online := nodeToTarget(&registryv1.Node{
		Id:                    "node-online",
		Name:                  "minimouse",
		Os:                    "darwin",
		Arch:                  "amd64",
		Status:                registryv1.NodeStatus_NODE_STATUS_ONLINE,
		Online:                true,
		RegistryRecordPresent: true,
		HeartbeatFresh:        true,
		ChannelHeld:           true,
		ProtocolCompatible:    true,
		Dispatchable:          true,
		Scopes:                []string{"vrooli-onboarding:read", "vrooli-onboarding:write"},
	})
	if !online.Available || online.NextAction != "" {
		t.Fatalf("online target = %+v, want available without next action", online)
	}

	offline := nodeToTarget(&registryv1.Node{
		Id:     "node-offline",
		Name:   "offline-node",
		Status: registryv1.NodeStatus_NODE_STATUS_OFFLINE,
	})
	if offline.NextAction == "" {
		t.Fatal("offline target should include a recovery action")
	}
}

func TestNodeToTargetRejectsConnectedNodeWithoutOnboardingScopes(t *testing.T) {
	target := nodeToTarget(&registryv1.Node{
		Id:                    "node-read-only",
		Name:                  "read-only-node",
		Status:                registryv1.NodeStatus_NODE_STATUS_ONLINE,
		Online:                true,
		RegistryRecordPresent: true,
		HeartbeatFresh:        true,
		ChannelHeld:           true,
		ProtocolCompatible:    true,
		Dispatchable:          true,
		Scopes:                []string{"system-monitor:read"},
	})
	if target.Available {
		t.Fatal("node without onboarding scopes must not be advertised as available")
	}
	if !strings.Contains(target.Reason, "vrooli-onboarding:read") || target.NextAction == "" {
		t.Fatalf("target = %+v, want missing-scope reason and remediation", target)
	}
}

func TestProbeOnboardingTargetUsesBoundedReadOnlySessionRequest(t *testing.T) {
	reacher := &targetProbeReacher{}
	if err := probeOnboardingTarget(context.Background(), reacher, "node-1"); err != nil {
		t.Fatal(err)
	}
	if reacher.request.NodeID != "node-1" || reacher.request.Scenario != "vrooli-onboarding" || reacher.request.Procedure != onboardingTargetProbeProcedure || reacher.request.Timeout != 1200*time.Millisecond || reacher.request.MaxResponse != 64<<10 {
		t.Fatalf("probe request = %+v", reacher.request)
	}
	if len(reacher.request.Body) == 0 {
		t.Fatal("probe request did not contain a serialized session request")
	}
}

func TestProbeOnboardingTargetPreservesReachFailure(t *testing.T) {
	want := errors.New("stale deployment")
	reacher := &targetProbeReacher{err: want}
	if err := probeOnboardingTarget(context.Background(), reacher, "node-1"); !errors.Is(err, want) {
		t.Fatalf("probe error = %v, want %v", err, want)
	}
}
