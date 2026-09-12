package components

import (
	"context"
	"database/sql"
	"fmt"

	_ "embed"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the components domain's SQL contribution. Applied by
// database.EnsureSchemas via the modules.AllSchemas() registry.
// Forward-only declarative — re-runs are no-ops (CREATE TABLE IF NOT
// EXISTS). New columns land as ALTER TABLE … ADD COLUMN IF NOT EXISTS
// appended to schema.sql.
func Schema() string { return schemaSQL }

// EnsureMigrations applies the additive brownfield part of the presence
// change before indexes or readers refer to the new column. SQLite cannot
// express ADD COLUMN IF NOT EXISTS, so the catalog is inspected explicitly.
func EnsureMigrations(ctx context.Context, db *sql.DB) error {
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('component_versions') WHERE name='presence'`).Scan(&count); err != nil {
		return fmt.Errorf("inspect component_versions presence: %w", err)
	}
	if count == 0 {
		if _, err := db.ExecContext(ctx, `ALTER TABLE component_versions ADD COLUMN presence TEXT NOT NULL DEFAULT 'materialized'`); err != nil {
			return fmt.Errorf("add component_versions presence: %w", err)
		}
	}
	if err := ensureColumn(ctx, db, "component_test_reports", "source_revision", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_component_versions_component_presence ON component_versions(component_id, presence)`); err != nil {
		return fmt.Errorf("index component_versions presence: %w", err)
	}
	return nil
}

func ensureColumn(ctx context.Context, db *sql.DB, table, column, definition string) error {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name=?", table)
	if err := db.QueryRowContext(ctx, query, column).Scan(&count); err != nil {
		return fmt.Errorf("inspect %s %s: %w", table, column, err)
	}
	if count == 0 {
		if _, err := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)); err != nil {
			return fmt.Errorf("add %s %s: %w", table, column, err)
		}
	}
	return nil
}
