package database

import (
	"context"
	"slices"
	"testing"
	"time"

	"agent-manager/internal/invocationreadmodel"

	"github.com/sirupsen/logrus"
)

// The pricing model picker ranks models from the durable invocation read
// model; placeholder models never appear as choices.
func TestStatsRepositoryRanksModelsFromDurableRuns(t *testing.T) {
	ctx := context.Background()
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	repos := NewRepositories(db, logrus.New())
	readModel := repos.InvocationReadModel.(invocationreadmodel.RunStore)
	now := time.Date(2026, 1, 2, 12, 0, 0, 0, time.UTC)
	for i, run := range []struct {
		model string
		at    time.Time
	}{
		{"popular-old", now.Add(-3 * time.Hour)},
		{"popular-old", now.Add(-2 * time.Hour)},
		{"popular-old", now.Add(-90 * time.Minute)},
		{"newest", now},
		{"stale", now.Add(-48 * time.Hour)},
		{"unknown", now},
		{"", now},
	} {
		if err := readModel.ReplaceRun(ctx, invocationreadmodel.RunFact{RunID: "rank-run-" + string(rune('a'+i)), OccurredAt: run.at, CreatedAt: run.at, Status: "complete", RunnerType: "codex", Model: run.model, ProjectedAt: now}); err != nil {
			t.Fatalf("project run fact: %v", err)
		}
	}

	popular, err := repos.Stats.GetPopularModels(ctx, now.Add(-24*time.Hour), 10)
	if err != nil || !slices.Equal(popular, []string{"popular-old", "newest"}) {
		t.Fatalf("popular models=%v err=%v, want most-used first within the window", popular, err)
	}
	recent, err := repos.Stats.GetRecentModels(ctx, 10)
	if err != nil || !slices.Equal(recent, []string{"newest", "popular-old", "stale"}) {
		t.Fatalf("recent models=%v err=%v, want most recent use first", recent, err)
	}
	if limited, err := repos.Stats.GetRecentModels(ctx, 1); err != nil || len(limited) != 1 {
		t.Fatalf("limited recent models=%v err=%v", limited, err)
	}
}
