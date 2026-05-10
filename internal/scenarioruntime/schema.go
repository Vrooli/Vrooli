package scenarioruntime

import (
	"context"
	"database/sql"
	"fmt"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS schema_version (
  version INTEGER NOT NULL,
  applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS runtime_instances (
  instance_id TEXT PRIMARY KEY,
  scenario TEXT NOT NULL,
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
  UNIQUE(scenario, generation)
);
CREATE INDEX IF NOT EXISTS idx_runtime_instances_scenario ON runtime_instances(scenario);
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
  FOREIGN KEY(instance_id) REFERENCES runtime_instances(instance_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_runtime_port_claims_instance ON runtime_port_claims(instance_id);
CREATE INDEX IF NOT EXISTS idx_runtime_port_claims_scenario ON runtime_port_claims(scenario);
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
`

func (s *SQLiteStore) ensureSchema(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `
CREATE TABLE IF NOT EXISTS schema_version (
  version INTEGER NOT NULL,
  applied_at TEXT NOT NULL
)`); err != nil {
		return fmt.Errorf("prepare runtime registry schema version table: %w", err)
	}
	current, err := readSchemaVersion(ctx, s.db)
	if err != nil {
		return fmt.Errorf("read runtime registry schema version: %w", err)
	}
	if current > SchemaVersion {
		return fmt.Errorf("runtime registry schema_version %d > expected %d: binary is older than database", current, SchemaVersion)
	}
	if current == SchemaVersion {
		return nil
	}
	if current != 0 {
		return fmt.Errorf("runtime registry schema_version %d -> %d requires greenfield rebuild or an operator-run temporary conversion script", current, SchemaVersion)
	}
	if _, err := s.db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply runtime registry schema: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `DELETE FROM schema_version`); err != nil {
		return fmt.Errorf("clear runtime registry schema version: %w", err)
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO schema_version (version, applied_at) VALUES (?, ?)`, SchemaVersion, formatTime(s.now())); err != nil {
		return fmt.Errorf("stamp runtime registry schema version: %w", err)
	}
	return nil
}

func readSchemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	var v sql.NullInt64
	if err := db.QueryRowContext(ctx, `SELECT MAX(version) FROM schema_version`).Scan(&v); err != nil {
		return 0, err
	}
	if !v.Valid {
		return 0, nil
	}
	return int(v.Int64), nil
}
