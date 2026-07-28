package forest

import (
	"context"
	"database/sql"
	"fmt"
)

// EnsureMigrations adds cache columns without recreating the derived forest.
// Existing summaries receive an empty vector and become rankable again on the
// next rebuild or compaction pass.
func EnsureMigrations(ctx context.Context, db *sql.DB) error {
	var exists int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='summaries'`).Scan(&exists); err != nil {
		return fmt.Errorf("inspect summaries table: %w", err)
	}
	if exists == 0 {
		return nil
	}
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM pragma_table_info('summaries') WHERE name='vector_json'`).Scan(&count); err != nil {
		return fmt.Errorf("inspect summaries.vector_json: %w", err)
	}
	if count == 0 {
		if _, err := db.ExecContext(ctx, `ALTER TABLE summaries ADD COLUMN vector_json TEXT NOT NULL DEFAULT '[]'`); err != nil {
			return fmt.Errorf("add summaries.vector_json: %w", err)
		}
	}
	return nil
}
