package pairing

import (
	"context"
	"database/sql"
	"fmt"
)

// Migrate evolves the pre-correlation pairing table before EnsureSchemas checks
// its declared shape. It never recreates or deletes pairing history.
func Migrate(ctx context.Context, db SQLExecutor) error {
	exists, err := pairingTableExists(ctx, db, "pairing_codes")
	if err != nil || !exists {
		return err
	}
	for _, column := range []struct{ name, ddl string }{
		{"correlation_id", "TEXT NOT NULL DEFAULT ''"},
		{"claimed_at", "TEXT NOT NULL DEFAULT ''"},
	} {
		present, err := pairingColumnExists(ctx, db, "pairing_codes", column.name)
		if err != nil {
			return err
		}
		if !present {
			if _, err = db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE pairing_codes ADD COLUMN %s %s", column.name, column.ddl)); err != nil {
				return fmt.Errorf("add pairing_codes.%s: %w", column.name, err)
			}
		}
	}
	return nil
}

func pairingTableExists(ctx context.Context, db SQLExecutor, table string) (bool, error) {
	var one int
	err := db.QueryRowContext(ctx, "SELECT 1 FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&one)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
}
func pairingColumnExists(ctx context.Context, db SQLExecutor, table, column string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%q)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var dflt any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &dflt, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
