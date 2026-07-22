package main

import (
	"context"
	"strings"
	"testing"

	"swarm-manager/internal/backlog"
	"swarm-manager/internal/records"
)

// captureIndexer records every IndexRecord call so a test can assert the
// auto-captured record was born indexed (not an empty, unindexed stub).
type captureIndexer struct{ indexed []records.Record }

func (c *captureIndexer) IndexRecord(_ context.Context, r records.Record) error {
	c.indexed = append(c.indexed, r)
	return nil
}

func newCaptureTestAdapter(t *testing.T) (*recordCaptureAdapter, *captureIndexer) {
	t.Helper()
	idx := &captureIndexer{}
	svc := records.NewService(records.NewFileStore(t.TempDir()), idx, nil)
	return newRecordCaptureAdapter(svc), idx
}

// TestCaptureAdapter_AcceptCreatesFilledIndexedRecord is the core Phase 3 proof:
// a terminal accept produces ONE filled, indexed, non-stub record whose
// trigger/approach are the item's title/description, scenario is derived from
// the acceptance globs, and milestone is linked back.
func TestCaptureAdapter_AcceptCreatesFilledIndexedRecord(t *testing.T) {
	a, idx := newCaptureTestAdapter(t)
	id, err := a.CreateBacklogRecord(context.Background(), backlog.BacklogRecordRequest{
		Kind: "fix", Name: "silence-race",
		Title:       "Fix the silence race",
		Description: "Debounce VAD stop events behind a 300ms timer.",
		AcceptanceAllow: []string{
			"scenarios/web-console/ui/**",
			"scenarios/web-console/api/**",
		},
		Milestone: "voice-reliability",
		Status:     backlog.StatusCompleted,
		DecidedBy:  "agent-x",
	})
	if err != nil {
		t.Fatalf("CreateBacklogRecord: %v", err)
	}
	if id == "" {
		t.Fatal("expected a record id")
	}
	if len(idx.indexed) != 1 {
		t.Fatalf("expected exactly 1 indexed record (born indexed, not a stub), got %d", len(idx.indexed))
	}
	r := idx.indexed[0]
	if r.ID != id {
		t.Errorf("indexed id = %q, want %q", r.ID, id)
	}
	if r.Stub {
		t.Error("auto-captured record must be non-stub (filled at birth)")
	}
	if r.Kind != records.KindFix {
		t.Errorf("kind = %q, want fix", r.Kind)
	}
	if r.Scenario != "web-console" {
		t.Errorf("scenario = %q, want web-console (derived from acceptance globs)", r.Scenario)
	}
	if r.Trigger != "Fix the silence race" {
		t.Errorf("trigger = %q, want the item title", r.Trigger)
	}
	if r.Approach != "Debounce VAD stop events behind a 300ms timer." {
		t.Errorf("approach = %q, want the item description", r.Approach)
	}
	if r.MilestoneID != "voice-reliability" {
		t.Errorf("milestone_id = %q, want voice-reliability", r.MilestoneID)
	}
	if r.BacklogRef != "fix/silence-race" {
		t.Errorf("backlog_ref = %q, want fix/silence-race", r.BacklogRef)
	}
	if r.Outcome != records.OutcomeShipped {
		t.Errorf("outcome = %q, want shipped", r.Outcome)
	}
	if r.CreatedBy != "agent-x" {
		t.Errorf("created_by = %q, want agent-x", r.CreatedBy)
	}
}

// TestCaptureAdapter_FollowupCapturesNothing: a followup item isn't done, so no
// record (and nothing indexed).
func TestCaptureAdapter_FollowupCapturesNothing(t *testing.T) {
	a, idx := newCaptureTestAdapter(t)
	id, err := a.CreateBacklogRecord(context.Background(), backlog.BacklogRecordRequest{
		Kind: "fix", Name: "later", Title: "Do later",
		Status: backlog.StatusNeedsFollowup, DecidedBy: "user",
	})
	if err != nil {
		t.Fatalf("CreateBacklogRecord: %v", err)
	}
	if id != "" {
		t.Errorf("followup must not capture a record, got id %q", id)
	}
	if len(idx.indexed) != 0 {
		t.Errorf("followup must index nothing, got %d", len(idx.indexed))
	}
}

// TestCaptureAdapter_FailMapsToAbandoned: a fail decision records an abandoned
// outcome (still a searchable record — failures are lessons too).
func TestCaptureAdapter_FailMapsToAbandoned(t *testing.T) {
	a, idx := newCaptureTestAdapter(t)
	_, err := a.CreateBacklogRecord(context.Background(), backlog.BacklogRecordRequest{
		Kind: "execute", Name: "doomed", Title: "Attempted the thing",
		Status: backlog.StatusFailed, DecidedBy: "user",
	})
	if err != nil {
		t.Fatalf("CreateBacklogRecord: %v", err)
	}
	if len(idx.indexed) != 1 {
		t.Fatalf("expected 1 indexed record, got %d", len(idx.indexed))
	}
	if got := idx.indexed[0].Outcome; got != records.OutcomeAbandoned {
		t.Errorf("outcome = %q, want abandoned", got)
	}
}

// TestCaptureAdapter_EmptyItemGetsTemplatedApproach: a bare item (no title AND
// no description) still earns a valid, searchable floor record via a templated
// approach, and the scenario falls back to swarm-manager when no glob names one.
func TestCaptureAdapter_EmptyItemGetsTemplatedApproach(t *testing.T) {
	a, idx := newCaptureTestAdapter(t)
	id, err := a.CreateBacklogRecord(context.Background(), backlog.BacklogRecordRequest{
		Kind: "chore", Name: "tidy-up",
		Status: backlog.StatusCompleted, DecidedBy: "user",
	})
	if err != nil {
		t.Fatalf("CreateBacklogRecord: %v", err)
	}
	if id == "" || len(idx.indexed) != 1 {
		t.Fatalf("expected exactly 1 indexed record, got id=%q count=%d", id, len(idx.indexed))
	}
	r := idx.indexed[0]
	if r.Trigger != "" {
		t.Errorf("trigger = %q, want empty (no title)", r.Trigger)
	}
	if !strings.Contains(r.Approach, "Auto-captured") || !strings.Contains(r.Approach, "records supersede") {
		t.Errorf("approach = %q, want a templated enrich prompt", r.Approach)
	}
	if r.Scenario != "swarm-manager" {
		t.Errorf("scenario = %q, want swarm-manager fallback (no globs)", r.Scenario)
	}
}
