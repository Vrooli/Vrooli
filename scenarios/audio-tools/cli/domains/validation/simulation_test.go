package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimulatedQualificationCoversPolicyAndAdapterNegativeControls(t *testing.T) {
	result := runSimulatedQualification()
	require.NoError(t, result.validate())
	for _, name := range []string{"explicit-routing", "owned-settlement", "private-lifecycle", "adapter-maintainability"} {
		claim, ok := result.Claims[name].(map[string]any)
		require.True(t, ok, name)
		require.Equal(t, true, claim["in_band"], name)
		require.Equal(t, true, claim["negative_controls_passed"], name)
		require.Equal(t, "simulated", claim["execution_lane"], name)
	}
	require.NotContains(t, result.Claims, "dictation-trust")
	require.NotContains(t, result.Claims, "interactive-latency")
	require.NotContains(t, result.Claims, "corpus-quality")
	require.ElementsMatch(t, []string{"explicit-routing", "owned-settlement", "private-lifecycle", "acceptance-integrity", "adapter-maintainability"}, result.Details["supported_rows"])
	require.Contains(t, result.Checks, "route_owned_requires_consent:passed")
	require.Contains(t, result.Checks, "settlement_retry_is_idempotent:passed")
	require.Contains(t, result.Checks, "lifecycle_delete_removes_retained_item:passed")
}

func TestSimulatedSettlementRejectsDuplicateReservationAndOverSettlement(t *testing.T) {
	ledger := simulatedSettlement{reserved: map[string]int{}, settled: map[string]int{}, allowance: 4}
	require.True(t, ledger.reserve("op", 4))
	require.False(t, ledger.reserve("op", 1))
	require.False(t, ledger.finalize("op", 5))
	require.True(t, ledger.finalize("op", 2))
	require.True(t, ledger.finalize("op", 2))
	require.Equal(t, 2, ledger.debit)
}

func TestAcceptanceClaimPassesRequiresExplicitSimulationEnvelope(t *testing.T) {
	claim := map[string]any{"in_band": true, "negative_controls_passed": true}
	receipt := map[string]any{
		"passed": true, "target_revision": "portable-voice-v1", "method": "audio-tools-acceptance-v1",
		"execution_lane": "simulated", "simulated_as_native": false,
		"acceptance": map[string]any{"explicit-routing": claim},
	}
	require.True(t, acceptanceClaimPasses(receipt))

	delete(receipt, "execution_lane")
	require.False(t, acceptanceClaimPasses(receipt), "missing lane must not pass")
	receipt["execution_lane"] = "simulated"
	receipt["simulated_as_native"] = true
	require.False(t, acceptanceClaimPasses(receipt), "simulated-as-native must not pass")
}
