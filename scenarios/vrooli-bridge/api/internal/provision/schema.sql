-- Provisioning ops + events + node versions — owned by internal/provision/.
-- Embedded by schema.go and applied via database.EnsureSchemas at boot
-- (api/main.go through modules.AllSchemas). The PRIVILEGED tier (OT-P0-006):
-- durable, server-owned provisioning ops and their append-only event history,
-- plus the per-node version history. Times are RFC3339Nano strings matching the
-- wire format and the time.Time round-trip in sqlite.go. Use CREATE TABLE IF
-- NOT EXISTS so re-runs are no-ops (migrate, never recreate). Postgres-
-- compatible column types for forward scale.
CREATE TABLE IF NOT EXISTS provisioning_ops (
  id                 TEXT PRIMARY KEY,
  node_id            TEXT NOT NULL,
  target_revision    TEXT NOT NULL DEFAULT '',
  rollback_revision  TEXT NOT NULL DEFAULT '',
  status             INTEGER NOT NULL DEFAULT 0,
  resulting_revision TEXT NOT NULL DEFAULT '',
  exit_code          INTEGER NOT NULL DEFAULT 0,
  timeout_seconds    INTEGER NOT NULL DEFAULT 0,
  created_at         TEXT NOT NULL,
  started_at         TEXT NOT NULL DEFAULT '',
  finished_at        TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_provisioning_ops_created_at ON provisioning_ops(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_provisioning_ops_node_id ON provisioning_ops(node_id);

-- provision_events is append-only: rows are only ever INSERTed and SELECTed
-- (there is no UPDATE/DELETE in sqlite.go). The (op_id, sequence) pair is unique
-- so a node re-sending an event (at-least-once delivery) is de-duplicated rather
-- than double-stored.
CREATE TABLE IF NOT EXISTS provision_events (
  op_id        TEXT NOT NULL,
  sequence     INTEGER NOT NULL,
  kind         INTEGER NOT NULL DEFAULT 0,
  log_chunk    TEXT NOT NULL DEFAULT '',
  status       TEXT NOT NULL DEFAULT '',
  revision     TEXT NOT NULL DEFAULT '',
  exit_code    INTEGER NOT NULL DEFAULT 0,
  emitted_at   TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (op_id, sequence)
);

CREATE INDEX IF NOT EXISTS idx_provision_events_op ON provision_events(op_id, sequence);

-- node_versions keeps the latest known project revision per node (one row per
-- node — UPSERT on node_id replaces it). The op_id records which provisioning
-- op last set it.
CREATE TABLE IF NOT EXISTS node_versions (
  node_id     TEXT PRIMARY KEY,
  revision    TEXT NOT NULL DEFAULT '',
  op_id       TEXT NOT NULL DEFAULT '',
  reported_at TEXT NOT NULL DEFAULT ''
);
