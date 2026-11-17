package policy

import (
	"context"
	"database/sql"
	"fmt"
)

// EnsureMigrations adds the scope seam to brownfield databases. The
// operations are additive and replay-safe; the journal remains append-only.
func EnsureMigrations(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS scope_policies (
		scope TEXT PRIMARY KEY,
		frontier_target INTEGER,
		wake_budget_lines INTEGER,
		wake_budget_chars INTEGER,
		max_entry_lines INTEGER,
		max_entry_chars INTEGER,
		updated_at TEXT NOT NULL
	)`); err != nil {
		return fmt.Errorf("create scope_policies migration: %w", err)
	}
	for _, table := range []string{"entries", "facet_definitions", "facet_policies", "summaries"} {
		exists, err := tableExists(ctx, db, table)
		if err != nil {
			return err
		}
		if !exists {
			continue
		}
		if err := addScopeColumn(ctx, db, table); err != nil {
			return err
		}
	}
	if exists, err := tableExists(ctx, db, "entries"); err != nil {
		return err
	} else if exists {
		if _, err := db.ExecContext(ctx, `DROP INDEX IF EXISTS idx_entries_import_key`); err != nil {
			return fmt.Errorf("drop legacy import-key index: %w", err)
		}
		if _, err := db.ExecContext(ctx, `CREATE UNIQUE INDEX IF NOT EXISTS idx_entries_scope_import_key ON entries(scope, import_key) WHERE import_key IS NOT NULL`); err != nil {
			return fmt.Errorf("create scoped import-key index: %w", err)
		}
	}
	return nil
}

func tableExists(ctx context.Context, db *sql.DB, table string) (bool, error) {
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?`, table).Scan(&n); err != nil {
		return false, fmt.Errorf("inspect %s table: %w", table, err)
	}
	return n != 0, nil
}

func addScopeColumn(ctx context.Context, db *sql.DB, table string) error {
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info(?) WHERE name='scope'`, table).Scan(&n); err != nil {
		return fmt.Errorf("inspect %s.scope: %w", table, err)
	}
	if n == 0 {
		if _, err := db.ExecContext(ctx, `ALTER TABLE `+table+` ADD COLUMN scope TEXT NOT NULL DEFAULT 'agent-memory'`); err != nil {
			return fmt.Errorf("add %s.scope: %w", table, err)
		}
	}
	return nil
}
