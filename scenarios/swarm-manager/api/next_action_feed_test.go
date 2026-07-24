package main

import (
	"os"
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
)

// [REQ:SWM-P0-010] The cross-entity feed uses the same resolver contract as
// per-item actions and ranks judgment/planning work before housekeeping.
func TestNextActionFeedRanksGoalAndBacklogEntries(t *testing.T) {
	root := t.TempDir()
	handler := backlog.NewHandler(root, root)
	item := backlog.BacklogItem{Name: "suggestion", Title: "Accept me", Kind: backlog.KindIdea, Status: backlog.StatusSuggested, Priority: 9, Created: "2026-07-24T00:00:00Z", Updated: "2026-07-24T00:00:00Z"}
	if err := os.MkdirAll(handler.Store().ItemDir(item.Kind, item.Name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := handler.Store().SaveItem(item); err != nil {
		t.Fatal(err)
	}
	goalService := goals.NewService(goals.NewStore(root), handler.Store())
	if _, err := goalService.Create(goals.CreateRequest{Name: "empty-goal", Title: "Empty goal", Priority: 5}); err != nil {
		t.Fatal(err)
	}
	entries, err := (nextActionFeed{backlog: handler, goals: goalService}).resolve(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %#v", entries)
	}
	if entries[0].EntityKind != "goal" || entries[0].Action.ID != backlog.NextActionPlanGoal {
		t.Fatalf("first entry = %#v; want goal plan action", entries[0])
	}
	if entries[1].EntityRef != "idea/suggestion" || entries[1].Action.ID != backlog.NextActionAcceptSuggestion {
		t.Fatalf("second entry = %#v; want suggested backlog item", entries[1])
	}
}
