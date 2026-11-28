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
	if err := drills.EnsureColumns(context.Background(), dbHandle); err != nil {
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

func TestSQLiteRepository_RetirePreservesCompletedEvidence(t *testing.T) {
	repo := newDrillRepo(t)
	ctx := context.Background()
	finishedAt := time.Date(2026, 8, 19, 17, 32, 23, 0, time.UTC)
	created, err := repo.Create(ctx, drills.Drill{ID: "orphaned", PlanID: "removed-plan", TargetID: "t", DestinationID: "d", SnapshotID: "snapshot-proof", RestoreID: "restore-proof", Status: drills.StatusRequested, RequestedAt: finishedAt.Add(-time.Minute)})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Start(ctx, created.ID, finishedAt.Add(-time.Second)); err != nil {
		t.Fatal(err)
	}
	if err := repo.Finish(ctx, created.ID, drills.EventVerify, "", "", finishedAt); err != nil {
		t.Fatal(err)
	}
	created, err = repo.Get(ctx, created.ID)
	if err != nil {
		t.Fatal(err)
	}
	retiredAt := finishedAt.Add(24 * time.Hour)
	retired, err := repo.Retire(ctx, created.ID, "orphaned plan reference", retiredAt)
	if err != nil {
		t.Fatal(err)
	}
	if retired.Status != drills.StatusVerified || retired.SnapshotID != "snapshot-proof" || retired.RestoreID != "restore-proof" || retired.FinishedAt != finishedAt || !retired.RetiredAt.Equal(retiredAt) || retired.RetirementReason != "orphaned plan reference" {
		t.Fatalf("retirement did not preserve evidence: %+v", retired)
	}
	again, err := repo.Retire(ctx, created.ID, "second reason", retiredAt.Add(time.Hour))
	if err != nil || !again.RetiredAt.Equal(retiredAt) || again.RetirementReason != "orphaned plan reference" {
		t.Fatalf("repeat retirement changed the original receipt: %+v err=%v", again, err)
	}
}

func TestSQLiteRepository_RejectsRetiringActiveDrill(t *testing.T) {
	repo := newDrillRepo(t)
	ctx := context.Background()
	created, err := repo.Create(ctx, drills.Drill{ID: "active", PlanID: "p", TargetID: "t", DestinationID: "d", Status: drills.StatusRequested})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := repo.Retire(ctx, created.ID, "cleanup", time.Now().UTC()); err == nil {
		t.Fatal("active drill was retired")
	}
	got, err := repo.Get(ctx, created.ID)
	if err != nil || !got.RetiredAt.IsZero() || got.Status != drills.StatusRequested {
		t.Fatalf("active drill changed: %+v err=%v", got, err)
	}
}

func TestSQLiteRepository_ListPageContinuesAcrossEqualTimestamps(t *testing.T) {
	repo := newDrillRepo(t)
	ctx := context.Background()
	newer := time.Date(2026, 11, 13, 18, 0, 0, 123456000, time.UTC)
	older := newer.Add(-time.Hour)
	for _, drill := range []drills.Drill{
		{ID: "a", PlanID: "p", TargetID: "t", DestinationID: "d", Status: drills.StatusRequested, RequestedAt: newer},
		{ID: "b", PlanID: "p", TargetID: "t", DestinationID: "d", Status: drills.StatusRequested, RequestedAt: newer},
		{ID: "c", PlanID: "p", TargetID: "t", DestinationID: "d", Status: drills.StatusRequested, RequestedAt: older},
		{ID: "d", PlanID: "p", TargetID: "t", DestinationID: "d", Status: drills.StatusRequested, RequestedAt: older},
	} {
		if _, err := repo.Create(ctx, drill); err != nil {
			t.Fatal(err)
		}
	}
	first, err := repo.ListPage(ctx, "p", "t", 2, "")
	if err != nil || len(first.Drills) != 2 || first.NextPageToken == "" {
		t.Fatalf("first page = %+v, err=%v", first, err)
	}
	if first.Drills[0].ID != "b" || first.Drills[1].ID != "a" {
		t.Fatalf("first page ids = %q,%q; want b,a", first.Drills[0].ID, first.Drills[1].ID)
	}
	second, err := repo.ListPage(ctx, "p", "t", 2, first.NextPageToken)
	if err != nil || len(second.Drills) != 2 || second.NextPageToken != "" {
		t.Fatalf("second page = %+v, err=%v", second, err)
	}
	if second.Drills[0].ID != "d" || second.Drills[1].ID != "c" {
		t.Fatalf("second page ids = %q,%q; want d,c", second.Drills[0].ID, second.Drills[1].ID)
	}
	if _, err := repo.ListPage(ctx, "p", "t", 2, "not-a-cursor"); err == nil {
		t.Fatal("invalid cursor was accepted")
	}
}

func TestEnsureColumnsDoesNotReplayHistoricalFailures(t *testing.T) {
	ctx := context.Background()
	dbHandle := db.NewSQLite(t)
	_, err := dbHandle.ExecContext(ctx, `CREATE TABLE recovery_drills (
		id TEXT PRIMARY KEY, plan_id TEXT NOT NULL, target_id TEXT NOT NULL,
		destination_id TEXT NOT NULL, snapshot_id TEXT NOT NULL DEFAULT '', restore_id TEXT NOT NULL DEFAULT '',
		status TEXT NOT NULL, scheduled INTEGER NOT NULL DEFAULT 0, idempotency_key TEXT NOT NULL DEFAULT '',
		error TEXT NOT NULL DEFAULT '', next_action TEXT NOT NULL DEFAULT '', requested_at TEXT NOT NULL,
		started_at TEXT NOT NULL DEFAULT '', finished_at TEXT NOT NULL DEFAULT '')`)
	if err != nil {
		t.Fatal(err)
	}
	_, err = dbHandle.ExecContext(ctx, `INSERT INTO recovery_drills(id,plan_id,target_id,destination_id,status,requested_at,error) VALUES('old','p','t','d','failed','2026-08-01T00:00:00Z','old failure')`)
	if err != nil {
		t.Fatal(err)
	}
	if err := drills.EnsureColumns(ctx, dbHandle); err != nil {
		t.Fatal(err)
	}
	repo := drills.NewSQLiteRepository(dbHandle)
	old, err := repo.Get(ctx, "old")
	if err != nil || old.NotificationState != drills.NotificationUntracked {
		t.Fatalf("legacy failure state = %+v, err=%v; want explicitly untracked", old, err)
	}
	pending, err := repo.ListPendingNotifications(ctx, 100)
	if err != nil || len(pending) != 0 {
		t.Fatalf("migration replayed old failures: pending=%+v err=%v", pending, err)
	}
	created, err := repo.Create(ctx, drills.Drill{PlanID: "p", TargetID: "t", DestinationID: "d", Status: drills.StatusFailed, RequestedAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	pending, err = repo.ListPendingNotifications(ctx, 100)
	if err != nil || len(pending) != 1 || pending[0].ID != created.ID || pending[0].NotificationState != drills.NotificationPending {
		t.Fatalf("new failure missing from outbox: pending=%+v err=%v", pending, err)
	}
}
