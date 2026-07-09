package autofiler

import (
	"context"
	"testing"
	"time"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/identity"
	"swarm-manager/internal/settings"
)

type fakeBacklogReader struct {
	items []backlog.BacklogItem
	err   error
}

func (r fakeBacklogReader) LoadAll(_ []backlog.BacklogKind) ([]backlog.BacklogItem, error) {
	return append([]backlog.BacklogItem(nil), r.items...), r.err
}

type fakeTransitionCounter struct {
	counts map[string]int
}

func (c fakeTransitionCounter) CountStatusTransitionsInRange(_ context.Context, _ eventlog.EventType, toStatus string, _, _ time.Time) (int, error) {
	return c.counts[toStatus], nil
}

func TestRemainingBudgetCountsOnlyOpenAutoFiledItems(t *testing.T) {
	archivedAt := "2026-07-09T00:00:00Z"
	items := []backlog.BacklogItem{
		autoFiledItem("fix", "suggested", backlog.StatusSuggested, nil),
		autoFiledItem("fix", "ready", backlog.StatusReady, nil),
		autoFiledItem("fix", "completed", backlog.StatusCompleted, nil),
		autoFiledItem("fix", "archived", backlog.StatusSuggested, &archivedAt),
		{Name: "manual", Kind: backlog.KindFix, Status: backlog.StatusBacklog},
	}

	budget, err := RemainingBudget(fakeBacklogReader{items: items}, 3)
	if err != nil {
		t.Fatalf("RemainingBudget: %v", err)
	}
	if budget != 1 {
		t.Fatalf("budget = %d, want 1", budget)
	}
}

func TestVelocityBrake(t *testing.T) {
	now := time.Date(2026, 7, 9, 12, 0, 0, 0, time.UTC)
	cfg := settings.DefaultSettings().AutoFiler
	cfg.MinVelocityTransitions = 2

	state, err := VelocityBrake(context.Background(), fakeTransitionCounter{
		counts: map[string]int{
			string(backlog.StatusReady):     1,
			string(backlog.StatusCompleted): 1,
		},
	}, cfg, now)
	if err != nil {
		t.Fatalf("VelocityBrake: %v", err)
	}
	if state.Braked || state.Observed != 2 {
		t.Fatalf("state = %+v, want unbraked observed=2", state)
	}

	state, err = VelocityBrake(context.Background(), fakeTransitionCounter{}, cfg, now)
	if err != nil {
		t.Fatalf("VelocityBrake second: %v", err)
	}
	if !state.Braked || state.Observed != 0 {
		t.Fatalf("state = %+v, want braked observed=0", state)
	}
}

func autoFiledItem(kind, name string, status backlog.BacklogStatus, archivedAt *string) backlog.BacklogItem {
	return backlog.BacklogItem{
		Name:       name,
		Kind:       backlog.BacklogKind(kind),
		Status:     status,
		ArchivedAt: archivedAt,
		CreatedBy:  &identity.Provenance{Type: identity.TypeAgent, Source: Origin(StrategyFeaturePending, "finding-"+name)},
	}
}
