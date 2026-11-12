package capacity

import (
	"context"
	"database/sql"
	"fmt"
)

const schemaSQL = `
CREATE TABLE IF NOT EXISTS capacity_claims (
  claim_id TEXT PRIMARY KEY,
  owner_kind TEXT NOT NULL,
  owner_id TEXT NOT NULL,
  instance_id TEXT NOT NULL DEFAULT '',
  resource_kind TEXT NOT NULL,
  gpu_index INTEGER,
  amount_bytes INTEGER NOT NULL DEFAULT 0,
  preferred_bytes INTEGER NOT NULL DEFAULT 0,
  floor_bytes INTEGER NOT NULL DEFAULT 0,
  priority INTEGER NOT NULL DEFAULT 10,
  protected INTEGER NOT NULL DEFAULT 0,
  yield_when_idle INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL,
  activity_state TEXT NOT NULL DEFAULT 'idle',
  generation INTEGER NOT NULL DEFAULT 1,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_heartbeat_at TEXT,
  heartbeat_deadline_at TEXT,
  last_active_at TEXT,
  degrade_profile TEXT NOT NULL DEFAULT '',
  observed_bytes INTEGER NOT NULL DEFAULT 0,
  observed_peak_bytes INTEGER NOT NULL DEFAULT 0,
  observed_at TEXT,
  idle_unload_ttl_seconds INTEGER NOT NULL DEFAULT 0,
  idle_grace_seconds INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_capacity_claims_owner ON capacity_claims(owner_kind, owner_id);
CREATE INDEX IF NOT EXISTS idx_capacity_claims_status ON capacity_claims(status);
CREATE INDEX IF NOT EXISTS idx_capacity_claims_resource ON capacity_claims(resource_kind, gpu_index);
CREATE INDEX IF NOT EXISTS idx_capacity_claims_heartbeat_deadline ON capacity_claims(heartbeat_deadline_at);
CREATE INDEX IF NOT EXISTS idx_capacity_claims_activity ON capacity_claims(activity_state);

CREATE TABLE IF NOT EXISTS capacity_policy (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS capacity_footprints (
  resource TEXT NOT NULL,
  rung TEXT NOT NULL,
  tunables_key TEXT NOT NULL,
  gpu_index INTEGER NOT NULL DEFAULT 0,
  peak_bytes INTEGER NOT NULL,
  samples INTEGER NOT NULL DEFAULT 1,
  first_seen_at TEXT NOT NULL,
  last_seen_at TEXT NOT NULL,
  source TEXT NOT NULL CHECK (source IN ('manifest_default', 'measured')),
  PRIMARY KEY (resource, rung, tunables_key, gpu_index)
);
CREATE INDEX IF NOT EXISTS idx_capacity_footprints_resource ON capacity_footprints(resource);

-- Disk reservations are tracked by filesystem identity rather than path. The
-- table lives in the host capacity ledger so image, video and media clients
-- share one admission decision across processes.
CREATE TABLE IF NOT EXISTS capacity_disk_reservations (
  reservation_id TEXT PRIMARY KEY,
  owner_id TEXT NOT NULL,
  filesystem_id TEXT NOT NULL,
  path TEXT NOT NULL,
  peak_bytes INTEGER NOT NULL CHECK (peak_bytes > 0),
  floor_bytes INTEGER NOT NULL CHECK (floor_bytes >= 0),
  status TEXT NOT NULL CHECK (status IN ('active', 'released', 'expired')),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  heartbeat_deadline_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_capacity_disk_reservations_filesystem ON capacity_disk_reservations(filesystem_id, status, heartbeat_deadline_at);

-- A disk job is one atomic reservation across all filesystems its producer
-- can write to. Stale heartbeats are observable state, never a release signal.
CREATE TABLE IF NOT EXISTS capacity_disk_jobs (
  job_lease_id TEXT PRIMARY KEY,
  owner_id TEXT NOT NULL,
  job_id TEXT NOT NULL,
  process_id INTEGER NOT NULL,
  process_start TEXT NOT NULL,
  capability_hash TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('active','completed','reconciled')),
  floor_bytes INTEGER NOT NULL CHECK (floor_bytes >= 0),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  last_heartbeat_at TEXT NOT NULL,
  heartbeat_deadline_at TEXT NOT NULL,
  cleanup_verified INTEGER NOT NULL DEFAULT 0,
  UNIQUE(owner_id, job_id)
);
CREATE INDEX IF NOT EXISTS idx_capacity_disk_jobs_status ON capacity_disk_jobs(status, heartbeat_deadline_at);

CREATE TABLE IF NOT EXISTS capacity_disk_job_filesystems (
  job_lease_id TEXT NOT NULL REFERENCES capacity_disk_jobs(job_lease_id) ON DELETE CASCADE,
  filesystem_id TEXT NOT NULL,
  filesystem_path TEXT NOT NULL,
  peak_bytes INTEGER NOT NULL CHECK (peak_bytes > 0),
  baseline_free_bytes INTEGER NOT NULL CHECK (baseline_free_bytes >= 0),
  available_at_admission_bytes INTEGER NOT NULL CHECK (available_at_admission_bytes >= 0),
  other_reserved_bytes INTEGER NOT NULL CHECK (other_reserved_bytes >= 0),
  actual_delta_bytes INTEGER,
  PRIMARY KEY(job_lease_id, filesystem_id)
);
CREATE INDEX IF NOT EXISTS idx_capacity_disk_job_filesystems_fs ON capacity_disk_job_filesystems(filesystem_id, job_lease_id);

CREATE TABLE IF NOT EXISTS capacity_disk_job_paths (
  job_lease_id TEXT NOT NULL REFERENCES capacity_disk_jobs(job_lease_id) ON DELETE CASCADE,
  filesystem_id TEXT NOT NULL,
  canonical_path TEXT NOT NULL,
  role TEXT NOT NULL,
  peak_bytes INTEGER NOT NULL CHECK (peak_bytes > 0),
  PRIMARY KEY(job_lease_id, canonical_path)
);
CREATE INDEX IF NOT EXISTS idx_capacity_disk_job_paths_fs ON capacity_disk_job_paths(filesystem_id, job_lease_id);

-- Owners may report a byte-count checkpoint for an exact admitted path. A
-- history row is immutable evidence; sample order and monotonicity are checked
-- under the admission lock before insertion.
CREATE TABLE IF NOT EXISTS capacity_disk_job_path_samples (
  sample_id INTEGER PRIMARY KEY AUTOINCREMENT,
  job_lease_id TEXT NOT NULL REFERENCES capacity_disk_jobs(job_lease_id) ON DELETE CASCADE,
  canonical_path TEXT NOT NULL,
  filesystem_id TEXT NOT NULL,
  bytes INTEGER NOT NULL CHECK (bytes >= 0),
  baseline_bytes INTEGER NOT NULL CHECK (baseline_bytes >= 0),
  delta_bytes INTEGER NOT NULL CHECK (delta_bytes >= 0),
  reserved_peak_bytes INTEGER NOT NULL CHECK (reserved_peak_bytes > 0),
  exceeded_reserved_peak INTEGER NOT NULL CHECK (exceeded_reserved_peak IN (0,1)),
  observed_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_capacity_disk_job_path_samples_latest ON capacity_disk_job_path_samples(job_lease_id, canonical_path, sample_id);

-- Residency transitions are immutable-identity, revisioned owner reports.
-- A single incomplete transition per claim prevents two unload attempts from
-- independently claiming the same physical capacity. The JSON payload retains
-- unknown/partial evidence without turning it into a guessed backend or size.
CREATE TABLE IF NOT EXISTS capacity_residency_transitions (
  transition_id TEXT PRIMARY KEY,
  claim_id TEXT NOT NULL REFERENCES capacity_claims(claim_id),
  claim_generation INTEGER NOT NULL CHECK (claim_generation > 0),
  revision INTEGER NOT NULL CHECK (revision > 0),
  state TEXT NOT NULL CHECK (state IN ('started', 'completed')),
  transition_json TEXT NOT NULL,
  started_at TEXT NOT NULL,
  completed_at TEXT
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_capacity_residency_one_open_per_claim
  ON capacity_residency_transitions(claim_id) WHERE state = 'started';
CREATE INDEX IF NOT EXISTS idx_capacity_residency_claim_history
  ON capacity_residency_transitions(claim_id, started_at DESC);

-- Every state change is also retained as an append-only owner audit event. The
-- transition row is the current replay view; events preserve each CAS result,
-- including unknown recovery observations and the final claim release.
CREATE TABLE IF NOT EXISTS capacity_residency_transition_events (
  event_id INTEGER PRIMARY KEY AUTOINCREMENT,
  transition_id TEXT NOT NULL REFERENCES capacity_residency_transitions(transition_id),
  claim_id TEXT NOT NULL REFERENCES capacity_claims(claim_id),
  claim_generation INTEGER NOT NULL CHECK (claim_generation > 0),
  revision INTEGER NOT NULL CHECK (revision > 0),
  event_kind TEXT NOT NULL CHECK (event_kind IN ('started','completed','reconciled','recovery_unknown','recovery_terminal_observed','recovered','claim_released')),
  transition_json TEXT NOT NULL,
  recorded_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_capacity_residency_events_transition
  ON capacity_residency_transition_events(transition_id, event_id);
`

