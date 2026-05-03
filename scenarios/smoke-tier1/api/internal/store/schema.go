package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
	"strings"
)

// schemaSQL is the canonical SQLite schema for the scenario. Embedded so
// the binary needs no on-disk schema file at runtime.
//
//go:embed schema.sql
var schemaSQL string

// EnsureSchema applies schemaSQL to db. Statements use CREATE TABLE IF
// NOT EXISTS / equivalent guards so the call is idempotent — main.go
// invokes it on every boot and tests that need real tables can reuse
// the same entry point.
//
// Empty / comment-only schemas are a no-op; the placeholder template
// ships zero tables and EnsureSchema returns nil.
func EnsureSchema(ctx context.Context, db *sql.DB) error {
	script := strings.TrimSpace(schemaSQL)
	if stripComments(script) == "" {
		return nil
	}
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

// stripComments removes leading -- comments and blank lines so a
// placeholder schema (the template default) is recognised as empty
// without round-tripping through the SQLite parser.
func stripComments(script string) string {
	var out strings.Builder
	for _, line := range strings.Split(script, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "--") {
			continue
		}
		out.WriteString(trimmed)
		out.WriteByte('\n')
	}
	return strings.TrimSpace(out.String())
}
