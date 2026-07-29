package database

import (
	"context"
	"testing"

	"agent-manager/internal/runreport"
	"github.com/google/uuid"
)

func TestInvocationFactRepositoryRebuildIsIdempotent(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &invocationFactRepository{db: db}
	runID := uuid.New()
	facts := []runreport.InvocationFact{{Version: "invocation-fact.v1", CallEventID: "call-a", ToolName: "shell", Ownership: "unknown", Outcome: "failure", Fingerprint: "fp", Availability: "available"}}
	if err := repo.ReplaceInvocationFacts(context.Background(), runID, facts); err != nil {
		t.Fatal(err)
	}
	if err := repo.ReplaceInvocationFacts(context.Background(), runID, facts); err != nil {
		t.Fatal(err)
	}
	got, err := repo.InvocationFacts(context.Background(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].CallEventID != "call-a" || got[0].Version != facts[0].Version {
		t.Fatalf("facts=%+v", got)
	}
}

func TestReceiptEvidenceRepositoryRebuildKeepsOnlyVerifiedIDs(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()
	repo := &invocationFactRepository{db: db}
	runID := uuid.New()

	if err := repo.ReplaceReceiptEvidence(context.Background(), runID, "available", []string{"event-b", "event-a"}); err != nil {
		t.Fatal(err)
	}
	if err := repo.ReplaceReceiptEvidence(context.Background(), runID, "unobserved", nil); err != nil {
		t.Fatal(err)
	}
	got, err := repo.ReceiptEvidence(context.Background(), runID)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("receipt evidence=%v, want no stale identifiers", got)
	}
}
