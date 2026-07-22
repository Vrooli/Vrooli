package operations

import (
	"context"
	"errors"
	"testing"
	"time"

	"swarm-manager/internal/agentactivity"
	"swarm-manager/internal/overview"
	"swarm-manager/internal/stats"
)

type fakeOverviewReader struct {
	resp *overview.OverviewResponse
	err  error
}

func (f fakeOverviewReader) GetOverview() (*overview.OverviewResponse, error) {
	if f.err != nil {
		return nil, f.err
	}
	return f.resp, nil
}

type fakeStatsReader struct {
	resp stats.StatsResponse
	err  error
}

func (f fakeStatsReader) Refresh(context.Context) error { return f.err }
func (f fakeStatsReader) GetStats() stats.StatsResponse { return f.resp }

type fakeHandoffReader struct {
	items    []DirectorHandoff
	warnings []string
}

func (f fakeHandoffReader) ReadDirectorHandoffs(context.Context) ([]DirectorHandoff, []string) {
	return f.items, f.warnings
}

func TestBriefingBuilderBuildsBoundedDeterministicBriefing(t *testing.T) {
	now := fixedNow()()
	records := []agentactivity.Record{
		{
			ActivityID: "run-attention", RunID: "run-1", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "stale-run", OwnerTitle: "Stale run", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusNeedsReview,
			RequestedAt: now.Add(-10 * time.Minute).Format(time.RFC3339),
		},
		{
			ActivityID: "run-active", RunID: "run-2", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "active-run", Purpose: agentactivity.PurposeWorkshop, Status: agentactivity.StatusRunning,
			RequestedAt: now.Add(-20 * time.Minute).Format(time.RFC3339),
		},
		{
			ActivityID: "run-finished", RunID: "run-3", OwnerType: agentactivity.OwnerBacklog, OwnerKind: "feature",
			OwnerName: "finished-run", Purpose: agentactivity.PurposeProcess, Status: agentactivity.StatusComplete,
			RequestedAt: now.Add(-40 * time.Minute).Format(time.RFC3339),
			FinishedAt:  now.Add(-5 * time.Minute).Format(time.RFC3339),
		},
	}
	agg := mustAgg(t, AggregatorConfig{
		Activities: &fakeActivityLister{records: records},
		Governance: &fakeGovernance{resp: &executionGovernanceForBriefing},
	})
	builder, err := NewBriefingBuilder(BriefingBuilderConfig{
		Aggregator: agg,
		Overview: fakeOverviewReader{resp: &overview.OverviewResponse{
			Summary:         overview.OverviewSummary{TotalItems: 42, ActiveGoals: 7},
			DependencyGraph: overview.DependencyGraph{Blocked: []string{"feature/blocked"}},
		}},
		Stats: fakeStatsReader{resp: stats.StatsResponse{Session: stats.SessionStats{ActiveSessions: 3}}},
		Handoffs: fakeHandoffReader{items: []DirectorHandoff{{
			SourcePath: "scenarios/prompt-manager/store/teams/director-swarm/members/operator/last-handoff.md",
			Title:      "operator",
			Excerpt:    "Latest handoff.",
		}}},
		Now: fixedNow(),
	})
	if err != nil {
		t.Fatalf("NewBriefingBuilder: %v", err)
	}
	briefing, err := builder.Build(context.Background(), Filters{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if briefing.Summary.ActiveActivityCount != 2 || briefing.Summary.RecentlyFinishedCount != 1 {
		t.Fatalf("summary counts = %+v", briefing.Summary)
	}
	if briefing.Summary.TotalBacklogItems != 42 || briefing.Summary.ActiveGoals != 7 || briefing.Summary.BlockedItems != 1 || briefing.Summary.ActiveSessions != 3 {
		t.Fatalf("summary enrichment = %+v", briefing.Summary)
	}
	if len(briefing.NeedsAttention) == 0 || briefing.NeedsAttention[0].Reason != "needs_review" {
		t.Fatalf("needs attention = %+v", briefing.NeedsAttention)
	}
	if len(briefing.RecommendedNextActions) == 0 || briefing.RecommendedNextActions[0].Command == "" {
		t.Fatalf("recommended actions = %+v", briefing.RecommendedNextActions)
	}
	if len(briefing.DrillDownCommands) == 0 || briefing.DrillDownCommands[0].Command != "swarm-manager operations brief --json" {
		t.Fatalf("drilldowns = %+v", briefing.DrillDownCommands)
	}
}

func TestBriefingBuilderKeepsOptionalSourceFailuresAsWarnings(t *testing.T) {
	agg := mustAgg(t, AggregatorConfig{
		Activities: &fakeActivityLister{},
		Governance: &fakeGovernance{resp: defaultGovernance()},
	})
	builder, err := NewBriefingBuilder(BriefingBuilderConfig{
		Aggregator: agg,
		Overview:   fakeOverviewReader{err: errors.New("overview down")},
		Stats:      fakeStatsReader{err: errors.New("stats down")},
		Handoffs:   fakeHandoffReader{warnings: []string{"handoff down"}},
		Now:        fixedNow(),
	})
	if err != nil {
		t.Fatalf("NewBriefingBuilder: %v", err)
	}
	briefing, err := builder.Build(context.Background(), Filters{})
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if len(briefing.Warnings) != 3 {
		t.Fatalf("warnings = %+v, want 3 optional-source warnings", briefing.Warnings)
	}
}

var executionGovernanceForBriefing = *defaultGovernance()
