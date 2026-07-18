package machines

import (
	"testing"

	"github.com/stretchr/testify/require"
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
