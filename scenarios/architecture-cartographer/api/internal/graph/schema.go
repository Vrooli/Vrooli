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
	if _, err := db.ExecContext(ctx, addSourceFingerprintColumnSQL); err != nil && !isIgnorableSourceFingerprintColumnMigration(err) {
		return fmt.Errorf("migrate graph schema: %w", err)
	}
	return nil
}

func isIgnorableSourceFingerprintColumnMigration(err error) bool {
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "duplicate column") ||
		strings.Contains(msg, "already exists") ||
		strings.Contains(msg, "no such table")
}
