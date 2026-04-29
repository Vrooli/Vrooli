package repository

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// TestHealState_RoundTrip exercises the full upsert/get/clear cycle
// against a real SQLite database. Pins the column shape and the
// idempotency contract that the heal tracker relies on.
func TestHealState_RoundTrip(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	// To satisfy the foreign-key constraint we need a real sandbox row.
	sb := newTestSandbox()
	if err := repo.Create(ctx, sb); err != nil {
		t.Fatalf("create sandbox: %v", err)
	}

	// Initially empty.
	got, err := repo.GetHealState(ctx, sb.ID)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got != nil {
		t.Fatalf("expected nil for empty store, got %+v", got)
	}

	// Upsert.
	now := time.Now().UTC().Truncate(time.Second)
	row := HealStateRow{
		SandboxID:           sb.ID,
		ConsecutiveFailures: 3,
		LastAttempt:         now,
		LastError:           "mount stale",
	}
	if err := repo.UpsertHealState(ctx, row); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	got, err = repo.GetHealState(ctx, sb.ID)
	if err != nil {
		t.Fatalf("Get after upsert: %v", err)
	}
	if got == nil {
		t.Fatal("expected row, got nil")
	}
	if got.ConsecutiveFailures != 3 {
		t.Errorf("ConsecutiveFailures = %d, want 3", got.ConsecutiveFailures)
	}
	if got.LastError != "mount stale" {
		t.Errorf("LastError = %q, want %q", got.LastError, "mount stale")
	}
	if !got.LastAttempt.Equal(now) {
		t.Errorf("LastAttempt = %v, want %v", got.LastAttempt, now)
	}

	// Upsert is idempotent + replaces values.
	row.ConsecutiveFailures = 5
	row.LastError = "still stale"
	if err := repo.UpsertHealState(ctx, row); err != nil {
		t.Fatalf("Upsert (replace): %v", err)
	}
	got, _ = repo.GetHealState(ctx, sb.ID)
	if got.ConsecutiveFailures != 5 {
		t.Errorf("after replace ConsecutiveFailures = %d, want 5", got.ConsecutiveFailures)
	}

	// List sees the row.
	rows, err := repo.ListHealState(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("List len = %d, want 1", len(rows))
	}

	// Clear.
	if err := repo.ClearHealState(ctx, sb.ID); err != nil {
		t.Fatalf("Clear: %v", err)
	}
	got, err = repo.GetHealState(ctx, sb.ID)
	if err != nil {
		t.Fatalf("Get after clear: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil after clear, got %+v", got)
	}

	// Clear is idempotent.
	if err := repo.ClearHealState(ctx, sb.ID); err != nil {
		t.Errorf("clear should be idempotent, got %v", err)
	}
}

// TestHealState_ForeignKeyCascade — when a sandbox is hard-deleted
// (which our schema does not normally do — Delete sets status='deleted')
// the heal_state row goes with it. The foreign key cascade is the
// safety net so a future row-delete path can't leave dangling state.
func TestHealState_SurvivesSoftDelete(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	sb := newTestSandbox()
	if err := repo.Create(ctx, sb); err != nil {
		t.Fatal(err)
	}
	row := HealStateRow{
		SandboxID:           sb.ID,
		ConsecutiveFailures: 1,
		LastAttempt:         time.Now().UTC(),
		LastError:           "",
	}
	if err := repo.UpsertHealState(ctx, row); err != nil {
		t.Fatal(err)
	}
	// Soft-delete (sets status=deleted). The heal_state row should
	// survive — clearing on the sandbox.Service layer is what handles
	// this, not the schema.
	if err := repo.Delete(ctx, sb.ID); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetHealState(ctx, sb.ID)
	if err != nil {
		t.Fatalf("Get after soft-delete: %v", err)
	}
	if got == nil {
		t.Error("expected heal_state row to survive soft-delete; service layer is responsible for clearing it")
	}
}

// TestHealState_Empty surfaces the (nil, nil) contract for a missing row.
func TestHealState_Empty(t *testing.T) {
	repo := newTestRepo(t)
	got, err := repo.GetHealState(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("Get on empty store: %v", err)
	}
	if got != nil {
		t.Errorf("expected nil, got %+v", got)
	}
}
