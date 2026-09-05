-- Fleet rollouts + per-node results — owned by internal/fleet/. Embedded by
-- schema.go and applied via database.EnsureSchemas at boot (api/main.go through
-- modules.AllSchemas). Fleet-wide version rolls (OT-P1-001): one durable Rollout
-- record per RollFleet and its per-node ledger. Times are RFC3339Nano strings
-- matching the wire format and the time.Time round-trip in sqlite.go. Use CREATE
-- TABLE IF NOT EXISTS so re-runs are no-ops (migrate, never recreate). Postgres-
-- compatible column types for forward scale.
CREATE TABLE IF NOT EXISTS rollouts (
  id              TEXT PRIMARY KEY,
  target_revision TEXT NOT NULL DEFAULT '',
  status          INTEGER NOT NULL DEFAULT 0,
  total_nodes     INTEGER NOT NULL DEFAULT 0,
  dispatched      INTEGER NOT NULL DEFAULT 0,
  skipped         INTEGER NOT NULL DEFAULT 0,
  failed          INTEGER NOT NULL DEFAULT 0,
  created_at      TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_rollouts_created_at ON rollouts(created_at DESC);

-- rollout_results is the per-node ledger, written once with the rollout and
-- never updated. The (rollout_id, node_id) pair is unique (one line per node).
CREATE TABLE IF NOT EXISTS rollout_results (
  rollout_id  TEXT NOT NULL,
  node_id     TEXT NOT NULL,
  disposition INTEGER NOT NULL DEFAULT 0,
  op_id       TEXT NOT NULL DEFAULT '',
  detail      TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (rollout_id, node_id)
);

CREATE INDEX IF NOT EXISTS idx_rollout_results_rollout ON rollout_results(rollout_id, node_id);
