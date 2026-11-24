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

// EnsureCompatibility adds fields introduced after the initial programs
// table was created. Existing records default to unverified attribution.
func EnsureCompatibility(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(programs)")
	if err != nil {
		return fmt.Errorf("inspect programs schema: %w", err)
	}
	defer rows.Close()
	columns := make(map[string]bool)
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return fmt.Errorf("scan programs schema: %w", err)
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate programs schema: %w", err)
	}
	for _, name := range []string{"caller_identity_verified", "source_receipt_format", "stdout_receipt_format", "failure_detail_receipt_format"} {
		if !columns[name] {
			if _, err := db.ExecContext(ctx, "ALTER TABLE programs ADD COLUMN "+name+" INTEGER NOT NULL DEFAULT 0"); err != nil {
				return fmt.Errorf("add %s to programs: %w", name, err)
			}
		}
	}
	if !columns["service_owner_json"] {
		if _, err := db.ExecContext(ctx, "ALTER TABLE programs ADD COLUMN service_owner_json TEXT NOT NULL DEFAULT ''"); err != nil {
			return fmt.Errorf("add service ownership to programs: %w", err)
		}
	}
	return nil
}
