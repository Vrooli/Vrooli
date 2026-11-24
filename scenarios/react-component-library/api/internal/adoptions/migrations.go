package adoptions

import (
	"context"
	"database/sql"
	"fmt"
)

// migrationDB is the narrow startup/query seam shared by *sql.DB and the
// api-core RoutedDB. Keep it local to adoptions so migration needs do not
// widen the domain repository interface.
type migrationDB interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	BeginTx(context.Context, *sql.TxOptions) (*sql.Tx, error)
}

// EnsureMigrations applies adoptions-owned additive schema repair and cleans
// historical orphaned file rows. Call it after Schema has been applied and
// before constructing adoptions services. Both operations share a transaction
// so a failure leaves the database unchanged and startup can report the cause.
func EnsureMigrations(ctx context.Context, db migrationDB) error {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin adoptions migration: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	rows, err := tx.QueryContext(ctx, `PRAGMA table_info(adoption_records)`)
	if err != nil {
		return fmt.Errorf("inspect adoption_records schema: %w", err)
	}
	hasTable, hasMode := false, false
	for rows.Next() {
		var cid, notNull, primaryKey int
		var name, columnType string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			_ = rows.Close()
			return fmt.Errorf("read adoption_records schema: %w", err)
		}
		hasTable = true
		if name == "mode" {
			hasMode = true
		}
	}
	if err := rows.Err(); err != nil {
		_ = rows.Close()
		return fmt.Errorf("read adoption_records schema: %w", err)
	}
	if err := rows.Close(); err != nil {
		return fmt.Errorf("close adoption_records schema: %w", err)
	}
	if !hasTable {
		return fmt.Errorf("inspect adoption_records schema: table does not exist; apply adoptions schema first")
	}
	if !hasMode {
		if _, err := tx.ExecContext(ctx, `ALTER TABLE adoption_records ADD COLUMN mode TEXT NOT NULL DEFAULT 'copied'`); err != nil {
			return fmt.Errorf("add adoption_records.mode: %w", err)
		}
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM adoption_files WHERE NOT EXISTS (SELECT 1 FROM adoption_records WHERE adoption_records.id = adoption_files.adoption_id)`); err != nil {
		return fmt.Errorf("remove orphaned adoption_files rows: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit adoptions migration: %w", err)
	}
	return nil
}
