package graph

import (
	"context"
	"fmt"
	"strings"

	_ "embed"
)

//go:embed schema.sql
var schemaSQL string

// Schema returns the graph domain's SQL contribution.
func Schema() string { return schemaSQL }

// MigrateSchema applies graph-owned one-shot migrations that must run before
// api-core's schema drift check. Fresh databases may not have the table yet;
// that is fine because Schema creates it with the latest columns.
func MigrateSchema(ctx context.Context, db SQLExecutor) error {
	// Under SQLite, adding a column to a CREATE TABLE IF NOT EXISTS block is a
	// silent no-op on a database that already has the table, so every added
	// column needs an explicit ALTER here. Each is idempotent: a duplicate
	// column or a missing table is ignorable, which makes this safe to run on
	// every startup and safe to re-run.
	migrations := []string{
		addSourceFingerprintColumnSQL,
		addPayloadCodecColumnSQL,
	}
	for _, migration := range migrations {
		if _, err := db.ExecContext(ctx, migration); err != nil && !isIgnorableColumnMigration(err) {
			return fmt.Errorf("migrate graph schema: %w", err)
		}
	}
	return nil
}

func isIgnorableColumnMigration(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "no such table")
}
