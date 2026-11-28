package validation

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSimulatedQualificationCoversPolicyAndAdapterNegativeControls(t *testing.T) {
	result := runSimulatedQualification()
	require.NoError(t, result.validate())
	for _, name := range []string{"explicit-routing", "owned-settlement", "private-lifecycle", "acceptance-integrity", "adapter-maintainability"} {
		claim, ok := result.Claims[name].(map[string]any)
		require.True(t, ok, name)
		require.Equal(t, true, claim["in_band"], name)
		require.Equal(t, true, claim["negative_controls_passed"], name)
		require.Equal(t, "simulated", claim["execution_lane"], name)
		require.Equal(t, name, claim["target_id"], name)
		require.Equal(t, "portable-voice-v2", claim["target_revision"], name)
		require.Equal(t, "audio-tools-acceptance-v2", claim["method"], name)
		require.NotEmpty(t, claim["checks"], name)
	}
	require.NotContains(t, result.Claims, "dictation-trust")
	require.NotContains(t, result.Claims, "interactive-latency")
	require.NotContains(t, result.Claims, "corpus-quality")
	require.ElementsMatch(t, []string{"explicit-routing", "owned-settlement", "private-lifecycle", "acceptance-integrity", "adapter-maintainability"}, result.Details["supported_rows"])
	require.Contains(t, result.Checks, "route_owned_requires_consent:passed")
	require.Contains(t, result.Checks, "settlement_retry_is_idempotent:passed")
	require.Contains(t, result.Checks, "lifecycle_delete_removes_retained_item:passed")
}

func TestSimulatedDeviceRouteMatrixCoversDeclaredD1D6CellsWithoutDeviceClaims(t *testing.T) {
	result := simulatedDeviceRouteMatrix()
	require.Equal(t, 9, result["cell_count"])
	require.Equal(t, 27, result["route_case_count"])
	require.Equal(t, false, result["native_device_claim"])
	cells := result["cells"].([]map[string]any)
	require.Len(t, cells, 9)
	seen := map[string]bool{}
	for _, cell := range cells {
		id := cell["cell_id"].(string)
		seen[id] = true
		require.Equal(t, "simulated", cell["execution_lane"], id)
		require.Equal(t, "deterministic route policy only", cell["simulation_scope"], id)
		require.Equal(t, "not_run", cell["physical_qualification"], id)
		routes := cell["route_outcomes"].(map[string]string)
		require.Equal(t, map[string]string{"local": "local", "byok": "byok", "owned": "owned"}, routes, id)
	}
	require.ElementsMatch(t, []string{
		"D1-linux-chromium-cpu", "D1-linux-chromium-accelerator", "D2-linux-chromium-constrained",
		"D3-macos-safari", "D3-macos-chromium", "D4-windows-chromium",
		"D5-ios-safari", "D5-ios-installed-pwa", "D6-android-chrome",
	}, keysOf(seen))
	for _, id := range []string{"D5-ios-safari", "D5-ios-installed-pwa", "D6-android-chrome"} {
		require.Equal(t, "named physical device required", findSimulationCell(cells, id)["physical_requirement"], id)
	}
}

func findSimulationCell(cells []map[string]any, id string) map[string]any {
	for _, cell := range cells {
		if cell["cell_id"] == id {
			return cell
		}
	}
	return nil
}

func keysOf(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

func TestSimulatedReceiptBindsProviderCellToDeviceProfile(t *testing.T) {
	result := runSimulatedQualification()
	payload, err := simulatedAcceptanceMachineJSON(result, "D2", "build-1", "artifact-1")
	require.NoError(t, err)
	var receipt map[string]any
	require.NoError(t, json.Unmarshal(payload, &receipt))
	cell, ok := receipt["provider_cell"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, receipt["device_profile"], cell["device_profile"])
	require.Equal(t, "D2", cell["device_profile"])
	require.Equal(t, "simulated-policy", cell["engine_id"])
	require.Equal(t, "simulation-v1", cell["model_id"])
	require.Equal(t, "simulation", cell["strategy"])
	claims, ok := receipt["acceptance"].(map[string]any)
	require.True(t, ok)
	claim, ok := claims["explicit-routing"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, "artifact-1", claim["receipt_reference"])
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
		"passed": true, "target_revision": "portable-voice-v2", "method": "audio-tools-acceptance-v2",
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
