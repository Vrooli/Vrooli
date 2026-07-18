package registry

import (
	"context"
	"database/sql"
	"fmt"
)

// Migrate adds the opaque pairing correlation without altering historic Node
// identity. Existing Nodes remain uncorrelated and are therefore never adopted.
func Migrate(ctx context.Context, db SQLExecutor) error {
	var one int
	err := db.QueryRowContext(ctx, "SELECT 1 FROM sqlite_master WHERE type='table' AND name='nodes'").Scan(&one)
	if err == sql.ErrNoRows {
		return nil
	}
	if err != nil {
		return err
	}
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(nodes)")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			return err
		}
		if name == "pairing_correlation_id" {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	if _, err = db.ExecContext(ctx, "ALTER TABLE nodes ADD COLUMN pairing_correlation_id TEXT NOT NULL DEFAULT ''"); err != nil {
		return fmt.Errorf("add nodes.pairing_correlation_id: %w", err)
	}
	return nil
}
