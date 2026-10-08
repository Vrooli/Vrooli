// DOC: docs/internal/STORAGE_AUDIT.md#Schema-Status
// DOC: README.md#Event-Schema
// DOC: PRD.md#OT-P0-001
//
// Package domain contains the core business entities for the lifestyle dashboard.
package domain

import (
	"database/sql"

	storageSchema "lifestyle-dashboard/internal/lifestyle"
)

// InitSchema is retained for focused repository tests and compatibility with
// callers that construct an isolated SQLite database. Production startup uses
// the same embedded provider through api-core/database.EnsureSchemas.
func InitSchema(db *sql.DB) error {
	_, err := db.Exec(storageSchema.Schema())
	return err
}
