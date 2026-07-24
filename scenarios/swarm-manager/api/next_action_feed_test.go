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

// countingDecisionCounter records how many whole-store proposal scans one
// projection performs.
type countingDecisionCounter struct {
	scans  int
	counts readyDecisionCounts
}

func (c *countingDecisionCounter) countReadyDecisions() (readyDecisionCounts, error) {
	c.scans++
	return c.counts, nil
}

// The proposal store is scanned once per request, never once per entity. This
// is the invariant that keeps the operator inbox linear: a per-item scan made
// the feed cost items x whole-store reads, which dominated request time at
// production data scale.
func TestNextActionFeedScansProposalStoreOncePerRequest(t *testing.T) {
	root := t.TempDir()
	handler := backlog.NewHandler(root, root)
	for _, name := range []string{"first", "second", "third", "fourth"} {
		item := backlog.BacklogItem{Name: name, Title: name, Kind: backlog.KindIdea, Status: backlog.StatusSuggested, Created: "2026-07-24T00:00:00Z", Updated: "2026-07-24T00:00:00Z"}
		if err := os.MkdirAll(handler.Store().ItemDir(item.Kind, item.Name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := handler.Store().SaveItem(item); err != nil {
			t.Fatal(err)
		}
	}
	goalService := goals.NewService(goals.NewStore(root), handler.Store())
	if _, err := goalService.Create(goals.CreateRequest{Name: "scoped-goal", Title: "Scoped goal", Priority: 5, Targets: []string{"idea/first", "idea/second"}}); err != nil {
		t.Fatal(err)
	}
	counter := &countingDecisionCounter{counts: readyDecisionCounts{items: map[string]int{"idea/first": 1}, goals: map[string]int{}}}

	entries, err := (nextActionFeed{backlog: handler, goals: goalService, decisions: counter}).resolve(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if counter.scans != 1 {
		t.Fatalf("proposal store scanned %d times for a 4-item feed; want exactly 1", counter.scans)
	}

	// The pre-resolved counts must still reach the per-item projection.
	var decided int
	for _, entry := range entries {
		if entry.EntityRef == "idea/first" && entry.Action.ID == backlog.NextActionDecide {
			decided++
		}
	}
	if decided != 1 {
		t.Fatalf("entries = %#v; want idea/first to resolve to a decide action from the pre-resolved counts", entries)
	}
}
