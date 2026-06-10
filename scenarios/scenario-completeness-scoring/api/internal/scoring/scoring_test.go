package scoring

import (
	"strings"
	"testing"

	architecturev1 "github.com/vrooli/vrooli/packages/proto/gen/go/architecture/v1"

	"scenario-completeness-scoring/internal/signals"
)

func cleanSnapshot() signals.Snapshot {
	phases := map[string]signals.PhaseResult{}
	for _, p := range []string{
		"structure", "standards", "docs", "business", "unit",
		"integration", "smoke", "lint", "security", "tidiness",
	} {
		phases[p] = signals.PhaseResult{Status: "passed", HasFindings: true}
	}
	return signals.Snapshot{
		Scenario: "fixture",
		Category: "utility",
		Requirements: signals.RequirementsSignals{
			Collected:      true,
			Total:          25,
			Passing:        25,
			TargetsTotal:   12,
			TargetsPassing: 12,
			WithValidation: 25,
			AvgDepth:       3.0,
		},
		Phases: signals.PhaseSignals{Collected: true, Phases: phases},
		UI: signals.UISignals{
			Collected:       true,
			IsTemplate:      false,
			FileCount:       40,
			APIEndpoints:    9,
			APIBeyondHealth: 8,
			HasRouting:      true,
			RouteCount:      5,
			TotalLOC:        1500,
		},
	}
}

func TestCompositeFullMarksOnCleanSnapshot(t *testing.T) {
	comp := computeComposite(cleanSnapshot())
	if comp.Score != 100 {
		t.Fatalf("score = %d, want 100; groups: %+v", comp.Score, comp.Groups)
	}
	if comp.Classification != "production_ready" {
		t.Fatalf("classification = %q", comp.Classification)
	}
}

func TestClassificationBoundaries(t *testing.T) {
	cases := []struct {
		score int
		want  string
	}{
		{100, "production_ready"},
		{96, "production_ready"},
		{95, "nearly_ready"},
		{81, "nearly_ready"},
		{80, "mostly_complete"},
		{61, "mostly_complete"},
		{60, "functional_incomplete"},
		{41, "functional_incomplete"},
		{40, "foundation_laid"},
		{21, "foundation_laid"},
		{20, "early_stage"},
		{0, "early_stage"},
	}
	for _, c := range cases {
		got, _ := classify(c.score)
		if got != c.want {
			t.Fatalf("classify(%d) = %q, want %q", c.score, got, c.want)
		}
	}
}

func TestTemplateUIScoresZeroTemplatePoints(t *testing.T) {
	snap := cleanSnapshot()
	snap.UI.IsTemplate = true
	comp := computeComposite(snap)

	for _, g := range comp.Groups {
		if g.ID != "ui" {
			continue
		}
		for _, m := range g.Metrics {
			if m.ID == "template_check" && m.Points != 0 {
				t.Fatalf("template points = %v, want 0", m.Points)
			}
		}
	}
	if comp.Score >= 100 {
		t.Fatalf("template UI should cost points, got %d", comp.Score)
	}
}

func TestDeriveMaturityCleanLadder(t *testing.T) {
	mat := deriveMaturity(cleanSnapshot())
	if !mat.LadderClean {
		t.Fatalf("expected clean ladder, working rung = %q (dims %+v)", mat.WorkingRung, mat.Dimensions)
	}
	if mat.SatisfiedThrough == "" {
		t.Fatal("expected satisfied-through rung label")
	}
	if !mat.BuildPassing {
		t.Fatal("expected build passing (unit passed)")
	}
}

func TestDeriveMaturityStandardsErrorsHoldR1(t *testing.T) {
	snap := cleanSnapshot()
	pr := snap.Phases.Phases["standards"]
	pr.Status = "failed"
	pr.Findings = []*architecturev1.ArchitectureFinding{
		{
			Source:   architecturev1.FindingSource_FINDING_SOURCE_STANDARDS,
			Severity: architecturev1.FindingSeverity_FINDING_SEVERITY_ERROR,
		},
	}
	snap.Phases.Phases["standards"] = pr

	mat := deriveMaturity(snap)
	if mat.LadderClean {
		t.Fatal("ladder should not be clean with standards errors")
	}
	// standards governs R0 in the shared ladder, so an error there holds R0.
	if mat.WorkingRung == "" {
		t.Fatal("expected a working rung")
	}
	found := false
	for _, d := range mat.Dimensions {
		if d.Dimension == "standards" {
			found = true
			if d.ErrorPlus != 1 || d.Approximate {
				t.Fatalf("standards dimension = %+v", d)
			}
		}
	}
	if !found {
		t.Fatalf("standards dimension missing: %+v", mat.Dimensions)
	}
}

func TestDeriveMaturityApproximatesWithoutFindings(t *testing.T) {
	snap := cleanSnapshot()
	// Older writer shape: failed phase, no findings key.
	snap.Phases.Phases["standards"] = signals.PhaseResult{Status: "failed"}

	mat := deriveMaturity(snap)
	for _, d := range mat.Dimensions {
		if d.Dimension == "standards" {
			if !d.Approximate || d.ErrorPlus != 1 {
				t.Fatalf("standards dimension = %+v, want approximate error", d)
			}
			return
		}
	}
	t.Fatalf("standards dimension missing: %+v", mat.Dimensions)
}

func TestDeriveMaturityNeverTestedIsHonest(t *testing.T) {
	snap := signals.Snapshot{
		Scenario:     "fixture",
		Category:     "utility",
		Requirements: signals.RequirementsSignals{Collected: true},
		Phases:       signals.PhaseSignals{Collected: false, Phases: map[string]signals.PhaseResult{}},
	}
	mat := deriveMaturity(snap)
	if mat.LadderClean {
		t.Fatal("never-tested scenario must not present a clean ladder")
	}
	if mat.BuildPassing {
		t.Fatal("no unit evidence must not read as build passing")
	}
}

func TestRecommendationsPrioritizedAndImpactful(t *testing.T) {
	snap := cleanSnapshot()
	snap.UI.IsTemplate = true
	snap.Requirements.Passing = 10 // 40% pass rate
	snap.Requirements.WithValidation = 10

	comp := computeComposite(snap)
	mat := deriveMaturity(snap)
	recs := buildRecommendations(snap, comp, mat)
	if len(recs) == 0 {
		t.Fatal("expected recommendations")
	}
	lastRank := -1
	for _, r := range recs {
		rank := priorityRank(r.Priority)
		if rank < lastRank {
			t.Fatalf("recommendations not priority-ordered: %+v", recs)
		}
		lastRank = rank
		// Rung blockers carry zero composite impact by design; everything
		// else must state its payoff.
		if r.Impact <= 0 && !strings.Contains(r.Description, "blocks rung") {
			t.Fatalf("recommendation with non-positive impact: %+v", r)
		}
	}

	plan := buildActionPlan(comp, recs)
	if len(plan) == 0 {
		t.Fatal("expected an action plan")
	}
	total := float64(comp.Score)
	for _, p := range plan {
		total += p.EstimatedPoints
	}
	if total > 100.5 {
		t.Fatalf("action plan projects past 100: %v", total)
	}
}

func TestCleanSnapshotYieldsNoRecommendations(t *testing.T) {
	snap := cleanSnapshot()
	comp := computeComposite(snap)
	mat := deriveMaturity(snap)
	if recs := buildRecommendations(snap, comp, mat); len(recs) != 0 {
		t.Fatalf("clean snapshot produced recommendations: %+v", recs)
	}
}
