package templates

import (
	"context"
	"database/sql"
	"fmt"
)

// Migrate applies the templates schema.
//
// studio_templates is a new table, not a column added to an existing one, so
// there is no brownfield ALTER TABLE step: re-applying the declarative
// CREATE TABLE/INDEX IF NOT EXISTS statements is the whole migration and is a
// no-op on every boot after the first.
func Migrate(ctx context.Context, db *sql.DB) error {
	if _, err := db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply templates schema: %w", err)
	}
	return nil
}
