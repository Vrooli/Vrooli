package restores

import (
	"context"
	"fmt"

	"data-backup-manager/internal/dbschema"
)

// addedColumns lists columns introduced on the restores table after it first
// shipped. SQLite has no "ADD COLUMN IF NOT EXISTS", so EnsureColumns
// introspects PRAGMA table_info and adds only the missing ones — keeping boot
// idempotent on both fresh and existing databases. New columns must also be
// present in schema.sql so a fresh CREATE TABLE includes them.
var addedColumns = []struct {
	name string
	ddl  string
}{
	{name: "updated_at", ddl: "ALTER TABLE restores ADD COLUMN updated_at TEXT NOT NULL DEFAULT ''"},
}

// EnsureColumns applies additive column migrations to an existing restores
// table. It is safe to call on every boot and on a freshly created table. Call
// it after EnsureSchemas has created the base table. This is what lets the
// async-restore heartbeat column (updated_at) land on a database that predates
// it without losing restore history.
func EnsureColumns(ctx context.Context, db SQLExecutor) error {
	existing, err := dbschema.TableColumns(ctx, db, "restores")
	if err != nil {
		return err
	}
	for _, col := range addedColumns {
		if _, ok := existing[col.name]; ok {
			continue
		}
		if _, err := db.ExecContext(ctx, col.ddl); err != nil {
			return fmt.Errorf("add restores column %q: %w", col.name, err)
		}
	}
	return nil
}
