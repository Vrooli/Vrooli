package transitionrun

import (
	"errors"
	"testing"
)

func validCorrelation() Correlation {
	return Correlation{TransitionKey: "capture.extract", SubjectKind: "capture", SubjectRef: "cap-1", ExecutionID: "exec-1", WorkflowKey: "capture-workflow", DefinitionDigest: "sha256:def", EntityVersion: "v1", FrontierDigest: "frontier-1", DeclaredOutcomes: []string{"complete", "blocked"}}
}

func validCompletion() Completion {
	return Completion{ExecutionID: "exec-1", DefinitionDigest: "sha256:def", EntityVersion: "v1", FrontierDigest: "frontier-1", Status: CompletionSucceeded, Outcome: "complete"}
}

func TestReplayAfterDeliveredResult(t *testing.T) {
	c := validCorrelation()
	c.ApplyState = ApplyStateComplete
	err := CanApply(c, validCompletion())
	var typed *AlreadyCompleteError
	if !errors.As(err, &typed) {
		t.Fatalf("CanApply error = %v, want AlreadyCompleteError", err)
	}
}

func TestStaleEntityVersion(t *testing.T) {
	completion := validCompletion()
	completion.EntityVersion = "v2"
	err := CanApply(validCorrelation(), completion)
	var typed *EntityVersionChangedError
	if !errors.As(err, &typed) {
		t.Fatalf("CanApply error = %v, want EntityVersionChangedError", err)
	}
}

func TestStaleFrontier(t *testing.T) {
	completion := validCompletion()
	completion.FrontierDigest = "frontier-2"
	err := CanApply(validCorrelation(), completion)
	var typed *FrontierChangedError
	if !errors.As(err, &typed) {
		t.Fatalf("CanApply error = %v, want FrontierChangedError", err)
	}
}

func TestCrashBetweenClaimedAndComplete(t *testing.T) {
	store := NewFileStore(t.TempDir())
	c := validCorrelation()
	c.ApplyState = ApplyStateClaimed
	if err := store.Put(c); err != nil {
		t.Fatal(err)
	}
	pending, err := store.ListUnapplied()
	if err != nil || len(pending) != 1 || pending[0].ExecutionID != c.ExecutionID {
		t.Fatalf("ListUnapplied = %#v, %v", pending, err)
	}
}

func TestApprovalOrdering(t *testing.T) {
	store := NewFileStore(t.TempDir())
	c := validCorrelation()
	c.ApprovalActor = "operator"
	c.ApprovalTime = "2026-07-27T00:00:00Z"
	c.ApplyState = ApplyStateClaimed
	if err := store.Put(c); err != nil {
		t.Fatal(err)
	}
	got, err := store.Get(c.ExecutionID)
	if err != nil || got.ApprovalActor != c.ApprovalActor || got.ApprovalTime != c.ApprovalTime {
		t.Fatalf("Get = %#v, %v", got, err)
	}
}

func TestCancellationDuringRun(t *testing.T) {
	completion := validCompletion()
	completion.Status = "cancelled"
	err := CanApply(validCorrelation(), completion)
	var typed *StatusNotSucceededError
	if !errors.As(err, &typed) {
		t.Fatalf("CanApply error = %v, want StatusNotSucceededError", err)
	}
}

func TestCanApplyRejectsDigestAndOutcome(t *testing.T) {
	completion := validCompletion()
	completion.DefinitionDigest = "sha256:other"
	var digest *DigestMismatchError
	if !errors.As(CanApply(validCorrelation(), completion), &digest) {
		t.Fatal("expected digest mismatch")
	}
	completion = validCompletion()
	completion.Outcome = "failed"
	var outcome *OutcomeNotDeclaredError
	if !errors.As(CanApply(validCorrelation(), completion), &outcome) {
		t.Fatal("expected undeclared outcome")
	}
}
