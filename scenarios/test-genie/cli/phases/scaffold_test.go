package phases

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/vrooli/maturity-go/assessment"

	"test-genie/internal/providerconformance"
)

// TestScaffoldDocHasEverySkeletonHeading proves the scaffolded remediation doc
// contains every required H2 heading (the SSOT is providerconformance), so its
// output passes the doc-skeleton check by construction.
func TestScaffoldDocHasEverySkeletonHeading(t *testing.T) {
	doc := scaffoldDoc("contracts")
	for _, heading := range providerconformance.RequiredDocHeadings {
		if !strings.Contains(doc, "## "+heading+"\n") {
			t.Errorf("scaffold doc missing required H2 heading %q\n---\n%s", heading, doc)
		}
	}
}

// TestScaffoldMaturityPassesLadderShape proves the scaffolded maturity stub
// satisfies exactly what validateMaturityContract requires: the top rung of every
// ladder carries a North Star (capability_summary) and every non-top rung names a
// next_unlock, with entry/exit gates on all rungs. Placeholders are non-empty, so
// the structure passes while prompting for real content.
func TestScaffoldMaturityPassesLadderShape(t *testing.T) {
	raw, err := scaffoldMaturity("demo", "contracts")
	if err != nil {
		t.Fatalf("scaffoldMaturity: %v", err)
	}
	var spec assessment.Spec
	if err := json.Unmarshal([]byte(raw), &spec); err != nil {
		t.Fatalf("unmarshal stub: %v", err)
	}
	checkLadder(t, "phase", spec.Levels)
	if len(spec.Capabilities) == 0 {
		t.Fatal("scaffold stub should declare at least one capability ladder")
	}
	for _, capability := range spec.Capabilities {
		checkLadder(t, "capability "+capability.ID, capability.Levels)
	}
}

func checkLadder(t *testing.T, label string, levels []assessment.Level) {
	t.Helper()
	if len(levels) == 0 {
		return
	}
	top := levels[len(levels)-1]
	if strings.TrimSpace(top.CapabilitySummary) == "" {
		t.Errorf("%s ladder top rung must carry a North Star (capability_summary)", label)
	}
	for i, level := range levels {
		if len(level.EntryCriteria) == 0 || len(level.ExitCriteria) == 0 {
			t.Errorf("%s rung %s missing entry/exit criteria", label, level.ID)
		}
		if i != len(levels)-1 && strings.TrimSpace(level.NextUnlock) == "" {
			t.Errorf("%s non-top rung %s missing next_unlock", label, level.ID)
		}
	}
}

func TestScaffoldRequiresPhaseArg(t *testing.T) {
	var buf strings.Builder
	if err := runScaffold(nil, &buf); err == nil {
		t.Fatal("expected an error when no phase is given")
	}
}
