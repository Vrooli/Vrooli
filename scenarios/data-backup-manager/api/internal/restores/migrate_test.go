package restores

import (
	"context"
	"testing"

	db "github.com/vrooli/api-core/databasetest"
)

func TestEnsureColumnsAddsUpdatedAtToLegacyTable(t *testing.T) {
	database := db.NewSQLite(t)
	ctx := context.Background()
	if _, err := database.ExecContext(ctx, `CREATE TABLE restores (
		id TEXT PRIMARY KEY, target_id TEXT NOT NULL, status TEXT NOT NULL, requested_at TEXT NOT NULL
	)`); err != nil {
		t.Fatalf("create legacy table: %v", err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO restores (id, target_id, status, requested_at)
		VALUES ('legacy', 'target-1', 'verified', '2026-10-28T00:00:00Z')`); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	if err := EnsureColumns(ctx, database); err != nil {
		t.Fatalf("EnsureColumns: %v", err)
	}
	if err := EnsureColumns(ctx, database); err != nil {
		t.Fatalf("EnsureColumns (idempotent): %v", err)
	}
	var id, updatedAt string
	if err := database.QueryRowContext(ctx, "SELECT id, updated_at FROM restores WHERE id = 'legacy'").Scan(&id, &updatedAt); err != nil {
		t.Fatalf("read migrated row: %v", err)
	}
	if id != "legacy" || updatedAt != "" {
		t.Fatalf("migrated row = (%q, %q), want preserved id and empty new column", id, updatedAt)
	}
}
