package runs

import (
	"context"
	"database/sql"
	"fmt"
)

// Migrate adds delivery-state columns to an existing runs table without
// recreating durable run history. Fresh databases skip this hook and receive
// the complete shape from schema.sql.
func Migrate(ctx context.Context, db SQLExecutor) error {
	var exists int
	err := db.QueryRowContext(ctx, "SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = 'runs'").Scan(&exists)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect runs table: %w", err)
	}
	columns := []struct {
		name string
		ddl  string
	}{
		{"queued_since", "TEXT NOT NULL DEFAULT ''"},
		{"pushed_at", "TEXT NOT NULL DEFAULT ''"},
		{"acked_at", "TEXT NOT NULL DEFAULT ''"},
		{"delivery_attempts", "INTEGER NOT NULL DEFAULT 0"},
		{"last_delivery_error", "TEXT NOT NULL DEFAULT ''"},
		{"delivery_lease_expires_at", "TEXT NOT NULL DEFAULT ''"},
	}
	for _, column := range columns {
		var found int
		err := db.QueryRowContext(ctx, "SELECT 1 FROM pragma_table_info('runs') WHERE name = ?", column.name).Scan(&found)
		if err == nil {
			continue
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("inspect runs.%s: %w", column.name, err)
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE runs ADD COLUMN "+column.name+" "+column.ddl); err != nil {
			return fmt.Errorf("add runs.%s: %w", column.name, err)
		}
	}
	return nil
}
