package reconcile

import (
	"context"
	_ "embed"
	"fmt"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the reconcile domain's SQL contribution.
func Schema() string { return schemaSQL }

// EnsureMigrations upgrades existing evidence stores with the component
// identity columns added after the original page-only projection.
func EnsureMigrations(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(reconcile_evidence)")
	if err != nil {
		return fmt.Errorf("inspect reconcile evidence schema: %w", err)
	}
	defer rows.Close()
	present := map[string]bool{}
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return fmt.Errorf("scan reconcile evidence schema: %w", err)
		}
		present[name] = true
	}
	for _, change := range []struct{ name, sql string }{
		{"document_kind", "ALTER TABLE reconcile_evidence ADD COLUMN document_kind TEXT NOT NULL DEFAULT 'page'"},
		{"component_id", "ALTER TABLE reconcile_evidence ADD COLUMN component_id TEXT NOT NULL DEFAULT ''"},
		{"component_title", "ALTER TABLE reconcile_evidence ADD COLUMN component_title TEXT NOT NULL DEFAULT ''"},
		{"example_name", "ALTER TABLE reconcile_evidence ADD COLUMN example_name TEXT NOT NULL DEFAULT ''"},
	} {
		if present[change.name] {
			continue
		}
		if _, err := db.ExecContext(ctx, change.sql); err != nil {
			return fmt.Errorf("migrate reconcile evidence %s: %w", change.name, err)
		}
	}
	return rows.Err()
}
