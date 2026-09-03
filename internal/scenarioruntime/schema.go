package scenarioruntime

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS runtime_instances (
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
);
CREATE INDEX IF NOT EXISTS idx_runtime_instances_scenario ON runtime_instances(scenario);
CREATE INDEX IF NOT EXISTS idx_runtime_instances_scenario_variant ON runtime_instances(scenario, variant);
CREATE INDEX IF NOT EXISTS idx_runtime_instances_status ON runtime_instances(status);
CREATE INDEX IF NOT EXISTS idx_runtime_instances_heartbeat_deadline ON runtime_instances(heartbeat_deadline_at);
CREATE INDEX IF NOT EXISTS idx_runtime_instances_host_boot ON runtime_instances(host_boot_id);
CREATE INDEX IF NOT EXISTS idx_runtime_instances_supervisor ON runtime_instances(supervisor_id);
CREATE INDEX IF NOT EXISTS idx_runtime_instances_reconcile ON runtime_instances(reconciliation_status, last_reconciled_at);

CREATE TABLE IF NOT EXISTS runtime_supervisor_sessions (
  supervisor_id TEXT PRIMARY KEY,
  host_boot_id TEXT NOT NULL,
  host_session_id TEXT NOT NULL,
  pid INTEGER,
  status TEXT NOT NULL,
  started_at TEXT NOT NULL,
  last_heartbeat_at TEXT NOT NULL,
  heartbeat_deadline_at TEXT NOT NULL,
  stopped_at TEXT,
  stop_reason TEXT NOT NULL DEFAULT '',
  version TEXT NOT NULL DEFAULT '',
  metadata_json TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_runtime_supervisor_sessions_status ON runtime_supervisor_sessions(status);
CREATE INDEX IF NOT EXISTS idx_runtime_supervisor_sessions_deadline ON runtime_supervisor_sessions(heartbeat_deadline_at);

CREATE TABLE IF NOT EXISTS runtime_port_claims (
  claim_id TEXT PRIMARY KEY,
  instance_id TEXT NOT NULL,
  scenario TEXT NOT NULL,
  variant TEXT NOT NULL DEFAULT 'live',
  port_name TEXT NOT NULL DEFAULT '',
  env_var TEXT NOT NULL DEFAULT '',
  port INTEGER NOT NULL,
  bind_host TEXT NOT NULL DEFAULT '127.0.0.1',
  url TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  expires_at TEXT,
  last_bound_at TEXT,
  last_listener_check_at TEXT,
  last_listener_seen_at TEXT,
  first_unbound_at TEXT,
  consecutive_listener_misses INTEGER NOT NULL DEFAULT 0,
  listener_status TEXT NOT NULL DEFAULT 'unknown',
  listener_pid INTEGER,
  listener_process_label TEXT NOT NULL DEFAULT '',
  FOREIGN KEY(instance_id) REFERENCES runtime_instances(instance_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_runtime_port_claims_instance ON runtime_port_claims(instance_id);
CREATE INDEX IF NOT EXISTS idx_runtime_port_claims_scenario ON runtime_port_claims(scenario);
CREATE INDEX IF NOT EXISTS idx_runtime_port_claims_scenario_variant ON runtime_port_claims(scenario, variant);
CREATE INDEX IF NOT EXISTS idx_runtime_port_claims_status ON runtime_port_claims(status);
CREATE INDEX IF NOT EXISTS idx_runtime_port_claims_expiry ON runtime_port_claims(expires_at);
CREATE UNIQUE INDEX IF NOT EXISTS idx_runtime_port_claims_active_port
  ON runtime_port_claims(port, bind_host)
  WHERE status IN ('reserved', 'bound');

CREATE TABLE IF NOT EXISTS runtime_health_snapshots (
  instance_id TEXT PRIMARY KEY,
  scenario TEXT NOT NULL,
  status TEXT NOT NULL,
  readiness INTEGER,
  checked_at TEXT,
  latency_ms INTEGER,
  error TEXT NOT NULL DEFAULT '',
  response_json TEXT NOT NULL DEFAULT '',
  schema_valid INTEGER,
  FOREIGN KEY(instance_id) REFERENCES runtime_instances(instance_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_runtime_health_snapshots_scenario ON runtime_health_snapshots(scenario);

CREATE TABLE IF NOT EXISTS runtime_process_refs (
  ref_id TEXT PRIMARY KEY,
  instance_id TEXT NOT NULL,
  pid INTEGER,
  pgid INTEGER,
  process_id TEXT NOT NULL DEFAULT '',
  step TEXT NOT NULL DEFAULT '',
  command TEXT NOT NULL DEFAULT '',
  log_file TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL,
  ended_at TEXT,
  host_boot_id TEXT NOT NULL DEFAULT '',
  FOREIGN KEY(instance_id) REFERENCES runtime_instances(instance_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_runtime_process_refs_instance ON runtime_process_refs(instance_id);
CREATE INDEX IF NOT EXISTS idx_runtime_process_refs_pid ON runtime_process_refs(pid);

CREATE TABLE IF NOT EXISTS runtime_events (
  event_id TEXT PRIMARY KEY,
  instance_id TEXT,
  scenario TEXT NOT NULL DEFAULT '',
  event_type TEXT NOT NULL,
  created_at TEXT NOT NULL,
  details_json TEXT NOT NULL DEFAULT '',
  FOREIGN KEY(instance_id) REFERENCES runtime_instances(instance_id) ON DELETE SET NULL
);
CREATE INDEX IF NOT EXISTS idx_runtime_events_instance ON runtime_events(instance_id);
CREATE INDEX IF NOT EXISTS idx_runtime_events_scenario ON runtime_events(scenario);
CREATE INDEX IF NOT EXISTS idx_runtime_events_type ON runtime_events(event_type);

CREATE TABLE IF NOT EXISTS runtime_start_operations (
  operation_id TEXT PRIMARY KEY,
  scenario TEXT NOT NULL,
  variant TEXT NOT NULL DEFAULT 'live',
  operation TEXT NOT NULL DEFAULT 'start',
  status TEXT NOT NULL,
  verdict TEXT NOT NULL DEFAULT '',
  error TEXT NOT NULL DEFAULT '',
  initiator_pid INTEGER,
  initiator_argv TEXT NOT NULL DEFAULT '',
  initiator_parent_pid INTEGER,
  initiator_parent_argv TEXT NOT NULL DEFAULT '',
  initiator_scope TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  finished_at TEXT,
  current_step TEXT NOT NULL DEFAULT '',
  dependency_current TEXT NOT NULL DEFAULT '',
  dependency_index INTEGER NOT NULL DEFAULT 0,
  dependency_total INTEGER NOT NULL DEFAULT 0,
  steps_json TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_runtime_start_operations_scenario_variant
  ON runtime_start_operations(scenario, variant);
CREATE INDEX IF NOT EXISTS idx_runtime_start_operations_status
  ON runtime_start_operations(status);

CREATE TABLE IF NOT EXISTS runtime_phase_durations (
  scenario TEXT NOT NULL,
  variant TEXT NOT NULL DEFAULT 'live',
  phase TEXT NOT NULL,
  duration_ms INTEGER NOT NULL,
  recorded_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_runtime_phase_durations_key
  ON runtime_phase_durations(scenario, variant, phase);

CREATE TABLE IF NOT EXISTS runtime_recovery_policies (
  scenario TEXT NOT NULL,
  variant TEXT NOT NULL DEFAULT 'live',
  critical INTEGER NOT NULL DEFAULT 0,
  dependency_tier INTEGER NOT NULL DEFAULT 0,
  enabled INTEGER NOT NULL DEFAULT 0,
  retry_budget INTEGER NOT NULL DEFAULT 0,
  opt_out INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (scenario, variant)
);
CREATE INDEX IF NOT EXISTS idx_runtime_recovery_policies_enabled
  ON runtime_recovery_policies(enabled, critical, opt_out, dependency_tier);

CREATE TABLE IF NOT EXISTS runtime_pressure_epochs (
  epoch_id TEXT PRIMARY KEY,
  status TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT '',
  detected_at TEXT NOT NULL,
  cleared_at TEXT,
  updated_at TEXT NOT NULL,
  details_json TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_runtime_pressure_epochs_status
  ON runtime_pressure_epochs(status, detected_at DESC);

CREATE TABLE IF NOT EXISTS runtime_recovery_decisions (
  decision_id TEXT PRIMARY KEY,
  epoch_id TEXT NOT NULL,
  scenario TEXT NOT NULL,
  variant TEXT NOT NULL DEFAULT 'live',
  state TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  attempt INTEGER NOT NULL DEFAULT 0,
  cooldown_until TEXT,
  idempotency_key TEXT NOT NULL,
  created_at TEXT NOT NULL,
  details_json TEXT NOT NULL DEFAULT '',
  FOREIGN KEY(epoch_id) REFERENCES runtime_pressure_epochs(epoch_id) ON DELETE CASCADE,
  UNIQUE(idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_runtime_recovery_decisions_epoch
  ON runtime_recovery_decisions(epoch_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_runtime_recovery_decisions_instance
  ON runtime_recovery_decisions(scenario, variant, created_at DESC);
` + editorLeaseSchemaSQL

func (s *SQLiteStore) ensureSchema(ctx context.Context) error {
	current, err := readSchemaVersion(ctx, s.db)
	if err != nil {
		return fmt.Errorf("read runtime registry schema version: %w", err)
	}
	if current > SchemaVersion {
		return &SchemaCompatibilityError{DatabaseVersion: current, BinaryVersion: SchemaVersion}
	}
	if current == SchemaVersion {
		return nil
	}
	// Additive migration ladder. Each rung applies ONLY its own step and stamps
	// ONLY its own version, so a database two versions behind climbs both rungs
	// instead of being stamped current after one of them — which is what the
	// previous single-step form would have done the moment a second rung
	// existed.
	if current > 0 {
		existing, err := runtimeSchemaExists(ctx, s.db)
		if err != nil {
			return fmt.Errorf("inspect runtime registry before migration: %w", err)
		}
		if !existing {
			return fmt.Errorf("runtime registry schema_version %d is unstamped or incomplete: requires greenfield rebuild or an operator-run temporary conversion script", current)
		}
		for current < SchemaVersion {
			step, ok := schemaMigrations[current]
			if !ok {
				return fmt.Errorf("runtime registry schema_version %d -> %d requires greenfield rebuild or an operator-run temporary conversion script", current, SchemaVersion)
			}
			if err := step(ctx, s.db); err != nil {
				return fmt.Errorf("migrate runtime registry schema %d -> %d: %w", current, current+1, err)
			}
			current++
			if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d`, current)); err != nil {
				return fmt.Errorf("stamp migrated runtime registry schema version %d: %w", current, err)
			}
		}
		return nil
	}
	if current != 0 {
		return fmt.Errorf("runtime registry schema_version %d -> %d requires greenfield rebuild or an operator-run temporary conversion script", current, SchemaVersion)
	}
	existing, err := runtimeSchemaExists(ctx, s.db)
	if err != nil {
		return fmt.Errorf("inspect runtime registry schema: %w", err)
	}
	if existing {
		return fmt.Errorf("runtime registry schema is unstamped or incompatible with schema_version %d: requires greenfield rebuild or an operator-run temporary conversion script", SchemaVersion)
	}
	if _, err := s.db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply runtime registry schema: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d`, SchemaVersion)); err != nil {
		return fmt.Errorf("stamp runtime registry schema version: %w", err)
	}
	return nil
}

// schemaMigrations maps a database's current version to the step that carries
// it to the next one. Steps must be additive and idempotent: they run against
// live registries that other processes may be reading.
var schemaMigrations = map[int]func(context.Context, *sql.DB) error{
	6: func(ctx context.Context, db *sql.DB) error {
		_, err := db.ExecContext(ctx, recoverySchemaSQL)
		return err
	},
	7: addStartOperationProvenance,
	8: func(ctx context.Context, db *sql.DB) error {
		_, err := db.ExecContext(ctx, editorLeaseSchemaSQL)
		return err
	},
}

// addStartOperationProvenance records WHO initiated a start, not merely its
// PID. Columns are added one at a time because SQLite has no "ADD COLUMN IF
// NOT EXISTS"; a column that is already present is treated as done rather than
// as a failure, so re-running the step is safe.
func addStartOperationProvenance(ctx context.Context, db *sql.DB) error {
	columns := []string{
		`ALTER TABLE runtime_start_operations ADD COLUMN initiator_argv TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE runtime_start_operations ADD COLUMN initiator_parent_pid INTEGER`,
		`ALTER TABLE runtime_start_operations ADD COLUMN initiator_parent_argv TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE runtime_start_operations ADD COLUMN initiator_scope TEXT NOT NULL DEFAULT ''`,
	}
	for _, statement := range columns {
		if _, err := db.ExecContext(ctx, statement); err != nil {
			if strings.Contains(strings.ToLower(err.Error()), "duplicate column name") {
				continue
			}
			return err
		}
	}
	return nil
}

// editorLeaseSchemaSQL is the live agent session projection (schema 9).
const editorLeaseSchemaSQL = `
CREATE TABLE IF NOT EXISTS runtime_editor_leases (
  session_id TEXT PRIMARY KEY,
  harness TEXT NOT NULL DEFAULT '',
  agent TEXT NOT NULL DEFAULT '',
  pid INTEGER,
  host_boot_id TEXT NOT NULL DEFAULT '',
  working_dir TEXT NOT NULL DEFAULT '',
  scope TEXT NOT NULL DEFAULT '',
  containment_method TEXT NOT NULL DEFAULT '',
  claims_json TEXT NOT NULL DEFAULT '[]',
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  last_heartbeat_at TEXT NOT NULL,
  heartbeat_deadline_at TEXT NOT NULL,
  stopped_at TEXT,
  stop_reason TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_runtime_editor_leases_status ON runtime_editor_leases(status);
CREATE INDEX IF NOT EXISTS idx_runtime_editor_leases_working_dir ON runtime_editor_leases(working_dir);
`

const recoverySchemaSQL = `
CREATE TABLE IF NOT EXISTS runtime_recovery_policies (
  scenario TEXT NOT NULL,
  variant TEXT NOT NULL DEFAULT 'live',
  critical INTEGER NOT NULL DEFAULT 0,
  dependency_tier INTEGER NOT NULL DEFAULT 0,
  enabled INTEGER NOT NULL DEFAULT 0,
  retry_budget INTEGER NOT NULL DEFAULT 0,
  opt_out INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (scenario, variant)
);
CREATE INDEX IF NOT EXISTS idx_runtime_recovery_policies_enabled
  ON runtime_recovery_policies(enabled, critical, opt_out, dependency_tier);
CREATE TABLE IF NOT EXISTS runtime_pressure_epochs (
  epoch_id TEXT PRIMARY KEY,
  status TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT '',
  detected_at TEXT NOT NULL,
  cleared_at TEXT,
  updated_at TEXT NOT NULL,
  details_json TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_runtime_pressure_epochs_status
  ON runtime_pressure_epochs(status, detected_at DESC);
CREATE TABLE IF NOT EXISTS runtime_recovery_decisions (
  decision_id TEXT PRIMARY KEY,
  epoch_id TEXT NOT NULL,
  scenario TEXT NOT NULL,
  variant TEXT NOT NULL DEFAULT 'live',
  state TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  attempt INTEGER NOT NULL DEFAULT 0,
  cooldown_until TEXT,
  idempotency_key TEXT NOT NULL,
  created_at TEXT NOT NULL,
  details_json TEXT NOT NULL DEFAULT '',
  FOREIGN KEY(epoch_id) REFERENCES runtime_pressure_epochs(epoch_id) ON DELETE CASCADE,
  UNIQUE(idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_runtime_recovery_decisions_epoch
  ON runtime_recovery_decisions(epoch_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_runtime_recovery_decisions_instance
  ON runtime_recovery_decisions(scenario, variant, created_at DESC);
`

func readSchemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	var v int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&v); err != nil {
		return 0, err
	}
	return v, nil
}

func runtimeSchemaExists(ctx context.Context, db *sql.DB) (bool, error) {
	rows, err := db.QueryContext(ctx, `
SELECT name
FROM sqlite_master
WHERE type = 'table'
  AND name IN ('schema_version', 'runtime_instances', 'runtime_port_claims', 'runtime_supervisor_sessions')`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	return rows.Next(), rows.Err()
}
