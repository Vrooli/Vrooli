package credentialgrant

import (
	"context"
	"fmt"
)

// Migrate adds receipt evidence to an existing grant table. The receipt is
// metadata only; credential values never enter this table.
func Migrate(ctx context.Context, db SQLExecutor) error {
	exists, err := tableExists(ctx, db, "credential_grants")
	if err != nil || !exists {
		return err
	}
	for _, column := range []string{"receipt_at", "receipt_accepted", "receipt_reason", "purge_state", "purge_receipt_at", "purge_accepted", "purge_reason", "revision"} {
		has, checkErr := columnExists(ctx, db, "credential_grants", column)
		if checkErr != nil {
			return fmt.Errorf("inspect credential_grants.%s: %w", column, checkErr)
		}
		if has {
			continue
		}
		columnType, defaultValue := "TEXT", "''"
		if column == "revision" {
			columnType, defaultValue = "INTEGER", "1"
		}
		if column == "receipt_accepted" {
			columnType, defaultValue = "INTEGER", "0"
		}
		if column == "purge_accepted" {
			columnType, defaultValue = "INTEGER", "0"
		}
		if _, alterErr := db.ExecContext(ctx, fmt.Sprintf("ALTER TABLE credential_grants ADD COLUMN %s %s NOT NULL DEFAULT %s", column, columnType, defaultValue)); alterErr != nil {
			return fmt.Errorf("add credential_grants.%s: %w", column, alterErr)
		}
	}
	if _, err := db.ExecContext(ctx, "CREATE TABLE IF NOT EXISTS credential_grant_mutation_receipts (actor_id TEXT NOT NULL, request_id TEXT NOT NULL PRIMARY KEY, operation TEXT NOT NULL, node_id TEXT NOT NULL, enrollment_generation INTEGER NOT NULL, subject TEXT NOT NULL, expected_revision INTEGER NOT NULL, result_revision INTEGER NOT NULL, payload_digest TEXT NOT NULL, grant_id TEXT NOT NULL, outcome TEXT NOT NULL, delivery_state TEXT NOT NULL DEFAULT 'pending', created_at TEXT NOT NULL)"); err != nil {
		return fmt.Errorf("create credential grant mutation receipts: %w", err)
	}
	hasDeliveryState, err := columnExists(ctx, db, "credential_grant_mutation_receipts", "delivery_state")
	if err != nil {
		return fmt.Errorf("inspect credential_grant_mutation_receipts.delivery_state: %w", err)
	}
	if !hasDeliveryState {
		if _, err := db.ExecContext(ctx, "ALTER TABLE credential_grant_mutation_receipts ADD COLUMN delivery_state TEXT NOT NULL DEFAULT 'pending'"); err != nil {
			return fmt.Errorf("add credential_grant_mutation_receipts.delivery_state: %w", err)
		}
	}
	return nil
}

func tableExists(ctx context.Context, db SQLExecutor, name string) (bool, error) {
	row := db.QueryRowContext(ctx, "SELECT 1 FROM sqlite_master WHERE type='table' AND name=?", name)
	var one int
	if err := row.Scan(&one); err != nil {
		return false, nil
	}
	return true, nil
}

func columnExists(ctx context.Context, db SQLExecutor, table, column string) (bool, error) {
	rows, err := db.QueryContext(ctx, fmt.Sprintf("PRAGMA table_info(%q)", table))
	if err != nil {
		return false, err
	}
	defer rows.Close()
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
