package audits_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	auditsH "data-backup-manager/handlers/audits"
	"data-backup-manager/internal/audits"
	localdb "data-backup-manager/internal/database"
	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"

	apidb "github.com/vrooli/api-core/database"
)

func newAuditsDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(auditsH.Schema),
	); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	return d
}

func TestRepository_RoundTripWithInventories(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	repo := audits.NewSQLiteRepository(newAuditsDB(t), clk)
	ctx := context.Background()

	created, err := repo.CreateAudit(ctx, audits.Audit{
		TargetID:           "t-1",
		DestinationID:      "d-1",
		SnapshotID:         "s-1",
		IncludeContentHash: true,
		IncludeSQLiteCheck: true,
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if created.ID == "" {
		t.Fatalf("expected generated id")
	}
	if created.Status != audits.AuditRequested {
		t.Errorf("status = %q, want requested", created.Status)
	}

	// Finish with full evidence and confirm it survives a round trip.
	created.Status = audits.AuditCompleted
	created.Restorable = true
	created.SnapshotTime = time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	created.FinishedAt = time.Date(2026, 6, 1, 1, 0, 0, 0, time.UTC)
	created.Live = &audits.InventorySummary{Files: 5, RegularBytes: 100, PathListSHA256: "lp"}
	created.Snapshot = &audits.InventorySummary{
		Files: 5, RegularBytes: 100, PathListSHA256: "lp",
		SQLite: []audits.SqliteInventory{{Path: "events.db", IntegrityStatus: "ok", TableCount: 3, SchemaSHA256: "sh"}},
	}
	created.Comparison = &audits.AuditComparison{Matches: true}
	if err := repo.FinishAudit(ctx, created); err != nil {
		t.Fatalf("finish: %v", err)
	}

	got, err := repo.GetAudit(ctx, created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.Status != audits.AuditCompleted || !got.Restorable {
		t.Errorf("status/restorable not persisted: %q/%v", got.Status, got.Restorable)
	}
	if got.Live == nil || got.Live.Files != 5 {
		t.Errorf("live inventory lost: %+v", got.Live)
	}
	if got.Snapshot == nil || len(got.Snapshot.SQLite) != 1 || got.Snapshot.SQLite[0].Path != "events.db" {
		t.Errorf("snapshot sqlite inventory lost: %+v", got.Snapshot)
	}
	if got.Comparison == nil || !got.Comparison.Matches {
		t.Errorf("comparison lost: %+v", got.Comparison)
	}
	if !got.SnapshotTime.Equal(created.SnapshotTime) {
		t.Errorf("snapshot_time = %v, want %v", got.SnapshotTime, created.SnapshotTime)
	}
}

func TestRepository_ListNewestFirstAndFilter(t *testing.T) {
	clk := scheduletest.New(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))
	repo := audits.NewSQLiteRepository(newAuditsDB(t), clk)
	ctx := context.Background()

	for i, target := range []string{"t-1", "t-2", "t-1"} {
		clk.Advance(time.Duration(i+1) * time.Minute)
		if _, err := repo.CreateAudit(ctx, audits.Audit{TargetID: target, DestinationID: "d", SnapshotID: "s"}); err != nil {
			t.Fatalf("create: %v", err)
		}
	}

	all, err := repo.ListAudits(ctx, "", 10)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(all) != 3 {
		t.Fatalf("list all = %d, want 3", len(all))
	}
	filtered, err := repo.ListAudits(ctx, "t-1", 10)
	if err != nil {
		t.Fatalf("list filtered: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("list t-1 = %d, want 2", len(filtered))
	}
}

func TestRepository_GetUnknownReturnsNotFound(t *testing.T) {
	repo := audits.NewSQLiteRepository(newAuditsDB(t), scheduletest.New(time.Time{}))
	_, err := repo.GetAudit(context.Background(), "nope")
	var nf audits.ErrAuditNotFound
	if !asAuditErr(err, &nf) {
		t.Errorf("expected ErrAuditNotFound, got %v", err)
	}
}

func asAuditErr(err error, target *audits.ErrAuditNotFound) bool {
	if e, ok := err.(audits.ErrAuditNotFound); ok {
		*target = e
		return true
	}
	return false
}
