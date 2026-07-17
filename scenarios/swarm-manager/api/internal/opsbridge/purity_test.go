package opsbridge

import (
	"testing"

	"swarm-manager/internal/archtest"
)

// TestBridgeHasNoModeNameBranches extends the generic-substrate purity
// invariant to the bridge: opsbridge wires the runner to production seams
// (engine, completion router, refresh driver) and routes COMPLETIONS by
// operation outcome + transition policy — never by which mode produced them.
// The forbidden vocabulary is derived from the authored modes/ catalog plus
// the sentinel and the operatingmode Mode* constants. Red-proof: archtest's
// TestModeNamePurityScannerFiresOnViolation exercises this exact scanner
// against a synthetic violation.
func TestBridgeHasNoModeNameBranches(t *testing.T) {
	archtest.RequireNoModeNameBranches(t, ".", "../../../modes",
		"ModeHolisticLoop", "ModePhasedPlanDrain", "ModeItemLevel")
}
