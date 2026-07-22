package operations

import (
	"context"
	"errors"
	"testing"
	"time"

	"swarm-manager/internal/goals"
	"swarm-manager/internal/overview"
)

func goalWithScope(name string, priority int, status string, scope goals.Scope) goals.GoalWithScope {
	return goals.GoalWithScope{Goal: goals.Goal{Name: name, Title: name + " title", Priority: priority, Status: status}, Scope: scope}
}

func mustSnapshotBuilder(t *testing.T, cfg SnapshotBuilderConfig) *SnapshotBuilder {
	t.Helper()
	b, err := NewSnapshotBuilder(cfg)
	if err != nil {
		t.Fatalf("NewSnapshotBuilder: %v", err)
	}
	return b
}

func TestSnapshotRanksGoalsAndClassifiesScope(t *testing.T) {
	resp := &overview.OverviewResponse{Goals: []goals.GoalWithScope{
		goalWithScope("low", 10, goals.StatusActive, goals.Scope{Ready: []string{"execute/low"}, Total: 1}),
		goalWithScope("high", 1, goals.StatusActive, goals.Scope{Ready: []string{"execute/high"}, Total: 1}),
		goalWithScope("blocked", 0, goals.StatusActive, goals.Scope{Blocked: []string{"execute/wait"}, Total: 1, BlockedCount: 1}),
		goalWithScope("complete", 0, goals.StatusArchived, goals.Scope{Completed: []string{"execute/done"}, Total: 1, CompletedCount: 1}),
	}, Summary: overview.OverviewSummary{ActiveGoals: 3}}
	snap, err := mustSnapshotBuilder(t, SnapshotBuilderConfig{Overview: fakeOverviewReader{resp: resp}, Now: fixedNow()}).GetSnapshot(context.Background())
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if snap.Goals[0].Name != "high" || snap.Goals[1].Name != "low" {
		t.Fatalf("priority order = %+v", snap.Goals)
	}
	byName := map[string]RankedGoal{}
	for _, goal := range snap.Goals {
		byName[goal.Name] = goal
	}
	if byName["blocked"].Readiness != ReadinessBlocked || byName["complete"].Readiness != ReadinessComplete {
		t.Fatalf("unexpected readiness: %+v", byName)
	}
	if snap.Summary.ReadyGoals != 2 || snap.Summary.BlockedGoals != 1 {
		t.Fatalf("unexpected summary: %+v", snap.Summary)
	}
}

func TestSnapshotCacheAndErrors(t *testing.T) {
	clock := &mutableClock{t: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)}
	counting := &countingOverviewReader{resp: &overview.OverviewResponse{Goals: []goals.GoalWithScope{goalWithScope("only", 1, goals.StatusActive, goals.Scope{Ready: []string{"execute/only"}, Total: 1})}}}
	b := mustSnapshotBuilder(t, SnapshotBuilderConfig{Overview: counting, TTL: time.Minute, Now: clock.Now})
	if _, err := b.GetSnapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	clock.advance(30 * time.Second)
	if _, err := b.GetSnapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if counting.calls != 1 {
		t.Fatalf("calls = %d, want 1", counting.calls)
	}
	clock.advance(31 * time.Second)
	if _, err := b.GetSnapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if counting.calls != 2 {
		t.Fatalf("calls = %d, want 2", counting.calls)
	}
	if _, err := mustSnapshotBuilder(t, SnapshotBuilderConfig{Overview: fakeOverviewReader{err: errors.New("disk gone")}, Now: fixedNow()}).GetSnapshot(context.Background()); err == nil {
		t.Fatal("expected error")
	}
}

type mutableClock struct{ t time.Time }

func (c *mutableClock) Now() time.Time          { return c.t }
func (c *mutableClock) advance(d time.Duration) { c.t = c.t.Add(d) }

type countingOverviewReader struct {
	resp  *overview.OverviewResponse
	calls int
}

func (c *countingOverviewReader) GetOverview() (*overview.OverviewResponse, error) {
	c.calls++
	return c.resp, nil
}
