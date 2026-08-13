package bindings

import (
	"context"
	"fmt"
	"program-runtime/internal/sessions"
)

import _ "embed"

//go:embed schema.sql
var schema string

// Schema returns the bindings domain schema for the central database.
func Schema() string { return schema }

func EnsureCompatibility(ctx context.Context, db sessions.SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(binding_invocations)")
	if err != nil {
		return fmt.Errorf("inspect binding invocation schema: %w", err)
	}
	defer rows.Close()
	found := map[string]bool{}
	for rows.Next() {
		var cid, notNull, primary int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primary); err != nil {
			return err
		}
		found[name] = true
	}
	for _, column := range []struct{ name, definition string }{{"origin", "TEXT NOT NULL DEFAULT 'organic'"}, {"invocation_class", "TEXT NOT NULL DEFAULT 'success'"}} {
		if found[column.name] {
			continue
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE binding_invocations ADD COLUMN "+column.name+" "+column.definition); err != nil {
			return fmt.Errorf("add binding_invocations.%s: %w", column.name, err)
		}
	}
	_, err = db.ExecContext(ctx, `UPDATE binding_invocations SET origin = CASE WHEN program_id = 'sweep' OR provenance = 'PROVENANCE_OPERATOR' OR provenance = '' THEN 'synthetic' ELSE 'organic' END WHERE origin = '' OR origin IS NULL`)
	if err != nil {
		return fmt.Errorf("backfill invocation origin: %w", err)
	}
	_, err = db.ExecContext(ctx, `UPDATE binding_invocations SET invocation_class = CASE WHEN outcome = 'success' THEN 'success' WHEN outcome = 'refused' THEN 'refused' WHEN lower(reason) LIKE '%invalid arguments%' OR lower(reason) LIKE '%missing required field%' THEN 'probe_invalid_argument' WHEN lower(reason) LIKE '%deadline%' OR lower(reason) LIKE '%timeout%' THEN 'probe_timeout' WHEN lower(reason) LIKE '%404%' OR lower(reason) LIKE '%page not found%' OR lower(reason) LIKE '%connection refused%' THEN 'target_unavailable' ELSE 'target_failed' END WHERE invocation_class = '' OR invocation_class IS NULL`)
	if err != nil {
		return fmt.Errorf("backfill invocation class: %w", err)
	}
	if _, err := db.ExecContext(ctx, "CREATE INDEX IF NOT EXISTS idx_binding_invocations_class ON binding_invocations(invocation_class)"); err != nil {
		return fmt.Errorf("index binding invocation class: %w", err)
	}
	return nil
}
