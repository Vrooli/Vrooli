package machines

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// Migrate brings an existing Machine schema to the declared shape before
// EnsureSchemas runs its drift check. A fresh database has no review table yet,
// so it remains a no-op and EnsureSchemas creates the complete schema.
func Migrate(ctx context.Context, db SQLExecutor) error {
	exists, err := migrationTableExists(ctx, db, "machine_migration_reviews")
	if err != nil || !exists {
		return err
	}
	var found int
	err = db.QueryRowContext(ctx, "SELECT 1 FROM pragma_table_info('machine_migration_reviews') WHERE name='confidence'").Scan(&found)
	if err == nil {
		return nil
	}
	if err != sql.ErrNoRows {
		return fmt.Errorf("inspect machine_migration_reviews.confidence: %w", err)
	}
	if _, err = db.ExecContext(ctx, "ALTER TABLE machine_migration_reviews ADD COLUMN confidence TEXT NOT NULL DEFAULT 'ambiguous'"); err != nil {
		return fmt.Errorf("add machine_migration_reviews.confidence: %w", err)
	}
	return nil
}

// BackfillLegacy records legacy Registry Nodes and onboarding operations as
// reviewable evidence. Neither legacy record contains the durable pairing
// correlation needed to prove which pre-contact Machine it belongs to, so this
// deliberately creates no Machine and never rewrites historic records.
//
// It is safe on every boot: the unique source/id key makes each evidence record
// immutable and idempotent. Call it after EnsureSchemas, when this domain's
// review table is guaranteed to exist.
func BackfillLegacy(ctx context.Context, db SQLExecutor) error {
	now := time.Now().UTC().Format(machineTimeFormat)
	for _, source := range []struct {
		table  string
		reason string
	}{
		{"nodes", "registry node has no durable Machine enrollment correlation"},
		{"onboarding_ops", "legacy onboarding operation has no durable Machine enrollment correlation"},
	} {
		exists, err := migrationTableExists(ctx, db, source.table)
		if err != nil {
			return fmt.Errorf("inspect legacy %s: %w", source.table, err)
		}
		if !exists {
			continue
		}
		query := fmt.Sprintf(`INSERT OR IGNORE INTO machine_migration_reviews
            (id,legacy_source,legacy_id,status,confidence,reason,created_at)
            SELECT 'legacy:' || ? || ':' || id, ?, id, 'needs_review', 'ambiguous', ?, ? FROM %s`, source.table)
		if _, err := db.ExecContext(ctx, query, source.table, source.table, source.reason, now); err != nil {
			return fmt.Errorf("preserve legacy %s for review: %w", source.table, err)
		}
	}
	return nil
}

func migrationTableExists(ctx context.Context, db SQLExecutor, table string) (bool, error) {
	var one int
	err := db.QueryRowContext(ctx,
		"SELECT 1 FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&one)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
}
