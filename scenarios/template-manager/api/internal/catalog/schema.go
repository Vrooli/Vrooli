package catalog

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the template-manager catalog schema. It is intentionally
// forward-only and idempotent so startup can run it on every boot.
func Schema() string { return schemaSQL }

// MigrateDriftDebtIdentity adds an explicit scenario identity to existing debt
// rows. The schema remains forward-only; this narrow SQLite migration upgrades
// databases created before scenario_slug was part of the drift-debt contract.
func MigrateDriftDebtIdentity(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(debt_entries)`)
	if err != nil {
		return fmt.Errorf("inspect debt_entries schema: %w", err)
	}
	hasScenarioSlug := false
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue sql.NullString
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			return fmt.Errorf("read debt_entries schema: %w", err)
		}
		if name == "scenario_slug" {
			hasScenarioSlug = true
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("iterate debt_entries schema: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close debt_entries schema: %w", err)
	}
	if !hasScenarioSlug {
		if _, err := db.ExecContext(ctx, `ALTER TABLE debt_entries ADD COLUMN scenario_slug TEXT NOT NULL DEFAULT ''`); err != nil {
			return fmt.Errorf("add drift debt scenario identity: %w", err)
		}
	}
	if _, err := db.ExecContext(ctx, `
UPDATE debt_entries
SET scenario_slug = CASE
  WHEN substr(title, -length(' has template drift')) = ' has template drift'
    THEN substr(title, 1, length(title) - length(' has template drift'))
  WHEN substr(title, -length(' template drift could not be verified')) = ' template drift could not be verified'
    THEN substr(title, 1, length(title) - length(' template drift could not be verified'))
  ELSE ''
END
WHERE source = 'template drift' AND scenario_slug = ''
  AND (substr(title, -length(' has template drift')) = ' has template drift'
    OR substr(title, -length(' template drift could not be verified')) = ' template drift could not be verified')`); err != nil {
		return fmt.Errorf("backfill drift debt scenario identities: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_debt_entries_scenario_source_status ON debt_entries(scenario_slug, source, status)`); err != nil {
		return fmt.Errorf("index drift debt scenario identities: %w", err)
	}
	return nil
}
