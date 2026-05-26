package restores_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	apidb "github.com/vrooli/api-core/database"

	localdb "data-backup-manager/internal/database"
	"data-backup-manager/internal/restores"
	"data-backup-manager/internal/testutil/db"
	"data-backup-manager/internal/testutil/mocks"
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
	clk := mocks.NewFakeClock(time.Time{})
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
