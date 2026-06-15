package baseline

import (
	"strings"
	"testing"
)

// bucketPhaseDiffs folds a flat per-phase compare into one diff per phase-set
// surface (option-c). Phases with no owning surface are dropped.
func TestBucketPhaseDiffs(t *testing.T) {
	cmp := CompareResult{
		Verdict: "regression",
		Phases: []PhaseDiff{
			{Phase: "structure", Verdict: "clean"},
			{Phase: "standards", Verdict: "new-failure", NewFailures: []string{"PRD-009"}},
			{Phase: "unit", Verdict: "clean", Cleared: []string{"TestWasFlaky"}},
			{Phase: "integration", Verdict: "regression", Regressions: []string{"TestInt"}},
			{Phase: "smoke", Verdict: "preexisting", Preexisting: []string{"smoke-flake"}},
			{Phase: "playbooks", Verdict: "clean"},
			{Phase: "some-unmapped-phase", Verdict: "regression", Regressions: []string{"ignored"}},
		},
	}
	got := bucketPhaseDiffs(cmp)

	if len(got) != 4 {
		t.Fatalf("expected 4 surfaces (structure, rules, tests, workflows), got %d: %v", len(got), got)
	}
	if got[SurfaceStructure].Verdict != VerdictClean {
		t.Errorf("structure verdict = %s", got[SurfaceStructure].Verdict)
	}
	if got[SurfaceRules].Verdict != VerdictNewFailure || len(got[SurfaceRules].NewFailures) != 1 {
		t.Errorf("rules diff = %+v", got[SurfaceRules])
	}
	// tests aggregates unit(clean+cleared) + integration(regression) + smoke(preexisting) ⇒ worst = regression.
	tests := got[SurfaceTests]
	if tests.Verdict != VerdictRegression {
		t.Errorf("tests verdict = %s, want regression", tests.Verdict)
	}
	if len(tests.Regressions) != 1 || len(tests.Preexisting) != 1 || len(tests.Cleared) != 1 {
		t.Errorf("tests aggregation lost findings: %+v", tests)
	}
	if got[SurfaceWorkflows].Verdict != VerdictClean {
		t.Errorf("workflows verdict = %s", got[SurfaceWorkflows].Verdict)
	}
	// The unmapped phase contributes to no surface.
	if _, ok := got["some-unmapped-phase"]; ok {
		t.Error("unmapped phase should not create a surface")
	}
}

// TestDiffVisuals proves the visuals surface is advisory: every per-page delta
// is the neutral `changed` tier (never a failing verdict), and it never affects
// the diff exit code. A clearly-broken render is NOT a concern here — it fails
// earlier, at smoke time, on the test/smoke surface.
func TestDiffVisuals(t *testing.T) {
	// All identical ⇒ clean.
	d := diffVisuals([]VisualDelta{
		{Page: "/", Status: "identical"},
		{Page: "/dashboard", Status: "identical"},
	})
	if d.Verdict != VerdictClean {
		t.Errorf("identical: verdict = %s, want clean", d.Verdict)
	}
	if len(d.Changed) != 0 {
		t.Errorf("identical: Changed = %v, want empty", d.Changed)
	}

	// A changed page ⇒ changed (advisory), with magnitude, and in the Changed
	// bucket — NOT NewFailures/Regressions.
	d = diffVisuals([]VisualDelta{
		{Page: "/", Status: "identical"},
		{Page: "/dashboard", Status: "changed", ChangedFraction: 0.12},
	})
	if d.Verdict != VerdictChanged {
		t.Errorf("changed page: verdict = %s, want changed", d.Verdict)
	}
	if len(d.NewFailures) != 0 || len(d.Regressions) != 0 {
		t.Errorf("changed page must not populate failure buckets: new=%v reg=%v", d.NewFailures, d.Regressions)
	}
	if len(d.Changed) != 1 || !strings.Contains(d.Changed[0], "12%") {
		t.Errorf("changed page: Changed = %v, want one entry carrying magnitude", d.Changed)
	}

	// Added/removed pages are neutral review items, not failures.
	d = diffVisuals([]VisualDelta{
		{Page: "/new", Status: "added"},
		{Page: "/gone", Status: "removed"},
	})
	if d.Verdict != VerdictChanged {
		t.Errorf("added/removed: verdict = %s, want changed", d.Verdict)
	}
	if len(d.Changed) != 2 {
		t.Errorf("added/removed: Changed = %v, want two review entries", d.Changed)
	}

	// A changed-only verdict must roll up to exit 0 (advisory).
	if got := exitCodeForVerdictTest(string(VerdictChanged)); got != 0 {
		t.Errorf("changed verdict exit code = %d, want 0 (advisory)", got)
	}
}

// exitCodeForVerdictTest mirrors the CLI's exit-code rule (regression→1,
// not-comparable→2, else 0) so the advisory contract is asserted at the API
// boundary without importing the CLI package.
func exitCodeForVerdictTest(verdict string) int {
	switch verdict {
	case string(VerdictRegression):
		return 1
	case string(VerdictNotComparable):
		return 2
	default:
		return 0
	}
}

// phaseSurface is the inverse index built from surfacePhases. Verify it covers
// every phase declared and points back to the right surface.
func TestPhaseSurfaceInverseIndex(t *testing.T) {
	for surface, phases := range surfacePhases {
		for _, p := range phases {
			if phaseSurface[p] != surface {
				t.Errorf("phase %q maps to %q, want %q", p, phaseSurface[p], surface)
			}
		}
	}
}
