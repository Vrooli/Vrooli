package report

import (
	"bytes"
	"strings"
	"testing"

	catalogphases "test-genie/internal/orchestrator/phases"

	execTypes "test-genie/cli/internal/execute"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
)

// TestScorecardRendersForEveryCatalogPhase is the anti-drift guard proving the
// scorecard carries ZERO phase-specific knowledge: for every phase in the live
// catalog, a standing renders a uniform block with no per-phase branch. Adding or
// removing a catalog phase therefore requires no scorecard code change.
func TestScorecardRendersForEveryCatalogPhase(t *testing.T) {
	names := catalogphases.ValidPhaseNames()
	if len(names) == 0 {
		t.Fatal("catalog exposed zero phases")
	}
	for _, name := range names {
		standing := &commonv1.PhasePresentation{
			Phase:                name,
			CurrentLevel:         "L1",
			CurrentLevelLabel:    "Foundation",
			NextLevel:            "L2",
			CeilingLevel:         "L4",
			NorthStar:            "The " + name + " capability is fully realized.",
			NextAction:           "Advance " + name + " to the next rung.",
			BlockingFindingCodes: []string{name + ".example"},
			DocumentationTopics:  []string{name + " maturity next move"},
		}
		var buf bytes.Buffer
		pr := New(&buf, "demo", "", nil, nil, false, nil, nil)
		pr.printPhaseResults([]execTypes.Phase{{Name: name, Status: "passed", PhasePresentation: standing}})
		out := buf.String()
		if !strings.Contains(out, "standing:") {
			t.Errorf("phase %q: scorecard did not render a standing block\n%s", name, out)
		}
		if !strings.Contains(out, "North Star:") {
			t.Errorf("phase %q: scorecard omitted the North Star", name)
		}
	}
}
