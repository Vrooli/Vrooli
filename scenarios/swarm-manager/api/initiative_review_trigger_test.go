package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestInitiativeReviewTrigger_E2E covers the end-to-end trigger chain:
//
//	POST /api/v1/backlog/{kind}/{name}/review-decide
//	       → itemTerminalHandler callback
//	       → initiativereview.Service.TriggerForItem
//	       → TriggerIfReady
//	       → initiative status flips to in_review
//
// This is the integration hook installed in routes_initiative_review.go:47.
// Without this test, a regression in SetItemTerminalHandler wiring (or a
// rename of TriggerForItem) would silently break the auto-trigger without
// breaking the individual service-level tests, and W5 (initiative review)
// would depend on a surface that isn't actually exercised.
func TestInitiativeReviewTrigger_E2E(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()
	rootDir := srv.scenarioRoot

	// 1. Create initiative owning one item.
	mustPost(t, h, "/api/v1/initiatives", map[string]any{
		"name":        "rev-trigger",
		"title":       "Review Trigger",
		"description": "Integration test initiative.",
		"status":      "active",
		"priority":    5,
		"items":       []string{"execute/solo"},
	})

	// 2. Seed the backlog item directly on disk in review_pending. We can't
	//    PATCH status=review_pending through the HTTP surface — the validator
	//    rejects it by design (only the execution/review system writes those
	//    statuses). Writing the spec.json directly simulates "execution
	//    finalized and left the item in review_pending".
	itemDir := filepath.Join(rootDir, "execute", "solo")
	if err := os.MkdirAll(itemDir, 0o755); err != nil {
		t.Fatal(err)
	}
	spec := map[string]any{
		"name":       "solo",
		"kind":       "execute",
		"title":      "Solo",
		"status":     "review_pending",
		"priority":   5,
		"initiative": "rev-trigger",
		"tags":       []string{},
	}
	specBody, _ := json.Marshal(spec)
	if err := os.WriteFile(filepath.Join(itemDir, "spec.json"), specBody, 0o644); err != nil {
		t.Fatal(err)
	}

	// 3. Call review-decide. This exercises the backlog review-decide HTTP
	//    handler, the itemTerminalHandler callback, and TriggerIfReady end-
	//    to-end. Accept maps to status=completed, which is terminal, which
	//    satisfies TriggerIfReady's "all items terminal" guard for a
	//    single-item initiative.
	mustPost(t, h, "/api/v1/backlog/execute/solo/review-decide", map[string]any{
		"decision":  "accept",
		"rationale": "shipped clean",
	})

	// 4. Verify the item landed in completed.
	itemFile := filepath.Join(itemDir, "spec.json")
	data, err := os.ReadFile(itemFile)
	if err != nil {
		t.Fatalf("read item spec: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("parse item spec: %v", err)
	}
	if s, _ := got["status"].(string); s != "completed" {
		t.Fatalf("item status = %q, want completed", s)
	}

	// 5. Verify the initiative trigger fired and the initiative is now in
	//    in_review. TriggerForItem runs synchronously from the HTTP request
	//    goroutine (see routes_initiative_review.go:47), so we don't need
	//    to poll — but we give a tiny budget in case the service spawns
	//    async work downstream.
	initFile := filepath.Join(rootDir, "initiatives", "rev-trigger", "initiative.json")
	deadline := time.Now().Add(2 * time.Second)
	var lastStatus string
	for time.Now().Before(deadline) {
		raw, readErr := os.ReadFile(initFile)
		if readErr == nil {
			var meta map[string]any
			if json.Unmarshal(raw, &meta) == nil {
				lastStatus, _ = meta["status"].(string)
				if lastStatus == "in_review" {
					return
				}
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("initiative status = %q, want in_review (trigger did not fire or failed)", lastStatus)
}

// TestInitiativeReviewTrigger_NotReady_DoesNotFlip is the negative case: when
// only some items are terminal, review-decide on one terminal item should
// NOT start initiative review. This pins the "all items terminal" guard.
func TestInitiativeReviewTrigger_NotReady_DoesNotFlip(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()
	rootDir := srv.scenarioRoot

	mustPost(t, h, "/api/v1/initiatives", map[string]any{
		"name":     "rev-partial",
		"title":    "Partial",
		"status":   "active",
		"priority": 5,
		"items":    []string{"execute/done", "execute/busy"},
	})

	// done: review_pending (will be decided below).
	seedSpec(t, rootDir, "execute", "done", map[string]any{
		"name":       "done",
		"kind":       "execute",
		"title":      "Done",
		"status":     "review_pending",
		"priority":   5,
		"initiative": "rev-partial",
		"tags":       []string{},
	})
	// busy: still in_progress — non-terminal, so initiative is NOT ready.
	seedSpec(t, rootDir, "execute", "busy", map[string]any{
		"name":       "busy",
		"kind":       "execute",
		"title":      "Busy",
		"status":     "in_progress",
		"priority":   5,
		"initiative": "rev-partial",
		"tags":       []string{},
	})

	mustPost(t, h, "/api/v1/backlog/execute/done/review-decide", map[string]any{
		"decision": "accept",
	})

	// Initiative must remain active — the trigger no-ops because a busy item
	// is still running.
	initFile := filepath.Join(rootDir, "initiatives", "rev-partial", "initiative.json")
	raw, err := os.ReadFile(initFile)
	if err != nil {
		t.Fatalf("read initiative: %v", err)
	}
	var meta map[string]any
	if err := json.Unmarshal(raw, &meta); err != nil {
		t.Fatalf("parse initiative: %v", err)
	}
	got, _ := meta["status"].(string)
	if got != "active" {
		t.Fatalf("initiative status = %q, want active (partial terminal must not trigger review)", got)
	}
}

func seedSpec(t *testing.T, rootDir, kind, name string, spec map[string]any) {
	t.Helper()
	dir := filepath.Join(rootDir, kind, name)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	body, _ := json.Marshal(spec)
	if err := os.WriteFile(filepath.Join(dir, "spec.json"), body, 0o644); err != nil {
		t.Fatalf("seed spec %s/%s: %v", kind, name, err)
	}
}
