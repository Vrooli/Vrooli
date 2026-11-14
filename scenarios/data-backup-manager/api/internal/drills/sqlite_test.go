package drills_test

import (
	"context"
	"testing"
	"time"

	drills "data-backup-manager/internal/drills"

	apidb "github.com/vrooli/api-core/database"
	db "github.com/vrooli/api-core/databasetest"
)

func newDrillRepo(t *testing.T) drills.Repository {
	t.Helper()
	dbHandle := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), dbHandle, apidb.SchemaProviderFunc(drills.Schema)); err != nil {
		t.Fatal(err)
	}
	return drills.NewSQLiteRepository(dbHandle)
}

func TestSQLiteRepository_IdempotencyAndLatest(t *testing.T) {
	repo := newDrillRepo(t)
	ctx := context.Background()
	now := time.Date(2026, 8, 5, 12, 0, 0, 0, time.UTC)
	first, err := repo.Create(ctx, drills.Drill{PlanID: "plan-1", TargetID: "target-1", DestinationID: "dest-1", Status: drills.StatusFailed, IdempotencyKey: "retry-1", RequestedAt: now, FinishedAt: now})
	if err != nil {
		t.Fatal(err)
	}
	got, ok, err := repo.FindByIdempotency(ctx, "retry-1")
	if err != nil || !ok {
		t.Fatalf("FindByIdempotency: got=%+v ok=%v err=%v", got, ok, err)
	}
	if got.ID != first.ID {
		t.Fatalf("idempotency returned %q, want %q", got.ID, first.ID)
	}
	latest, ok, err := repo.LatestForUnit(ctx, "plan-1", "target-1", "dest-1")
	if err != nil || !ok {
		t.Fatalf("LatestForUnit: got=%+v ok=%v err=%v", latest, ok, err)
	}
	if latest.IdempotencyKey != "retry-1" {
		t.Fatalf("latest key = %q", latest.IdempotencyKey)
	}
}

func TestSQLiteRepository_EnforcesDrillTransitionsWithoutMutation(t *testing.T) {
	repo := newDrillRepo(t)
	ctx := context.Background()
	drill, err := repo.Create(ctx, drills.Drill{PlanID: "plan-1", TargetID: "target-1", DestinationID: "dest-1", Status: drills.StatusRequested})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Finish(ctx, drill.ID, drills.EventVerify, "", "", time.Now()); err == nil {
		t.Fatal("requested drill accepted verification")
	}
	got, err := repo.Get(ctx, drill.ID)
	if err != nil || got.Status != drills.StatusRequested {
		t.Fatalf("rejected verification mutated state: drill=%+v err=%v", got, err)
	}
	if err := repo.Start(ctx, drill.ID, time.Now()); err != nil {
		t.Fatalf("start: %v", err)
	}
	if err := repo.Finish(ctx, drill.ID, drills.EventVerify, "", "", time.Now()); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if err := repo.Finish(ctx, drill.ID, drills.EventFail, "late failure", "", time.Now()); err == nil {
		t.Fatal("terminal drill accepted a second finish event")
	}
	got, err = repo.Get(ctx, drill.ID)
	if err != nil || got.Status != drills.StatusVerified {
		t.Fatalf("terminal rejection mutated state: drill=%+v err=%v", got, err)
	}
}
