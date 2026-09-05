package machines

import (
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/vrooli/packages/proto/privilegedops"
)

func TestEvaluateReadinessDerivesOwnedFactsWithoutGrantingScopes(t *testing.T) {
	machine := Machine{Lifecycle: LifecycleActive}
	trust := TrustRecord{HostKeyState: HostKeyVerified}
	policy := PolicySnapshot{RequiredCapabilities: []string{"agent.presence"}, SuggestedScopes: []string{"presence.read"}}
	projection := Projection{HasNode: true, Node: NodeSnapshot{Capabilities: []string{"agent.presence"}}, Presence: PresenceSnapshot{Connected: true}}
	result := EvaluateReadiness(machine, trust, policy, projection)
	require.False(t, result.Ready)
	require.Equal(t, []string{"scope_not_approved:presence.read"}, result.Reasons)

	projection.Node.ApprovedScopes = []string{"presence.read"}
	result = EvaluateReadiness(machine, trust, policy, projection)
	require.True(t, result.Ready)
	require.Empty(t, result.Reasons)
}

func TestEvaluateReadinessNamesEverySharedOnboardingCapability(t *testing.T) {
	capabilities := privilegedops.CapabilityNames()
	result := EvaluateReadiness(Machine{Lifecycle: LifecycleActive}, TrustRecord{HostKeyState: HostKeyVerified}, PolicySnapshot{RequiredCapabilities: capabilities}, Projection{
		HasNode:  true,
		Node:     NodeSnapshot{Capabilities: []string{privilegedops.CapabilityAgentPresence}},
		Presence: PresenceSnapshot{Connected: true},
	})
	for _, capability := range capabilities[1:] {
		require.Contains(t, result.Reasons, "missing_capability:"+capability)
	}
}

func TestComputeDriftNamesUnappliedProfileAndMissingCapability(t *testing.T) {
	machine := Machine{Lifecycle: LifecycleActive}
	policy := PolicySnapshot{ProfileID: "developer", ProfileVersion: "v2", RequiredCapabilities: []string{"ai-cli:codex"}}
	items := ComputeDrift(machine, policy, Projection{HasNode: true, Node: NodeSnapshot{Capabilities: []string{"ai-cli:claude"}}})
	if len(items) != 2 {
		t.Fatalf("drift = %+v, want profile and capability", items)
	}
	if items[0].Kind != "profile" || items[0].Name != "developer" {
		t.Fatalf("profile drift = %+v", items[0])
	}
	if items[1].Kind != "capability" || items[1].Name != "ai-cli:codex" {
		t.Fatalf("capability drift = %+v", items[1])
	}
}

func TestComputeDriftStopsAtProfileWhenNodeIsUnavailable(t *testing.T) {
	items := ComputeDrift(Machine{AppliedProfileID: "old", AppliedProfileVersion: "v1"}, PolicySnapshot{ProfileID: "new", ProfileVersion: "v2"}, Projection{})
	if len(items) != 1 || items[0].Kind != "profile" {
		t.Fatalf("drift = %+v, want only profile difference without node evidence", items)
	}
}

func TestComputeDriftNamesChangedSelectionField(t *testing.T) {
	items := ComputeDrift(Machine{
		AppliedProfileID:      "presence",
		AppliedProfileVersion: "v1",
		DesiredSelectionJSON:  `{"scenarios":["vrooli-bridge"],"session_mode":"service"}`,
		AppliedSelectionJSON:  `{"scenarios":["vrooli-bridge"],"session_mode":"interactive"}`,
	}, PolicySnapshot{ProfileID: "presence", ProfileVersion: "v1"}, Projection{})
	require.Len(t, items, 1)
	require.Equal(t, "session_mode", items[0].Name)
	require.Equal(t, "selection", items[0].Kind)
}
