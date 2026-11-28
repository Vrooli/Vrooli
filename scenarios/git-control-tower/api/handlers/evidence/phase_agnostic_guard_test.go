package evidence

import (
	"os"
	"strings"
	"testing"
)

// TestProductionBoundaryHasNoCurrentPhaseRegistry prevents GCT's generic
// evidence seam from quietly growing a second Test Genie phase registry. Phase
// display and applicability come only from each run's descriptor snapshot.
func TestProductionBoundaryHasNoCurrentPhaseRegistry(t *testing.T) {
	source, err := os.ReadFile("connect_handler.go")
	if err != nil {
		t.Fatal(err)
	}
	text := string(source)
	for _, phase := range []string{"structure", "standards", "unit", "integration", "business", "performance", "smoke"} {
		if strings.Contains(text, `"`+phase+`"`) {
			t.Errorf("generic evidence boundary contains fixed phase key %q", phase)
		}
	}
	for _, comparison := range []string{"GetProducingPhase() ==", "GetProducingPhase() !="} {
		if strings.Contains(text, comparison) {
			t.Errorf("generic evidence boundary filters by producer phase: %s", comparison)
		}
	}
}
