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
	if err := preserveDelegationsOnSessionReclaim(ctx, db); err != nil {
		return err
	}
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

// preserveDelegationsOnSessionReclaim upgrades the original child table,
// whose ON DELETE CASCADE erased the evidence when a kernel session was
// reclaimed. SQLite has no ALTER CONSTRAINT, so rebuild the table once while
// retaining every existing row.
func preserveDelegationsOnSessionReclaim(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA foreign_key_list(session_delegations)")
	if err != nil {
		return fmt.Errorf("inspect delegation foreign keys: %w", err)
	}
	defer rows.Close()
	var id, seq int
	var table, from, to, onUpdate, onDelete, match string
	hasForeignKey := false
	for rows.Next() {
		if err := rows.Scan(&id, &seq, &table, &from, &to, &onUpdate, &onDelete, &match); err != nil {
			return fmt.Errorf("scan delegation foreign key: %w", err)
		}
		hasForeignKey = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read delegation foreign keys: %w", err)
	}
	if !hasForeignKey {
		return nil
	}
	for _, statement := range []string{
		"PRAGMA foreign_keys=OFF",
		"CREATE TABLE session_delegations_new (session_id TEXT NOT NULL, execution_id TEXT PRIMARY KEY, owner TEXT NOT NULL, workflow_key TEXT NOT NULL, created_at TEXT NOT NULL, last_status TEXT NOT NULL DEFAULT '')",
		"INSERT INTO session_delegations_new SELECT session_id, execution_id, owner, workflow_key, created_at, last_status FROM session_delegations",
		"DROP TABLE session_delegations",
		"ALTER TABLE session_delegations_new RENAME TO session_delegations",
		"CREATE INDEX IF NOT EXISTS idx_session_delegations_session ON session_delegations(session_id, created_at)",
		"PRAGMA foreign_keys=ON",
	} {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			return fmt.Errorf("migrate delegation retention (%s): %w", statement, err)
		}
	}
	return nil
}
