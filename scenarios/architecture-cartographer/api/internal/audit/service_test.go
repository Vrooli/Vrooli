package audit

import (
	"context"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/graph"
)

func mk(t string, sev conflicts.Severity) conflicts.Conflict {
	return conflicts.Conflict{Type: t, Severity: sev, FindingClass: conflicts.FindingClassDeterministic}
}

func TestApplyFilters_IncludeAndExclude(t *testing.T) {
	in := []conflicts.Conflict{
		mk("cycle", conflicts.SeverityError),
		mk("mislocated_file", conflicts.SeverityWarn),
		mk("convergence_drift", conflicts.SeverityWarn),
		mk("authority_fallback", conflicts.SeverityInfo),
	}
	out := applyFilters(in, []string{"mislocated_file", "convergence_drift"}, []string{"convergence_drift"})
	if len(out) != 1 || out[0].Type != "mislocated_file" {
		t.Fatalf("include+exclude composition wrong: %+v", out)
	}
}

func TestApplyFilters_EmptyFiltersPassesThrough(t *testing.T) {
	in := []conflicts.Conflict{mk("cycle", conflicts.SeverityError)}
	out := applyFilters(in, nil, nil)
	if len(out) != 1 {
		t.Fatalf("nil filters must pass through; got %+v", out)
	}
}

func TestDecideOutcome_FailsOnAtOrAboveThreshold(t *testing.T) {
	t.Setenv("INTENT_ALIGNMENT_GATE", "")
	tests := []struct {
		name   string
		in     []conflicts.Conflict
		failOn conflicts.Severity
		want   Outcome
	}{
		{"only info, warn threshold → clean", []conflicts.Conflict{mk("x", conflicts.SeverityInfo)}, conflicts.SeverityWarn, OutcomeClean},
		{"warn present, warn threshold → clean", []conflicts.Conflict{mk("x", conflicts.SeverityWarn)}, conflicts.SeverityWarn, OutcomeClean},
		{"error present, error threshold → findings", []conflicts.Conflict{mk("x", conflicts.SeverityWarn), mk("y", conflicts.SeverityError)}, conflicts.SeverityError, OutcomeFindings},
		{"warn only, error threshold → clean", []conflicts.Conflict{mk("x", conflicts.SeverityWarn)}, conflicts.SeverityError, OutcomeClean},
		{"heuristic error → clean", []conflicts.Conflict{{Type: "x", Severity: conflicts.SeverityError, FindingClass: conflicts.FindingClassHeuristic}}, conflicts.SeverityError, OutcomeClean},
		{"empty → clean", nil, conflicts.SeverityWarn, OutcomeClean},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := decideOutcome(tc.in, tc.failOn); got != tc.want {
				t.Fatalf("got %s want %s", got, tc.want)
			}
		})
	}
}

func TestDecideOutcome_IntentAlignmentGate(t *testing.T) {
	intentFinding := []conflicts.Conflict{mk("intent.req_unowned_domain", conflicts.SeverityError)}

	t.Setenv("INTENT_ALIGNMENT_GATE", "")
	if got := decideOutcome(intentFinding, conflicts.SeverityError); got != OutcomeClean {
		t.Fatalf("default advisory intent gate outcome = %s, want clean", got)
	}

	t.Setenv("INTENT_ALIGNMENT_GATE", "off")
	if got := decideOutcome(intentFinding, conflicts.SeverityError); got != OutcomeClean {
		t.Fatalf("off intent gate outcome = %s, want clean", got)
	}

	t.Setenv("INTENT_ALIGNMENT_GATE", "strict")
	if got := decideOutcome(intentFinding, conflicts.SeverityError); got != OutcomeFindings {
		t.Fatalf("strict intent gate outcome = %s, want findings", got)
	}

	t.Setenv("INTENT_ALIGNMENT_GATE", "invalid")
	if got := decideOutcome(intentFinding, conflicts.SeverityError); got != OutcomeClean {
		t.Fatalf("invalid intent gate outcome = %s, want clean", got)
	}
}

func TestCountBySeverityAndType(t *testing.T) {
	in := []conflicts.Conflict{
		mk("cycle", conflicts.SeverityError),
		mk("cycle", conflicts.SeverityError),
		mk("mislocated_file", conflicts.SeverityWarn),
	}
	bs := countBySeverity(in)
	if bs["error"] != 2 || bs["warn"] != 1 {
		t.Fatalf("severity bucketing wrong: %+v", bs)
	}
	bt := countByType(in)
	if bt["cycle"] != 2 || bt["mislocated_file"] != 1 {
		t.Fatalf("type bucketing wrong: %+v", bt)
	}
}

type fakeVerdictProvider struct {
	verdicts []conflicts.Verdict
}

func (f fakeVerdictProvider) VerdictsFor(_ context.Context, _ string, _ []graph.Chunk) ([]conflicts.Verdict, error) {
	return append([]conflicts.Verdict(nil), f.verdicts...), nil
}

func (f fakeVerdictProvider) ContentVerdictsFor(ctx context.Context, scenario string, chunks []graph.Chunk) ([]conflicts.Verdict, error) {
	return f.VerdictsFor(ctx, scenario, chunks)
}

