package baseline

import "testing"

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

func TestDiffVisuals(t *testing.T) {
	base := []RunVisual{
		{Page: "/", ScreenshotRelPath: "a.png"},
		{Page: "/dashboard", ScreenshotRelPath: "b.png"},
	}

	// Page removed ⇒ regression.
	if d := diffVisuals(base, []RunVisual{{Page: "/", ScreenshotRelPath: "a.png"}}); d.Verdict != VerdictRegression {
		t.Errorf("removed page: verdict = %s, want regression", d.Verdict)
	}
	// Page added ⇒ new-failure (drift).
	added := append(append([]RunVisual{}, base...), RunVisual{Page: "/new", ScreenshotRelPath: "c.png"})
	if d := diffVisuals(base, added); d.Verdict != VerdictNewFailure {
		t.Errorf("added page: verdict = %s, want new-failure", d.Verdict)
	}
	// Identical ⇒ clean.
	if d := diffVisuals(base, base); d.Verdict != VerdictClean {
		t.Errorf("identical: verdict = %s, want clean", d.Verdict)
	}
	// Same pages, different screenshot count ⇒ drift (new-failure).
	fewerShots := []RunVisual{
		{Page: "/", ScreenshotRelPath: "a.png"},
		{Page: "/dashboard", ScreenshotRelPath: ""},
	}
	if d := diffVisuals(base, fewerShots); d.Verdict != VerdictNewFailure {
		t.Errorf("count change: verdict = %s, want new-failure", d.Verdict)
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
