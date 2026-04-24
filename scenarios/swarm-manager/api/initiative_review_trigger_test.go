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

// TestInitiativeReviewTrigger_AllFailed_Triggers pins that "all terminal" is
// the gate — not "all accepted." An initiative whose every member item ends
// in `failed` should still enter review so the user can decide whether the
// initiative as a whole is dead, partially salvageable, or worth a
// needs_followup verdict.
func TestInitiativeReviewTrigger_AllFailed_Triggers(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()
	rootDir := srv.scenarioRoot

	mustPost(t, h, "/api/v1/initiatives", map[string]any{
		"name":     "rev-all-failed",
		"title":    "All Failed",
		"status":   "active",
		"priority": 5,
		"items":    []string{"execute/broke1", "execute/broke2"},
	})

	seedSpec(t, rootDir, "execute", "broke1", map[string]any{
		"name": "broke1", "kind": "execute", "title": "Broke 1",
		"status":   "failed",
		"priority": 5, "initiative": "rev-all-failed", "tags": []string{},
	})
	seedSpec(t, rootDir, "execute", "broke2", map[string]any{
		"name": "broke2", "kind": "execute", "title": "Broke 2",
		"status":   "review_pending",
		"priority": 5, "initiative": "rev-all-failed", "tags": []string{},
	})

	// Fail the last review-pending item — now every member is terminal.
	mustPost(t, h, "/api/v1/backlog/execute/broke2/review-decide", map[string]any{
		"decision":  "fail",
		"rationale": "implementation regressed",
	})

	assertInitiativeStatusEventually(t, rootDir, "rev-all-failed", "in_review")
}

// TestInitiativeReviewTrigger_MixedTerminals_Triggers covers the common case
// where a multi-item initiative lands with a mix of verdicts (completed,
// failed, needs_followup, archived). All of those count as terminal for the
// readiness gate, so review must still fire once the last one lands.
func TestInitiativeReviewTrigger_MixedTerminals_Triggers(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()
	rootDir := srv.scenarioRoot

	mustPost(t, h, "/api/v1/initiatives", map[string]any{
		"name":     "rev-mixed",
		"title":    "Mixed Terminals",
		"status":   "active",
		"priority": 5,
		"items":    []string{"execute/done", "execute/dead", "execute/half", "execute/ghost"},
	})

	now := time.Now().UTC().Format(time.RFC3339)
	seedSpec(t, rootDir, "execute", "done", map[string]any{
		"name": "done", "kind": "execute", "title": "Done",
		"status":   "completed",
		"priority": 5, "initiative": "rev-mixed", "tags": []string{},
	})
	seedSpec(t, rootDir, "execute", "dead", map[string]any{
		"name": "dead", "kind": "execute", "title": "Dead",
		"status":   "failed",
		"priority": 5, "initiative": "rev-mixed", "tags": []string{},
	})
	seedSpec(t, rootDir, "execute", "half", map[string]any{
		"name": "half", "kind": "execute", "title": "Half",
		"status":   "needs_followup",
		"priority": 5, "initiative": "rev-mixed", "tags": []string{},
	})
	// Archived items count as terminal for rollup purposes (see
	// initiativereview/service.go findNonTerminalItems).
	seedSpec(t, rootDir, "execute", "ghost", map[string]any{
		"name": "ghost", "kind": "execute", "title": "Ghost",
		"status": "backlog", "archived_at": now,
		"priority": 5, "initiative": "rev-mixed", "tags": []string{},
	})

	// Nothing to decide — trigger via the manual-trigger endpoint so the
	// test doesn't depend on decide-driven transitions.
	mustPost(t, h, "/api/v1/initiatives/rev-mixed/review/trigger", map[string]any{})

	assertInitiativeStatusEventually(t, rootDir, "rev-mixed", "in_review")
}

// TestInitiativeReviewTrigger_FeedbackLockBlocks pins the coordination
// documented in internal/initiativereview/doc.go: if a feedback round
// currently holds the `.feedback-lock`, a subsequent review-start attempt
// must fail cleanly and leave the initiative in `active`. The user must
// finish or dismiss the feedback round before review will start — the
// alternative (racing two concurrent agents against the same initiative)
// is a real footgun.
func TestInitiativeReviewTrigger_FeedbackLockBlocks(t *testing.T) {
	srv := newTestServer(t)
	h := srv.Handler()
	rootDir := srv.scenarioRoot

	mustPost(t, h, "/api/v1/initiatives", map[string]any{
		"name":     "rev-locked",
		"title":    "Locked",
		"status":   "active",
		"priority": 5,
		"items":    []string{"execute/solo-locked"},
	})

	seedSpec(t, rootDir, "execute", "solo-locked", map[string]any{
		"name": "solo-locked", "kind": "execute", "title": "Solo Locked",
		"status":   "review_pending",
		"priority": 5, "initiative": "rev-locked", "tags": []string{},
	})

	// Simulate an in-flight feedback round holding the lock.
	initDir := filepath.Join(rootDir, "initiatives", "rev-locked")
	if err := os.MkdirAll(initDir, 0o755); err != nil {
		t.Fatal(err)
	}
	holder := map[string]any{
		"run_id":          "feedback-holder-1",
		"purpose":         "feedback",
		"round_number":    1,
		"acquired_at":     time.Now().UTC().Format(time.RFC3339),
		"acquired_by":     "swarm-manager:feedback-test",
		"initiative_name": "rev-locked",
	}
	body, _ := json.Marshal(holder)
	if err := os.WriteFile(filepath.Join(initDir, ".feedback-lock"), body, 0o644); err != nil {
		t.Fatalf("seed feedback lock: %v", err)
	}

	// Decide the item — review trigger fires internally, hits the lock,
	// and logs a warning. The HTTP call itself must still succeed (review
	// is a downstream consequence, not a precondition for decide).
	mustPost(t, h, "/api/v1/backlog/execute/solo-locked/review-decide", map[string]any{
		"decision": "accept",
	})

	// Initiative must remain active — the lock blocked review start.
	initFile := filepath.Join(initDir, "initiative.json")
	// Give any async settling a tiny budget; the assertion is "did NOT
	// flip," so we sample a few times to reduce flake risk.
	deadline := time.Now().Add(500 * time.Millisecond)
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(initFile)
		if err == nil {
			var meta map[string]any
			if json.Unmarshal(raw, &meta) == nil {
				if s, _ := meta["status"].(string); s != "active" {
					t.Fatalf("initiative status = %q, want active (feedback lock must block review start)", s)
				}
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
}

func assertInitiativeStatusEventually(t *testing.T, rootDir, name, want string) {
	t.Helper()
	initFile := filepath.Join(rootDir, "initiatives", name, "initiative.json")
	deadline := time.Now().Add(2 * time.Second)
	var lastStatus string
	for time.Now().Before(deadline) {
		raw, err := os.ReadFile(initFile)
		if err == nil {
			var meta map[string]any
			if json.Unmarshal(raw, &meta) == nil {
				lastStatus, _ = meta["status"].(string)
				if lastStatus == want {
					return
				}
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatalf("initiative %q status = %q, want %q", name, lastStatus, want)
}
