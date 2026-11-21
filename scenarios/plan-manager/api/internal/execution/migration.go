package execution

import (
	"context"
	"database/sql"
	"fmt"
)

// EnsureMigrations replaces only the redundant completion flag. Plan documents,
// handoffs, velocity and audit records remain intact. Existing full handoffs
// prove completion; all phases done without one means ready, never completed.
// Run before EnsureSchemas so legacy stores acquire the new canonical column.
func EnsureMigrations(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, "PRAGMA table_info(executions)")
	if err != nil {
		return err
	}
	exists, legacy, current := false, false, false
	for rows.Next() {
		var index, notnull, pk int
		var name, kind string
		var value sql.NullString
		if err := rows.Scan(&index, &name, &kind, &notnull, &value, &pk); err != nil {
			rows.Close()
			return err
		}
		exists = true
		legacy = legacy || name == "complete"
		current = current || name == "lifecycle_state"
	}
	err = rows.Err()
	rows.Close()
	if err != nil || !exists || current && !legacy {
		return err
	}
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if !current {
		if _, err = tx.ExecContext(ctx, "ALTER TABLE executions ADD COLUMN lifecycle_state TEXT NOT NULL DEFAULT 'active' CHECK (lifecycle_state IN ('active','ready','completed','abandoned'))"); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, schemaSQL); err != nil {
			return err
		}
		if _, err = tx.ExecContext(ctx, `UPDATE executions SET lifecycle_state=CASE
      WHEN EXISTS (SELECT 1 FROM execution_scope_states s WHERE s.execution_id=executions.id AND json_extract(s.document,'$.lifecycle_state')='abandoned') THEN 'abandoned'
      WHEN EXISTS (SELECT 1 FROM handoffs h WHERE h.execution_id=executions.id AND h.completeness='full') THEN 'completed'
      WHEN current_phase_id='' THEN 'ready'
      ELSE 'active' END`); err != nil {
			return fmt.Errorf("infer legacy execution states: %w", err)
		}
	}
	if legacy {
		if _, err = tx.ExecContext(ctx, "ALTER TABLE executions DROP COLUMN complete"); err != nil {
			return fmt.Errorf("retire derived completion flag: %w", err)
		}
	}
	return tx.Commit()
}
