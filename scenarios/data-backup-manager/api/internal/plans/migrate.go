package plans

import (
	"context"
	"fmt"
)

// EnsureColumns applies additive plan migrations to existing catalogs.
func EnsureColumns(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(plans)")
	if err != nil {
		return fmt.Errorf("inspect plans columns: %w", err)
	}
	defer rows.Close()
	seen := map[string]bool{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var def any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &def, &pk); err != nil {
			return fmt.Errorf("scan plans columns: %w", err)
		}
		seen[name] = true
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate plans columns: %w", err)
	}
	if !seen["protection_tier"] {
		if _, err := db.ExecContext(ctx, "ALTER TABLE plans ADD COLUMN protection_tier TEXT NOT NULL DEFAULT 'full_primary'"); err != nil {
			return fmt.Errorf("add plans protection tier: %w", err)
		}
	}
	if !seen["recovery_drill_schedule"] {
		if _, err := db.ExecContext(ctx, "ALTER TABLE plans ADD COLUMN recovery_drill_schedule TEXT NOT NULL DEFAULT ''"); err != nil {
			return fmt.Errorf("add plans recovery drill schedule: %w", err)
		}
	}
	return nil
}
