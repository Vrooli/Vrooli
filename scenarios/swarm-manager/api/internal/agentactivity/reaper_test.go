package agentactivity

import (
	"testing"
	"time"
)

func TestReapExpiredNeedsReviewFreesLaneAndRecordsReason(t *testing.T) {
	store := NewStore(t.TempDir() + "/activities.json")
	old := time.Now().UTC().Add(-2 * time.Hour).Format(time.RFC3339)
	current := time.Now().UTC().Format(time.RFC3339)
	if err := store.Save([]Record{
		{ActivityID: "parked", OwnerType: OwnerBacklog, OwnerKind: "execute", OwnerName: "old", Purpose: PurposeProcess, PhaseKind: "execute", Status: StatusNeedsReview, RequestedAt: old, UpdatedAt: old},
		{ActivityID: "fresh", OwnerType: OwnerBacklog, OwnerKind: "execute", OwnerName: "new", Purpose: PurposeProcess, PhaseKind: "execute", Status: StatusNeedsReview, RequestedAt: current, UpdatedAt: current},
	}); err != nil {
		t.Fatalf("seed activities: %v", err)
	}

	svc := NewService(ServiceConfig{StorePath: t.TempDir() + "/unused.json", NeedsReviewTTL: time.Hour})
	svc.store = store
	if err := svc.ReapExpiredNeedsReview(); err != nil {
		t.Fatalf("reap: %v", err)
	}
	records, err := store.Load()
	if err != nil {
		t.Fatalf("load activities: %v", err)
	}
	if records[0].Status != StatusFailed {
		t.Fatalf("expired status = %q, want failed", records[0].Status)
	}
	if records[0].FailureReason == "" {
		t.Fatal("expired record has no failure reason")
	}
	if records[1].Status != StatusNeedsReview {
		t.Fatalf("fresh status = %q, want needs_review", records[1].Status)
	}
	counts, err := svc.LaneActiveCounts()
	if err != nil {
		t.Fatalf("lane counts: %v", err)
	}
	if counts[LaneExecute] != 1 {
		t.Fatalf("execute active = %d, want fresh holder only", counts[LaneExecute])
	}
}
