package database

import (
	"context"
	"testing"
	"time"

	"agent-manager/internal/invocationreadmodel"
	"agent-manager/internal/runsignal"
)

// TestInvocationReadModelReadersHonourRunScope seeds one Agent Manager run and
// one operator session; with a scope set the cohort returns only runs in it,
// and without one both.
func TestInvocationReadModelReadersHonourRunScope(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	ctx := context.Background()
	repo := &invocationReadModelRepository{db: db}
	now := time.Date(2026, 10, 29, 12, 0, 0, 0, time.UTC)
	seed := func(id, mode string) {
		if _, err := db.ExecContext(ctx, `INSERT INTO runs (id, execution_mode, status) VALUES (?, ?, 'complete')`, id, mode); err != nil {
			t.Fatal(err)
		}
		started, ended := now, now.Add(time.Minute)
		fact := invocationreadmodel.Fact{InvocationFact: runsignal.InvocationFact{Version: "v1", CallEventID: id + "-call", ToolName: "shell", Ownership: "external", Outcome: "success", Fingerprint: id, Availability: "available"}, RunID: id, OccurredAt: now, TimeBasis: "call_event", RunStatus: "complete"}
		run := invocationreadmodel.RunFact{RunID: id, OccurredAt: ended, CreatedAt: started, StartedAt: &started, EndedAt: &ended, DurationMS: 60000, Status: "complete", TotalTokens: 10, CostTimeBasis: "ended_at", ProjectedAt: now}
		if err := repo.ReplaceProjection(ctx, []invocationreadmodel.Fact{fact}, nil, invocationreadmodel.Watermark{RunID: id, LastEventID: id + "-call", LastEventAt: now, ClassifierVersion: "v1", ProjectedAt: now}, run); err != nil {
			t.Fatal(err)
		}
	}
	seed("in-scope", "headless")
	seed("out-of-scope", "interactive")

	base := invocationreadmodel.Filter{From: &now, To: ptrTime(now.Add(time.Hour))}
	scoped := base
	scoped.Scope = invocationreadmodel.ScopeAgentManager
	check := func(name string, f invocationreadmodel.Filter, want int) {
		t.Helper()
		ids := func(c invocationreadmodel.Cohort, err error) int {
			if err != nil {
				t.Fatal(err)
			}
			return c.MatchedRuns
		}
		if got := ids(repo.Cohort(ctx, f, 10)); got != want {
			t.Errorf("%s fact cohort runs=%d want %d", name, got, want)
		}
	}
	check("scoped", scoped, 1)
	check("unscoped", base, 2)
	cohort, err := repo.Cohort(ctx, scoped, 10)
	if err != nil || len(cohort.RunIDs) != 1 || cohort.RunIDs[0] != "in-scope" {
		t.Fatalf("scoped cohort=%+v err=%v", cohort, err)
	}
}
