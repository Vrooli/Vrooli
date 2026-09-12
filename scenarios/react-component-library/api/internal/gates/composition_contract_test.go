package gates

import "testing"

func TestHardCodedElevationIsAContextsDifferFailure(t *testing.T) {
	if !hardCodedElevation(`style={{ boxShadow: "var(--elev-raised)" }}`) {
		t.Fatal("hard-coded elevation was not detected")
	}
	if hardCodedElevation("boxShadow: `var(--elev-${elevation})`") {
		t.Fatal("context-derived elevation was incorrectly treated as hard-coded")
	}
}

func TestCompositionContractCalibrationSourceUsesRequiredFailureShape(t *testing.T) {
	if got := hardCodedElevationLine([]byte("first\nboxShadow: \"var(--elev-raised)\"\n")); got != 2 {
		t.Fatalf("hard-coded elevation line = %d, want 2", got)
	}
}
