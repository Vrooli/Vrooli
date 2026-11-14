package control

import (
	"context"
	"fmt"
)

// EnsureIdentitySource upgrades databases created before stable controller
// identity became part of enrollment. SQLite does not support ADD COLUMN IF
// NOT EXISTS, so inspect the table before applying this forward-only change.
func EnsureIdentitySource(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(lighting_controllers)")
	if err != nil {
		return fmt.Errorf("inspect lighting controller schema: %w", err)
	}
	found := false
	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, primaryKey int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &primaryKey); err != nil {
			return fmt.Errorf("read lighting controller schema: %w", err)
		}
		if name == "identity_source" {
			found = true
			break
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read lighting controller schema: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close lighting controller schema: %w", err)
	}
	if found {
		return nil
	}
	if _, err := db.ExecContext(ctx, `ALTER TABLE lighting_controllers ADD COLUMN identity_source TEXT NOT NULL DEFAULT ''`); err != nil {
		return fmt.Errorf("add controller identity source: %w", err)
	}
	return nil
}
