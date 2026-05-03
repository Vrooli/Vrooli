package store

import (
	"context"
	"database/sql"
	_ "embed"
	"fmt"
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
func EnsureSchema(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}