func (s *SQLiteStore) ensureSchema(ctx context.Context) error {
	current, err := readSchemaVersion(ctx, s.db)
	if err != nil {
		return fmt.Errorf("read capacity ledger schema version: %w", err)
	}
	if current > SchemaVersion {
		return fmt.Errorf("capacity ledger schema_version %d > expected %d: binary is older than database", current, SchemaVersion)
	}
	if current == SchemaVersion {
		// The declarative schema is idempotent. Reapply it so whole new tables and
		// indexes reach existing greenfield ledgers without a migration ladder.
		if _, err := s.db.ExecContext(ctx, schemaSQL); err != nil {
			return fmt.Errorf("ensure capacity ledger schema: %w", err)
		}
		return ensureDiskJobPathSampleColumns(ctx, s.db)
	}
	if current != 0 {
		return fmt.Errorf("capacity ledger schema_version %d -> %d requires greenfield rebuild or an operator-run temporary conversion script", current, SchemaVersion)
	}
	existing, err := capacitySchemaExists(ctx, s.db)
	if err != nil {
		return fmt.Errorf("inspect capacity ledger schema: %w", err)
	}
	if existing {
		return fmt.Errorf("capacity ledger schema is unstamped or incompatible with schema_version %d: requires greenfield rebuild or an operator-run temporary conversion script", SchemaVersion)
	}
	if _, err := s.db.ExecContext(ctx, schemaSQL); err != nil {
		return fmt.Errorf("apply capacity ledger schema: %w", err)
	}
	if err := ensureDiskJobPathSampleColumns(ctx, s.db); err != nil {
		return err
	}
	if _, err := s.db.ExecContext(ctx, fmt.Sprintf(`PRAGMA user_version = %d`, SchemaVersion)); err != nil {
		return fmt.Errorf("stamp capacity ledger schema version: %w", err)
	}
	return nil
}

