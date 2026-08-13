package sessions

import (
	"context"
	"fmt"

	_ "embed"
)

//go:embed schema.sql
var schema string

// Schema returns the sessions domain schema for the central database
// bootstrap. The SQL is idempotent so lifecycle restarts are safe.
func Schema() string { return schema }

// EnsureCompatibility adds execution-budget columns to databases created by
// earlier program-runtime versions without deleting any session evidence.
func EnsureCompatibility(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(sessions)")
	if err != nil {
		return fmt.Errorf("inspect sessions schema: %w", err)
	}
	defer rows.Close()
	found := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primary int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primary); err != nil {
			return fmt.Errorf("scan sessions schema: %w", err)
		}
		found[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read sessions schema: %w", err)
	}
	columns := []struct{ name, definition string }{
		{"wall_budget_millis", "INTEGER NOT NULL DEFAULT 14400000"},
		{"wall_consumed_millis", "INTEGER NOT NULL DEFAULT 0"},
		{"cpu_budget_millis", "INTEGER NOT NULL DEFAULT 14400000"},
		{"cpu_consumed_millis", "INTEGER NOT NULL DEFAULT 0"},
	}
	for _, column := range columns {
		if found[column.name] {
			continue
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE sessions ADD COLUMN "+column.name+" "+column.definition); err != nil {
			return fmt.Errorf("add sessions.%s: %w", column.name, err)
		}
	}
	return nil
}
