package opscatalog

import (
	"testing"

	"swarm-manager/internal/archtest"
)

// TestCatalogLoaderHasNoModeNameBranches extends the generic-substrate purity
// invariant to the catalog loader: opscatalog reads and validates authored
// DATA (contracts, bindings, policies) and must treat every mode name as an
// opaque value — no shipped mode id may appear in its code, or a catalog edit
// could silently change loader behavior. The forbidden vocabulary is derived
// from the authored modes/ catalog plus the sentinel and the operatingmode
// Mode* constants. Red-proof: archtest's
// TestModeNamePurityScannerFiresOnViolation exercises this exact scanner
// against a synthetic violation.
func TestCatalogLoaderHasNoModeNameBranches(t *testing.T) {
	archtest.RequireNoModeNameBranches(t, ".", "../../../modes",
		"ModeHolisticLoop", "ModePhasedPlanDrain", "ModeItemLevel")
}
