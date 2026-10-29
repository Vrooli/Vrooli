package database

import (
	"context"
	"testing"

	"agent-manager/internal/domain"

	"github.com/google/uuid"
)

// Attached runs the reconciler failed only because their harness exited are
// corrected to complete and queued for re-projection; real failures stay.
func TestMigrateAttachedExitStatusCompletesNormalExitsOnly(t *testing.T) {
	db, cleanup := setupTestDB(t)
	t.Cleanup(cleanup)
	ctx := context.Background()
	repo := &runRepository{db: db}
	exited := &domain.Run{ID: uuid.New(), Tag: "attached-exited", ExecutionMode: domain.ExecutionModeAttached, Status: domain.RunStatusFailed, ErrorMsg: attachedExitReason}
	broken := &domain.Run{ID: uuid.New(), Tag: "attached-broken", ExecutionMode: domain.ExecutionModeAttached, Status: domain.RunStatusFailed, ErrorMsg: "identity verification failed"}
	for _, run := range []*domain.Run{exited, broken} {
		if err := repo.Create(ctx, run); err != nil {
			t.Fatal(err)
		}
	}
	for range 2 { // idempotent
		if err := db.migrateAttachedExitStatus(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if got, _ := repo.Get(ctx, exited.ID); got.Status != domain.RunStatusComplete || got.ErrorMsg != "" {
		t.Fatalf("exited attached run = %s %q", got.Status, got.ErrorMsg)
	}
	if got, _ := repo.Get(ctx, broken.ID); got.Status != domain.RunStatusFailed {
		t.Fatalf("real failure changed to %s", got.Status)
	}
	var queued int
	if err := db.GetContext(ctx, &queued, `SELECT COUNT(*) FROM invocation_read_model_pending WHERE run_id = ?`, exited.ID.String()); err != nil || queued != 1 {
		t.Fatalf("re-projection queued = %d, %v", queued, err)
	}
}
