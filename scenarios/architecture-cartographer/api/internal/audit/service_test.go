package audit

import (
	"context"
	"testing"

	"architecture-cartographer/internal/conflicts"
	"architecture-cartographer/internal/graph"
)

func mk(t string, sev conflicts.Severity) conflicts.Conflict {
	return conflicts.Conflict{Type: t, Severity: sev}
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
	tests := []struct {
		name   string
		in     []conflicts.Conflict
		failOn conflicts.Severity
		want   Outcome
	}{
		{"only info, warn threshold → clean", []conflicts.Conflict{mk("x", conflicts.SeverityInfo)}, conflicts.SeverityWarn, OutcomeClean},
		{"warn present, warn threshold → findings", []conflicts.Conflict{mk("x", conflicts.SeverityWarn)}, conflicts.SeverityWarn, OutcomeFindings},
		{"error present, error threshold → findings", []conflicts.Conflict{mk("x", conflicts.SeverityWarn), mk("y", conflicts.SeverityError)}, conflicts.SeverityError, OutcomeFindings},
		{"warn only, error threshold → clean", []conflicts.Conflict{mk("x", conflicts.SeverityWarn)}, conflicts.SeverityError, OutcomeClean},
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
