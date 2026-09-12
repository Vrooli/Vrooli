// DOC: docs/concepts/ARCHITECTURE.md#data-model
package main

import (
	"database/sql"

	storageSchema "stream-of-consciousness-analyzer/internal/thoughts"
)

// ensureSchema is retained for integration tests that provision an isolated
// database. Production startup applies the same embedded provider through
// api-core/database.EnsureSchemas.
func ensureSchema(db *sql.DB) error {
	_, err := db.Exec(storageSchema.Schema())
	return err
}
