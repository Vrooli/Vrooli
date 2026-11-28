package artifacts

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// Migrate adds closure shipment receipt inventory storage to existing Bridge
// databases before schema drift validation runs. Fresh databases are handled
// by the domain schema.
func Migrate(ctx context.Context, db *sql.DB) error {
	var found int
	err := db.QueryRowContext(ctx, "SELECT 1 FROM sqlite_master WHERE type='table' AND name='closure_shipments'").Scan(&found)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect closure shipments table: %w", err)
	}
	for _, column := range []struct{ name, ddl string }{
		{"actual_files_json", "ALTER TABLE closure_shipments ADD COLUMN actual_files_json TEXT NOT NULL DEFAULT '[]'"},
		{"selection_receipt_ref", "ALTER TABLE closure_shipments ADD COLUMN selection_receipt_ref TEXT"},
		{"selection_digest", "ALTER TABLE closure_shipments ADD COLUMN selection_digest TEXT NOT NULL DEFAULT ''"},
		{"deployment_id", "ALTER TABLE closure_shipments ADD COLUMN deployment_id TEXT NOT NULL DEFAULT ''"},
		{"machine_id", "ALTER TABLE closure_shipments ADD COLUMN machine_id TEXT NOT NULL DEFAULT ''"},
		{"node_id", "ALTER TABLE closure_shipments ADD COLUMN node_id TEXT NOT NULL DEFAULT ''"},
		{"enrollment_generation", "ALTER TABLE closure_shipments ADD COLUMN enrollment_generation INTEGER NOT NULL DEFAULT 0"},
		{"desired_revision", "ALTER TABLE closure_shipments ADD COLUMN desired_revision INTEGER NOT NULL DEFAULT 0"},
	} {
		err = db.QueryRowContext(ctx, "SELECT 1 FROM pragma_table_info('closure_shipments') WHERE name=?", column.name).Scan(&found)
		if errors.Is(err, sql.ErrNoRows) {
			if _, err := db.ExecContext(ctx, column.ddl); err != nil {
				return fmt.Errorf("add closure shipment %s: %w", column.name, err)
			}
			continue
		}
		if err != nil {
			return fmt.Errorf("inspect closure shipment %s: %w", column.name, err)
		}
	}
	if _, err := db.ExecContext(ctx, "CREATE UNIQUE INDEX IF NOT EXISTS idx_closure_shipments_selection_receipt ON closure_shipments(selection_receipt_ref) WHERE selection_receipt_ref IS NOT NULL"); err != nil {
		return fmt.Errorf("index closure shipment selection receipt: %w", err)
	}
	return nil
}
