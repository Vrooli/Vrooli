package drills

import (
	"context"
	"fmt"

	"data-backup-manager/internal/dbschema"
)

var addedColumns = []struct {
	name string
	ddl  string
}{
	{name: "notification_state", ddl: "ALTER TABLE recovery_drills ADD COLUMN notification_state TEXT NOT NULL DEFAULT ''"},
	{name: "notification_sent_at", ddl: "ALTER TABLE recovery_drills ADD COLUMN notification_sent_at TEXT NOT NULL DEFAULT ''"},
	{name: "retired_at", ddl: "ALTER TABLE recovery_drills ADD COLUMN retired_at TEXT NOT NULL DEFAULT ''"},
	{name: "retirement_reason", ddl: "ALTER TABLE recovery_drills ADD COLUMN retirement_reason TEXT NOT NULL DEFAULT ''"},
}

// EnsureColumns applies additive, idempotent notification and retirement columns.
// Old failed rows are marked not_replayed so startup does not replay historical alerts.
func EnsureColumns(ctx context.Context, db SQLExecutor) error {
	existing, err := dbschema.TableColumns(ctx, db, "recovery_drills")
	if err != nil {
		return err
	}
	for _, column := range addedColumns {
		if _, ok := existing[column.name]; ok {
			continue
		}
		if _, err := db.ExecContext(ctx, column.ddl); err != nil {
			return fmt.Errorf("add recovery drill column %q: %w", column.name, err)
		}
	}
	if _, err := db.ExecContext(ctx, `UPDATE recovery_drills SET notification_state=? WHERE status=? AND notification_state=''`, string(NotificationUntracked), string(StatusFailed)); err != nil {
		return fmt.Errorf("mark legacy recovery drill failures as untracked: %w", err)
	}
	if _, err := db.ExecContext(ctx, `CREATE INDEX IF NOT EXISTS idx_recovery_drills_notification ON recovery_drills(notification_state, requested_at)`); err != nil {
		return fmt.Errorf("index pending recovery drill notifications: %w", err)
	}
	return nil
}
