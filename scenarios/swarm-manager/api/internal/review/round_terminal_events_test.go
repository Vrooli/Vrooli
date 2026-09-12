package review

import (
	"context"
	"path/filepath"
	"testing"
)

type recordingReviewEvents struct {
	failed    []string
	completed []string
}

func (r *recordingReviewEvents) EmitReviewStarted(string, int)                   {}
func (r *recordingReviewEvents) EmitReviewEvidenceVerified(string, string)       {}
func (r *recordingReviewEvents) EmitReviewRequestCreated(string, string, string) {}

func (r *recordingReviewEvents) EmitReviewRoundCompleted(executionID string, _, _ int, _ string, _ float64) {
	r.completed = append(r.completed, executionID)
}

func (r *recordingReviewEvents) EmitReviewFailed(_ context.Context, executionID, reason string, _ float64) {
	r.failed = append(r.failed, executionID+"|"+reason)
}

// A review that never ran is pushback evidence. Recording the round without
// announcing it left the durability lane able to see reviews starting but
// never failing, which is why only one review.failed event exists in history.
func TestRecordUnavailableReviewEmitsReviewFailed(t *testing.T) {
	dir := t.TempDir()
	events := &recordingReviewEvents{}
	svc := &Service{
		dataRoot:    dir,
		itemDirFn:   func(_, _ string) string { return filepath.Join(dir, "item") },
		eventLogger: events,
	}

	if err := svc.RecordUnavailableReview("fix", "bug-1", "exec-42", "review agent did not run"); err != nil {
		t.Fatal(err)
	}

	if len(events.failed) != 1 {
		t.Fatalf("failed events = %v, want exactly one", events.failed)
	}
	if events.failed[0] != "exec-42|review agent did not run" {
		t.Fatalf("failed event = %q", events.failed[0])
	}
}

// Without an execution to attribute it to, the event would be unjoinable
// evidence. Skipping is better than emitting a dangling reference.
func TestRecordUnavailableReviewSkipsEmitWithoutExecution(t *testing.T) {
	dir := t.TempDir()
	events := &recordingReviewEvents{}
	svc := &Service{
		dataRoot:    dir,
		itemDirFn:   func(_, _ string) string { return filepath.Join(dir, "item") },
		eventLogger: events,
	}

	if err := svc.RecordUnavailableReview("fix", "bug-2", "", "no execution"); err != nil {
		t.Fatal(err)
	}
	if len(events.failed) != 0 {
		t.Fatalf("failed events = %v, want none", events.failed)
	}
}
