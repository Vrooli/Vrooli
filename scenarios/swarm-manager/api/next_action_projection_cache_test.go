package main

import (
	"os"
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/goals"
)

func newTestProjectionCache(t *testing.T) (*nextActionProjectionCache, *backlog.Handler) {
	t.Helper()
	root := t.TempDir()
	handler := backlog.NewHandler(root, root)
	goalService := goals.NewService(goals.NewStore(root), handler.Store())
	return newNextActionProjectionCache(nextActionFeed{backlog: handler, goals: goalService}), handler
}

func saveTestItem(t *testing.T, handler *backlog.Handler, item backlog.BacklogItem) {
	t.Helper()
	if err := os.MkdirAll(handler.Store().ItemDir(item.Kind, item.Name), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := handler.Store().SaveItem(item); err != nil {
		t.Fatal(err)
	}
}

func feedActionFor(entries []nextActionFeedEntry, ref string) (backlog.NextActionProjection, bool) {
	for _, entry := range entries {
		if entry.EntityRef == ref {
			return entry.Action, true
		}
	}
	return backlog.NextActionProjection{}, false
}

// Freshness is generation-based, not time-based: the operator inbox must show
// a mutation on the very next read, never after a TTL lapses.
func TestProjectionCacheRecomputesAfterInvalidation(t *testing.T) {
	cache, handler := newTestProjectionCache(t)
	item := backlog.BacklogItem{Name: "shifter", Title: "Shifter", Kind: backlog.KindIdea, Status: backlog.StatusSuggested, Created: "2026-07-24T00:00:00Z", Updated: "2026-07-24T00:00:00Z"}
	saveTestItem(t, handler, item)

	first, err := cache.Entries(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if action, ok := feedActionFor(first, "idea/shifter"); !ok || action.ID != backlog.NextActionAcceptSuggestion {
		t.Fatalf("first read = %#v; want an accept-suggestion action", first)
	}

	// Mutate the item without announcing it: the cache must still serve the
	// generation it already computed, which is what makes the announcement
	// load-bearing rather than decorative.
	item.Status = backlog.StatusReviewPending
	saveTestItem(t, handler, item)

	stale, err := cache.Entries(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if action, _ := feedActionFor(stale, "idea/shifter"); action.ID != backlog.NextActionAcceptSuggestion {
		t.Fatalf("unannounced mutation changed the served projection: %#v", stale)
	}

	cache.Invalidate()

	fresh, err := cache.Entries(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	if action, ok := feedActionFor(fresh, "idea/shifter"); !ok || action.ID != backlog.NextActionReview {
		t.Fatalf("read after invalidation = %#v; want the mutated review action", fresh)
	}
}

// The board and the inbox read the same computation, so their answers for one
// item cannot disagree.
func TestProjectionCacheServesBoardAndFeedFromOneComputation(t *testing.T) {
	cache, handler := newTestProjectionCache(t)
	saveTestItem(t, handler, backlog.BacklogItem{Name: "shared", Title: "Shared", Kind: backlog.KindIdea, Status: backlog.StatusSuggested, Created: "2026-07-24T00:00:00Z", Updated: "2026-07-24T00:00:00Z"})

	entries, err := cache.Entries(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	actions, err := cache.ResolveNextActions(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	feedAction, ok := feedActionFor(entries, "idea/shared")
	if !ok {
		t.Fatalf("item missing from the inbox feed: %#v", entries)
	}
	if actions["idea/shared"].ID != feedAction.ID {
		t.Fatalf("board action %q disagrees with inbox action %q", actions["idea/shared"].ID, feedAction.ID)
	}
}

// A cached generation is reused rather than recomputed, which is the whole
// point of the holder: a board interaction must not project twice.
func TestProjectionCacheReusesComputationWithinAGeneration(t *testing.T) {
	cache, handler := newTestProjectionCache(t)
	saveTestItem(t, handler, backlog.BacklogItem{Name: "counted", Title: "Counted", Kind: backlog.KindIdea, Status: backlog.StatusSuggested, Created: "2026-07-24T00:00:00Z", Updated: "2026-07-24T00:00:00Z"})
	counter := &countingDecisionCounter{counts: readyDecisionCounts{items: map[string]int{}, goals: map[string]int{}}}
	cache.feed.decisions = counter

	for range 3 {
		if _, err := cache.Entries(t.Context()); err != nil {
			t.Fatal(err)
		}
		if _, err := cache.ResolveNextActions(t.Context()); err != nil {
			t.Fatal(err)
		}
	}
	if counter.scans != 1 {
		t.Fatalf("projection computed %d times across 6 reads in one generation; want 1", counter.scans)
	}

	cache.Invalidate()
	if _, err := cache.Entries(t.Context()); err != nil {
		t.Fatal(err)
	}
	if counter.scans != 2 {
		t.Fatalf("projection computed %d times after an invalidation; want 2", counter.scans)
	}
}
