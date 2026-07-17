package opsrunner

import (
	"testing"

	"swarm-manager/internal/archtest"
)

// TestGenericRunnerHasNoModeNameBranches is the static guard for the phase
// invariant: the generic runner must contain NO branch keyed to a named mode,
// phase, or shipped methodology. Selection lives entirely in data (operation
// contracts, bindings, policies) and behind the ModePreparer/ExecutionDriver
// seams. The forbidden vocabulary is derived from the authored modes/ catalog
// (every shipped mode id plus the member-item-strategy sentinel), so authoring
// a new mode automatically extends the guard; the Mode* identifiers cover the
// operatingmode Go constants. Red-proof: archtest's
// TestModeNamePurityScannerFiresOnViolation exercises this exact scanner
// against a synthetic violation.
// [REQ:REQ-P0-009-OPERATION-SPAWN-BOUNDARY]
func TestGenericRunnerHasNoModeNameBranches(t *testing.T) {
	archtest.RequireNoModeNameBranches(t, ".", "../../../modes",
		"ModeHolisticLoop", "ModePhasedPlanDrain", "ModeItemLevel")
}
