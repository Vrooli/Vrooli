package destinations_test

import (
	"context"
	"testing"

	"data-backup-manager/internal/destinations"
	"data-backup-manager/internal/testutil/db"

	"github.com/vrooli/api-core/schedule"
)

// TestEnsureColumns_AdditiveAndIdempotent proves the migration adds
// repository_location to a legacy destinations table (created without it)
// without losing data, is a no-op when re-applied, and that a fresh row can
// then carry the new column. This is the path that was failing at runtime: a
// live DB created before repository_location existed.
func TestEnsureColumns_AdditiveAndIdempotent(t *testing.T) {
	d := db.NewSQLite(t)
	ctx := context.Background()

	// Legacy table: the destinations schema before repository_location.
	legacy := `CREATE TABLE destinations (
	  id TEXT PRIMARY KEY, name TEXT NOT NULL UNIQUE, backend_kind TEXT NOT NULL,
	  location TEXT NOT NULL, cap_bytes INTEGER NOT NULL DEFAULT 0,
	  cap_policy TEXT NOT NULL DEFAULT 'alert_block', encryption_algorithm TEXT NOT NULL DEFAULT '',
	  secret_ref TEXT NOT NULL DEFAULT '', created_at TEXT NOT NULL, updated_at TEXT NOT NULL)`
	if _, err := d.ExecContext(ctx, legacy); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if _, err := d.ExecContext(ctx, `INSERT INTO destinations
	  (id, name, backend_kind, location, cap_bytes, cap_policy, encryption_algorithm, secret_ref, created_at, updated_at)
	  VALUES ('legacy', 'legacy-dest', 'filesystem', '/mnt/legacy', 0, 'alert_block', 'AES256', '', '2026-05-29T00:00:00Z', '2026-05-29T00:00:00Z')`); err != nil {
		t.Fatalf("seed legacy row: %v", err)
	}

	// First migration adds the column; existing row survives.
	if err := destinations.EnsureColumns(ctx, d); err != nil {
		t.Fatalf("EnsureColumns: %v", err)
	}
	// Re-applying is a no-op (no "duplicate column" error).
	if err := destinations.EnsureColumns(ctx, d); err != nil {
		t.Fatalf("EnsureColumns (idempotent): %v", err)
	}

	repo := destinations.NewSQLiteRepository(d, schedule.System())
	got, err := repo.GetByID(ctx, "legacy")
	if err != nil {
		t.Fatalf("GetByID legacy: %v", err)
	}
	if got.RepositoryLocation != "" {
		t.Fatalf("legacy RepositoryLocation = %q, want empty", got.RepositoryLocation)
	}

	// A new row can write the new column.
	created, err := repo.Create(ctx, destinations.Destination{
		Name:               "new-dest",
		BackendKind:        destinations.BackendFilesystem,
		Location:           "/mnt/new",
		RepositoryLocation: "/mnt/new/repositories/new-dest.kopia",
	})
	if err != nil {
		t.Fatalf("Create new: %v", err)
	}
	roundtrip, err := repo.GetByID(ctx, created.ID)
	if err != nil {
		t.Fatalf("GetByID new: %v", err)
	}
	if roundtrip.RepositoryLocation != "/mnt/new/repositories/new-dest.kopia" {
		t.Fatalf("RepositoryLocation roundtrip = %q", roundtrip.RepositoryLocation)
	}
}
