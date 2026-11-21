package plans

import (
	"context"
	"fmt"
)

const defaultRecoveryDrillSchedule = "168h"

// EnsureColumns applies additive GFS policy migrations to existing catalogs.
func EnsureColumns(ctx context.Context, db SQLExecutor) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(plans)")
	if err != nil {
		return fmt.Errorf("inspect plans columns: %w", err)
	}
	cols := map[string]bool{}
	for rows.Next() {
		var cid, notNull, pk int
		var name, typ string
		var def any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &def, &pk); err != nil {
			rows.Close()
			return err
		}
		cols[name] = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if !cols["protection_tier"] {
		if _, err := db.ExecContext(ctx, "ALTER TABLE plans ADD COLUMN protection_tier TEXT NOT NULL DEFAULT 'full_primary'"); err != nil {
			return fmt.Errorf("add plans protection tier: %w", err)
		}
	}
	if !cols["recovery_drill_schedule"] {
		if _, err := db.ExecContext(ctx, "ALTER TABLE plans ADD COLUMN recovery_drill_schedule TEXT NOT NULL DEFAULT ''"); err != nil {
			return fmt.Errorf("add plans recovery drill schedule: %w", err)
		}
	}
	if !cols["target_selector"] {
		if _, err := db.ExecContext(ctx, "ALTER TABLE plans ADD COLUMN target_selector TEXT NOT NULL DEFAULT ''"); err != nil {
			return fmt.Errorf("add plans target selector: %w", err)
		}
	}
	for _, name := range []string{"keep_hourly", "keep_daily", "keep_weekly", "keep_monthly", "keep_annual"} {
		if cols[name] {
			continue
		}
		if _, err := db.ExecContext(ctx, "ALTER TABLE plans ADD COLUMN "+name+" INTEGER NOT NULL DEFAULT 0"); err != nil {
			return fmt.Errorf("add plans column %s: %w", name, err)
		}
	}
	// Existing enabled plans predate automatic restore drills. Bootstrap the
	// owner policy once in their durable catalog; disabled plans stay untouched.
	if _, err := db.ExecContext(ctx, "UPDATE plans SET recovery_drill_schedule = ? WHERE enabled = 1 AND trim(recovery_drill_schedule) = ''", defaultRecoveryDrillSchedule); err != nil {
		return fmt.Errorf("bootstrap weekly recovery drill schedules: %w", err)
	}
	return nil
}
