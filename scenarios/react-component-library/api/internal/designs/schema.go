package designs

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the designs domain's SQL contribution. Applied by
// database.EnsureSchemas via the modules.AllSchemas() registry.
func Schema() string { return schemaSQL }

// Migrate applies the additive, brownfield part of the designs schema.
//
// schema.sql runs with CREATE TABLE IF NOT EXISTS, which does not add columns
// to an existing studio_designs table. SQLite cannot express
// ADD COLUMN IF NOT EXISTS, so the catalog is inspected explicitly. Migrate is
// idempotent and never drops or rewrites user data; call it once after
// EnsureSchemas at boot.
func Migrate(ctx context.Context, db *sql.DB) error {
	// Re-apply the declarative schema first. Every statement is IF NOT EXISTS;
	// the effect on an old database is to add the new asset table (and its
	// index) before the column-dependent statements below.
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply designs schema: %w", err)
	}
	if err := ensureColumn(ctx, db, "studio_designs", "scenario", "TEXT"); err != nil {
		return err
	}
	if err := ensureColumn(ctx, db, "studio_designs", "page", "TEXT"); err != nil {
		return err
	}
	// options_json is NOT NULL with a literal default so existing rows read back
	// as {} without a rewrite. SQLite requires the default for ADD COLUMN when
	// the column is NOT NULL.
	if err := ensureColumn(ctx, db, "studio_designs", "options_json", "TEXT NOT NULL DEFAULT '{}'"); err != nil {
		return err
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS studio_designs_binding ON studio_designs (scenario, page)`); err != nil {
		return fmt.Errorf("index studio_designs binding: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS studio_design_assets_design ON studio_design_assets (design_id, created_at)`); err != nil {
		return fmt.Errorf("index studio_design_assets design: %w", err)
	}
	return nil
}

func ensureColumn(ctx context.Context, db *sql.DB, table, column, definition string) error {
	var count int
	query := fmt.Sprintf("SELECT COUNT(*) FROM pragma_table_info('%s') WHERE name = ?", table)
	if err := db.QueryRowContext(ctx, query, column).Scan(&count); err != nil {
		return fmt.Errorf("inspect %s %s: %w", table, column, err)
	}
	if count > 0 {
		return nil
	}
	statement := fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", table, column, definition)
	if _, err := db.ExecContext(ctx, statement); err != nil {
		return fmt.Errorf("add %s %s: %w", table, column, err)
	}
	return nil
}
