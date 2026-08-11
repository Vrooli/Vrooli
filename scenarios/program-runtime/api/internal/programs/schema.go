package programs

import (
	"context"
	"fmt"

	_ "embed"
)

//go:embed schema.sql
var schema string

// Schema returns the programs domain schema for the central database.
func Schema() string { return schema }

// EnsureCompatibility upgrades the small SQLite schema in place for fields
// added after the initial scenario database was created. The scenario uses
// SQLite in every supported deployment, and keeping this additive migration
// here lets operators restart an existing installation without data loss.
func EnsureCompatibility(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(programs)")
	if err != nil {
		return fmt.Errorf("inspect programs schema: %w", err)
	}
	defer rows.Close()
	found := false
	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return fmt.Errorf("scan programs schema: %w", err)
		}
		if name == "agent_bytes" {
			found = true
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read programs schema: %w", err)
	}
	if found {
		return nil
	}
	if _, err := db.ExecContext(ctx, "ALTER TABLE programs ADD COLUMN agent_bytes INTEGER NOT NULL DEFAULT 0"); err != nil {
		return fmt.Errorf("add programs.agent_bytes: %w", err)
	}
	return nil
}
