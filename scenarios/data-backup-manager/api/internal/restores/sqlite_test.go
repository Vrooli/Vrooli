package restores_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"

	localdb "data-backup-manager/internal/database"
	"data-backup-manager/internal/restores"
	db "github.com/vrooli/api-core/databasetest"

	"github.com/vrooli/api-core/scheduletest"
)

func newRestoresDB(t *testing.T) *sql.DB {
	t.Helper()
	d := db.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), d,
		apidb.SchemaProviderFunc(localdb.SystemSchema),
		apidb.SchemaProviderFunc(restores.Schema),
	); err != nil {
		t.Fatalf("ensure schemas: %v", err)
	}
	return d
}

// TestSQLiteRepository_RoundTrip is the real-sqlite round-trip: create / get /
// list with a restore record and a verify record.
func TestSQLiteRepository_RoundTrip(t *testing.T) {
	ctx := context.Background()
	clk := scheduletest.New(time.Time{})
	repo := restores.NewSQLiteRepository(newRestoresDB(t), clk)

	verifiedAt := clk.Now().UTC().Add(time.Minute)
	r := restores.Restore{
		TargetID:       "t1",
		DestinationID:  "dst-1",
		SnapshotID:     "snap-1",
		Mode:           restores.ModeVerify,
		Status:         restores.RestoreVerified,
		Checksum:       "deadbeef",
		LastVerifiedAt: verifiedAt,
		RequestedAt:    clk.Now().UTC(),
		FinishedAt:     verifiedAt,
	}

	created, err := repo.CreateRestore(ctx, r)
	if err != nil {
		t.Fatalf("CreateRestore: %v", err)
	}
	if created.ID == "" {
		t.Fatal("ID must be set after create")
	}

	got, err := repo.GetRestore(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetRestore: %v", err)
	}
	if got.Status != restores.RestoreVerified {
		t.Errorf("status = %s, want verified", got.Status)
	}
	if got.Checksum != "deadbeef" {
		t.Errorf("checksum = %q", got.Checksum)
	}
	if got.LastVerifiedAt.IsZero() {
		t.Error("last_verified_at must not be zero after a verified restore")
	}

	// A second restore record (mode=restore) for the same target.
	r2 := restores.Restore{
		TargetID:      "t1",
		DestinationID: "dst-1",
		SnapshotID:    "snap-2",
		Mode:          restores.ModeRestore,
		Status:        restores.RestoreRestored,
		Location:      "/tmp/dest",
		RequestedAt:   clk.Now().UTC(),
		FinishedAt:    clk.Now().UTC(),
	}
	if _, err := repo.CreateRestore(ctx, r2); err != nil {
		t.Fatalf("CreateRestore r2: %v", err)
	}

	// ListRestores with target filter.
	list, err := repo.ListRestores(ctx, "t1", 100)
	if err != nil {
		t.Fatalf("ListRestores: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected 2 restores, got %d", len(list))
	}

	// GetRestore for unknown id returns ErrRestoreNotFound.
	_, err = repo.GetRestore(ctx, "no-such-id")
	var notFound restores.ErrRestoreNotFound
	if !isErrRestoreNotFound(err, &notFound) {
		t.Fatalf("expected ErrRestoreNotFound, got %v", err)
	}
}

// TestLastVerifiedByTarget asserts the proven-restorable rollup: only
// status=verified, mode=verify records count; the newest verify wins per
// target; failed verifies and plain restores are ignored; and a target that
// has only been backed up (never verified) is absent from the result.
func TestLastVerifiedByTarget(t *testing.T) {
	ctx := context.Background()
	clk := scheduletest.New(time.Unix(1700000000, 0).UTC())
	repo := restores.NewSQLiteRepository(newRestoresDB(t), clk)

	base := clk.Now().UTC()
	mustCreate := func(r restores.Restore) {
		t.Helper()
		if _, err := repo.CreateRestore(ctx, r); err != nil {
			t.Fatalf("CreateRestore: %v", err)
		}
	}

	// t1: an older verify then a newer verify — newest must win.
	mustCreate(restores.Restore{
		TargetID: "t1", DestinationID: "d1", SnapshotID: "snap-old", Mode: restores.ModeVerify,
		Status: restores.RestoreVerified, LastVerifiedAt: base, RequestedAt: base, FinishedAt: base,
	})
	mustCreate(restores.Restore{
		TargetID: "t1", DestinationID: "d1", SnapshotID: "snap-new", Mode: restores.ModeVerify,
		Status: restores.RestoreVerified, LastVerifiedAt: base.Add(time.Hour), RequestedAt: base.Add(time.Hour), FinishedAt: base.Add(time.Hour),
	})
	// t1: a failed verify after the newest success — must NOT override it.
	mustCreate(restores.Restore{
		TargetID: "t1", DestinationID: "d1", SnapshotID: "snap-bad", Mode: restores.ModeVerify,
		Status: restores.RestoreFailed, RequestedAt: base.Add(2 * time.Hour), FinishedAt: base.Add(2 * time.Hour), Error: "boom",
	})
	// t2: only a plain restore (mode=restore) — never verified, must be absent.
	mustCreate(restores.Restore{
		TargetID: "t2", DestinationID: "d1", SnapshotID: "snap-r", Mode: restores.ModeRestore,
		Status: restores.RestoreRestored, Location: "/tmp/x", RequestedAt: base, FinishedAt: base,
	})

	all, err := repo.LastVerifiedByTarget(ctx, nil)
	if err != nil {
		t.Fatalf("LastVerifiedByTarget: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 verified target (t1 only), got %d: %+v", len(all), all)
	}
	got := all[0]
	if got.TargetID != "t1" {
		t.Fatalf("target = %q, want t1", got.TargetID)
	}
	if got.SnapshotID != "snap-new" {
		t.Errorf("snapshot = %q, want snap-new (newest verify wins)", got.SnapshotID)
	}
	if !got.LastVerifiedAt.Equal(base.Add(time.Hour)) {
		t.Errorf("last_verified_at = %v, want %v", got.LastVerifiedAt, base.Add(time.Hour))
	}

	// Filtered to a target with no verify history returns empty.
	none, err := repo.LastVerifiedByTarget(ctx, []string{"t2"})
	if err != nil {
		t.Fatalf("LastVerifiedByTarget(t2): %v", err)
	}
	if len(none) != 0 {
		t.Fatalf("t2 was never verified; expected empty, got %+v", none)
	}
}

func isErrRestoreNotFound(err error, out *restores.ErrRestoreNotFound) bool {
	if err == nil {
		return false
	}
	if nf, ok := err.(restores.ErrRestoreNotFound); ok {
		*out = nf
		return true
	}
	return false
}
