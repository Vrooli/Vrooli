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
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			return err
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, column := range []string{"pairing_correlation_id", "machine_arch", "binary_arch", "capability_inventory", "capability_probed_at", "configuration_op_id", "configuration_state", "configuration_at", "configuration_unmet"} {
		if columns[column] {
			continue
		}
		defaultValue := "''"
		if column == "capability_inventory" {
			defaultValue = "'[]'"
		}
		if column == "configuration_unmet" {
			defaultValue = "'[]'"
		}
		if _, err = db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE nodes ADD COLUMN %s TEXT NOT NULL DEFAULT %s", column, defaultValue)); err != nil {
			return fmt.Errorf("add nodes.%s: %w", column, err)
		}
	}
	return nil
}
