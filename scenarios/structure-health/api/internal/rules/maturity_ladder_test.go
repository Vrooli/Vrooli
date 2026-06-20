package rules

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/vrooli/maturity-go/assessment"
)

// scenarioRoot resolves the structure-health scenario directory (where .vrooli
// lives) from this test file's location: api/internal/rules → ../../.. .
func scenarioRoot(t *testing.T) string {
	t.Helper()
	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(thisFile), "..", "..", ".."))
}

// TestMaturityLadderIsWellFormed is the anti-drift gate for SH-BOUND-003: every
// finding code in .vrooli/maturity.json must carry a complete mapping (local
// level impact, dimension, default severity) and reference a defined ladder
// level (or the foundation_blocker sentinel). This keeps the emitted finding
// codes and the maturity ladder from drifting apart.
//
// [REQ:SH-BOUND-003]
func TestMaturityLadderIsWellFormed(t *testing.T) {
	spec, err := assessment.LoadSpecFromScenario(scenarioRoot(t))
	if err != nil {
		t.Fatalf("load maturity spec: %v", err)
	}
	if spec.Provider != "structure-health" || spec.Phase != "structure" {
		t.Fatalf("unexpected provider/phase: %q/%q", spec.Provider, spec.Phase)
	}
	if len(spec.Findings) == 0 {
		t.Fatal("maturity spec has no finding mappings")
	}

	levelIDs := map[string]bool{}
	for _, lvl := range spec.Levels {
		levelIDs[lvl.ID] = true
	}

	for code, m := range spec.Findings {
		if m.LocalLevelImpact == "" {
			t.Errorf("%s: empty local_level_impact", code)
		}
		if m.Dimension == "" {
			t.Errorf("%s: empty dimension", code)
		}
		if m.SeverityDefault == "" {
			t.Errorf("%s: empty severity_default", code)
		}
		// local_level_impact references a defined level (e.g. L0–L4) unless it
		// is the foundation_blocker sentinel used for unresolvable skeletons.
		if m.LocalLevelImpact != "" && !levelIDs[m.LocalLevelImpact] &&
			string(m.GlobalImpact) != "foundation_blocker" {
			t.Errorf("%s: local_level_impact %q is not a defined level", code, m.LocalLevelImpact)
		}
	}
}
