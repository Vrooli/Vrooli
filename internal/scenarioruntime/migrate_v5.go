package scenarioruntime

import (
	"context"
	"database/sql"
	"fmt"
)

// migrateV4ToV5 upgrades a schema-version-4 runtime registry to version 5 in
// place, preserving every row. Version 5 adds the `variant` dimension:
//   - runtime_instances gains a `variant` column and its uniqueness constraint
//     changes from (scenario, generation) to (scenario, variant, generation),
//     which requires a table rebuild (SQLite cannot drop a table-level UNIQUE
//     constraint in place);
//   - runtime_port_claims gains a denormalized `variant` column (a plain
//     ADD COLUMN, no rebuild).
//
// The registry is the source of truth for LIVE processes, so this migration is
// strictly row-preserving — every existing instance is copied forward with
// variant 'live', keeping running scenarios addressable. A destructive
// drop-and-recreate (the generic "greenfield rebuild" path) would orphan them.
//
// runtime_port_claims has a FOREIGN KEY ... ON DELETE CASCADE onto
// runtime_instances, so dropping the old instances table with foreign keys
// enabled would cascade-delete every claim. Foreign-key enforcement is
// therefore disabled for the rebuild (it cannot be toggled inside a
// transaction) and re-validated with PRAGMA foreign_key_check before commit.
func (s *SQLiteStore) migrateV4ToV5(ctx context.Context) error {
	conn, err := s.db.Conn(ctx)
	if err != nil {
		return fmt.Errorf("acquire migration connection: %w", err)
	}
	defer conn.Close()

	if _, err := conn.ExecContext(ctx, `PRAGMA foreign_keys=OFF`); err != nil {
		return fmt.Errorf("disable foreign keys: %w", err)
	}
	// Always restore the FK enforcement the rest of the store relies on, even
	// on the error paths below.
	restoreForeignKeys := func() error {
		_, e := conn.ExecContext(ctx, `PRAGMA foreign_keys=ON`)
		return e
	}

	tx, err := conn.BeginTx(ctx, nil)
	if err != nil {
		_ = restoreForeignKeys()
		return fmt.Errorf("begin migration transaction: %w", err)
	}

	for _, stmt := range migrateV5Statements {
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			_ = tx.Rollback()
			_ = restoreForeignKeys()
			return fmt.Errorf("apply migration step: %w", err)
		}
	}

	violations, err := foreignKeyViolations(ctx, tx)
	if err != nil {
		_ = tx.Rollback()
		_ = restoreForeignKeys()
		return err
	}
	if violations > 0 {
		_ = tx.Rollback()
		_ = restoreForeignKeys()
		return fmt.Errorf("migration left %d foreign-key violation(s); rolled back", violations)
	}

	if err := tx.Commit(); err != nil {
		_ = restoreForeignKeys()
		return fmt.Errorf("commit migration transaction: %w", err)
	}
	if err := restoreForeignKeys(); err != nil {
		return fmt.Errorf("re-enable foreign keys: %w", err)
	}
	if _, err := conn.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d`, SchemaVersion)); err != nil {
		return fmt.Errorf("stamp runtime registry schema version: %w", err)
	}
	return nil
}

func foreignKeyViolations(ctx context.Context, tx *sql.Tx) (int, error) {
	rows, err := tx.QueryContext(ctx, `PRAGMA foreign_key_check`)
	if err != nil {
		return 0, fmt.Errorf("foreign key check: %w", err)
	}
	defer rows.Close()
	count := 0
	for rows.Next() {
		count++
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate foreign key check: %w", err)
	}
	return count, nil
}

// migrateV5Statements is the frozen v4→v5 upgrade. The CREATE TABLE below is the
// version-5 shape of runtime_instances pinned at migration time; it must match
// the runtime_instances definition in schemaSQL for a fresh v5 install. The
// copy step defaults every existing row to variant 'live' and stamps its
// per-row schema_version to 5.
var migrateV5Statements = []string{
	`CREATE TABLE runtime_instances_v5 (
  instance_id TEXT PRIMARY KEY,
  scenario TEXT NOT NULL,
  variant TEXT NOT NULL DEFAULT 'live',
  generation INTEGER NOT NULL,
  scope_path TEXT NOT NULL DEFAULT '',
  sandbox_id TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  phase TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_heartbeat_at TEXT,
  heartbeat_deadline_at TEXT,
  stopped_at TEXT,
  stop_reason TEXT NOT NULL DEFAULT '',
  owner_kind TEXT NOT NULL DEFAULT 'lifecycle',
  owner_pid INTEGER,
  working_dir TEXT NOT NULL DEFAULT '',
  host_boot_id TEXT NOT NULL DEFAULT '',
  host_session_id TEXT NOT NULL DEFAULT '',
  supervisor_id TEXT NOT NULL DEFAULT '',
  supervised_at TEXT,
  last_reconciled_at TEXT,
  reconciliation_status TEXT NOT NULL DEFAULT '',
  reconciliation_reason TEXT NOT NULL DEFAULT '',
  supervision_policy TEXT NOT NULL DEFAULT 'managed',
  schema_version INTEGER NOT NULL,
  UNIQUE(scenario, variant, generation)
)`,
	`INSERT INTO runtime_instances_v5 (
  instance_id, scenario, variant, generation, scope_path, sandbox_id, status, phase,
  started_at, updated_at, last_heartbeat_at, heartbeat_deadline_at, stopped_at,
  stop_reason, owner_kind, owner_pid, working_dir, host_boot_id, host_session_id,
  supervisor_id, supervised_at, last_reconciled_at, reconciliation_status,
  reconciliation_reason, supervision_policy, schema_version
)
SELECT
  instance_id, scenario, 'live', generation, scope_path, sandbox_id, status, phase,
  started_at, updated_at, last_heartbeat_at, heartbeat_deadline_at, stopped_at,
  stop_reason, owner_kind, owner_pid, working_dir, host_boot_id, host_session_id,
  supervisor_id, supervised_at, last_reconciled_at, reconciliation_status,
  reconciliation_reason, supervision_policy, 5
FROM runtime_instances`,
	`DROP TABLE runtime_instances`,
	`ALTER TABLE runtime_instances_v5 RENAME TO runtime_instances`,
	`CREATE INDEX IF NOT EXISTS idx_runtime_instances_scenario ON runtime_instances(scenario)`,
	`CREATE INDEX IF NOT EXISTS idx_runtime_instances_scenario_variant ON runtime_instances(scenario, variant)`,
	`CREATE INDEX IF NOT EXISTS idx_runtime_instances_status ON runtime_instances(status)`,
	`CREATE INDEX IF NOT EXISTS idx_runtime_instances_heartbeat_deadline ON runtime_instances(heartbeat_deadline_at)`,
	`CREATE INDEX IF NOT EXISTS idx_runtime_instances_host_boot ON runtime_instances(host_boot_id)`,
	`CREATE INDEX IF NOT EXISTS idx_runtime_instances_supervisor ON runtime_instances(supervisor_id)`,
	`CREATE INDEX IF NOT EXISTS idx_runtime_instances_reconcile ON runtime_instances(reconciliation_status, last_reconciled_at)`,
	`ALTER TABLE runtime_port_claims ADD COLUMN variant TEXT NOT NULL DEFAULT 'live'`,
	`CREATE INDEX IF NOT EXISTS idx_runtime_port_claims_scenario_variant ON runtime_port_claims(scenario, variant)`,
}