// ensureDiskJobPathSampleColumns advances the table created by an earlier
// development build without rewriting or dropping sample history. Legacy rows
// receive a conservative one-byte marked-unknown reservation marker.
func ensureDiskJobPathSampleColumns(ctx context.Context, db *sql.DB) error {
	rows, err := db.QueryContext(ctx, `PRAGMA table_info(capacity_disk_job_path_samples)`)
	if err != nil {
		return fmt.Errorf("inspect disk path sample schema: %w", err)
	}
	columns := map[string]bool{}
	for rows.Next() {
		var cid, notnull, pk int
		var name, kind string
		var defaultValue any
		if err := rows.Scan(&cid, &name, &kind, &notnull, &defaultValue, &pk); err != nil {
			rows.Close()
			return err
		}
		columns[name] = true
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	rows.Close()
	if !columns["reserved_peak_bytes"] {
		if _, err := db.ExecContext(ctx, `ALTER TABLE capacity_disk_job_path_samples ADD COLUMN reserved_peak_bytes INTEGER NOT NULL DEFAULT 1 CHECK (reserved_peak_bytes > 0)`); err != nil {
			return fmt.Errorf("add disk path sample reserved peak: %w", err)
		}
	}
	if !columns["exceeded_reserved_peak"] {
		if _, err := db.ExecContext(ctx, `ALTER TABLE capacity_disk_job_path_samples ADD COLUMN exceeded_reserved_peak INTEGER NOT NULL DEFAULT 1 CHECK (exceeded_reserved_peak IN (0,1))`); err != nil {
			return fmt.Errorf("add disk path sample overrun marker: %w", err)
		}
	}
	return nil
}

func readSchemaVersion(ctx context.Context, db *sql.DB) (int, error) {
	var v int
	if err := db.QueryRowContext(ctx, `PRAGMA user_version`).Scan(&v); err != nil {
		return 0, err
	}
	return v, nil
}

func capacitySchemaExists(ctx context.Context, db *sql.DB) (bool, error) {
	rows, err := db.QueryContext(ctx, `
SELECT name
FROM sqlite_master
WHERE type = 'table'
  AND name IN ('capacity_claims', 'capacity_policy')`)
	if err != nil {
		return false, err
	}
	defer rows.Close()
	return rows.Next(), rows.Err()
}
