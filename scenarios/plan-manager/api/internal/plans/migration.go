package plans

import (
	"context"
	"database/sql"
	"fmt"

	"plan-manager/internal/clock"
)

const contentHashMigrationID = "20260721_complete_authored_content_hash"

// EnsureMigrations applies data migrations after Schema has created the plans
// table. The content-hash migration updates only plans.content_hash: it neither
// rewrites documents nor invokes the service, so timestamps, mirrors, audit
// facts, and graph edges remain untouched.
func EnsureMigrations(ctx context.Context, db SQLExecutor) error {
	if _, err := db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS plan_storage_migrations (id TEXT PRIMARY KEY, applied_at TEXT NOT NULL)`); err != nil {
		return fmt.Errorf("create plan migration ledger: %w", err)
	}
	var found string
	err := db.QueryRowContext(ctx, `SELECT id FROM plan_storage_migrations WHERE id = ?`, contentHashMigrationID).Scan(&found)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("inspect content-hash migration: %w", err)
	}
	repo := NewSQLiteRepository(db, clock.System{})
	plans, listErr := repo.List(ctx, ListFilter{IncludeArchived: true})
	if listErr != nil {
		return fmt.Errorf("list plans for content-hash migration: %w", listErr)
	}
	for _, plan := range plans {
		hash := contentHash(plan)
		if plan.ContentHash == hash {
			continue
		}
		if _, err := db.ExecContext(ctx, `UPDATE plans SET content_hash = ? WHERE id = ?`, hash, plan.ID); err != nil {
			return fmt.Errorf("rehash plan %q: %w", plan.ID, err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT OR IGNORE INTO plan_storage_migrations (id, applied_at) VALUES (?, datetime('now'))`, contentHashMigrationID); err != nil {
		return fmt.Errorf("record content-hash migration: %w", err)
	}
	return nil
}