func TestCoverageSummary_BucketsVerdictTiers(t *testing.T) {
	snap := graph.GraphSnapshot{Files: []graph.FileNode{
		{ID: "a", Path: "api/internal/orders/a.go"},
		{ID: "b", Path: "api/internal/orders/b.go"},
		{ID: "c", Path: "api/internal/orders/c.go"},
		{ID: "d", Path: "api/internal/orders/d.go"},
	}}
	got, err := coverageSummary(context.Background(), "demo", snap, fakeVerdictProvider{verdicts: []conflicts.Verdict{
		{Tier: "auto_place", TopDomain: "orders"},
		{Tier: "suggest", TopDomain: "orders"},
		{Tier: "conflict", TopDomain: "billing"},
		{Tier: "conflict", AllAbstained: true},
	}})
	if err != nil {
		t.Fatalf("coverageSummary: %v", err)
	}
	if got.TotalFiles != 4 {
		t.Fatalf("total files = %d, want 4", got.TotalFiles)
	}
	if got.AutoPlace.Count != 1 || got.Suggest.Count != 1 || got.Conflict.Count != 1 || got.AllAbstained.Count != 1 {
		t.Fatalf("unexpected bucket counts: %+v", got)
	}
	for name, bucket := range map[string]CoverageBucket{
		"auto_place":    got.AutoPlace,
		"suggest":       got.Suggest,
		"conflict":      got.Conflict,
		"all_abstained": got.AllAbstained,
	} {
		if bucket.Percent < 24.9 || bucket.Percent > 25.1 {
			t.Fatalf("%s percent = %.2f, want 25%%", name, bucket.Percent)
		}
	}
}

func TestCoverageSummary_MissingVerdictsCountAsAllAbstained(t *testing.T) {
	snap := graph.GraphSnapshot{Files: []graph.FileNode{
		{ID: "a", Path: "api/internal/orders/a.go"},
		{ID: "b", Path: "api/internal/orders/b.go"},
	}}
	got, err := coverageSummary(context.Background(), "demo", snap, fakeVerdictProvider{verdicts: []conflicts.Verdict{
		{Tier: "auto_place", TopDomain: "orders"},
	}})
	if err != nil {
		t.Fatalf("coverageSummary: %v", err)
	}
	if got.AutoPlace.Count != 1 || got.AllAbstained.Count != 1 {
		t.Fatalf("missing verdict should count as all_abstained: %+v", got)
	}
}

func TestCoverageSummary_NoProviderCountsAllAbstained(t *testing.T) {
	snap := graph.GraphSnapshot{Files: []graph.FileNode{
		{ID: "a", Path: "api/internal/orders/a.go"},
		{ID: "b", Path: "api/internal/orders/b.go"},
	}}
	got, err := coverageSummary(context.Background(), "demo", snap, nil)
	if err != nil {
		t.Fatalf("coverageSummary: %v", err)
	}
	if got.AllAbstained.Count != 2 || got.AllAbstained.Percent != 100 {
		t.Fatalf("nil provider should count all files as all_abstained: %+v", got)
	}
}

func TestScoreCategoriesBuildsAdvisoryMatrix(t *testing.T) {
	categories := scoreCategories(CoverageSummary{
		TotalFiles: 4,
		AutoPlace:  CoverageBucket{Count: 1, Percent: 25},
		Suggest:    CoverageBucket{Count: 1, Percent: 25},
	}, []conflicts.Conflict{
		{
			ID:           "b",
			StableID:     "b",
			Type:         "layering",
			Subtype:      "substrate-imports-product",
			Severity:     conflicts.SeverityBlocker,
			FindingClass: conflicts.FindingClassDeterministic,
			Locations:    []string{"api/internal/config/config.go"},
		},
		{
			ID:           "n",
			StableID:     "n",
			Type:         "naming",
			Severity:     conflicts.SeverityWarn,
			FindingClass: conflicts.FindingClassHeuristic,
			Locations:    []string{"api/internal/util"},
		},
	}, "high")

	if len(categories) != 6 {
		t.Fatalf("category count = %d, want 6", len(categories))
	}
	byKey := map[string]AuditCategory{}
	for _, category := range categories {
		byKey[category.Key] = category
	}
	if got := byKey["placement_legibility"].Score; got != 0.5 {
		t.Fatalf("placement score = %.2f, want 0.50", got)
	}
	if got := byKey["authority"].Score; got != 1 {
		t.Fatalf("authority score = %.2f, want 1.00", got)
	}
	if got := byKey["boundary_cleanliness"]; got.Score >= 1 || len(got.TopItems) != 1 || got.TopItems[0].Type != "layering" {
		t.Fatalf("boundary category did not surface layering item: %+v", got)
	}
	if got := byKey["naming_clarity"]; got.Score >= 1 || len(got.TopItems) != 1 || got.TopItems[0].FindingClass != conflicts.FindingClassHeuristic {
		t.Fatalf("naming category did not surface heuristic item: %+v", got)
	}
	if got := byKey["intent_coverage"]; got.Score != 1 || len(got.TopItems) != 0 {
		t.Fatalf("intent coverage category should be clean without intent findings: %+v", got)
	}
}

func TestScoreCategoriesMissingAuthorityScoresZero(t *testing.T) {
	categories := scoreCategories(CoverageSummary{}, nil, "missing")
	for _, category := range categories {
		if category.Key == "authority" && category.Score != 0 {
			t.Fatalf("missing authority score = %.2f, want 0", category.Score)
		}
	}
}
