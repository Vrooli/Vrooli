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
