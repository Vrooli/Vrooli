-- Restores — owned by internal/restores/. Records restore and verify
-- operations. Times are RFC3339Nano strings. CREATE ... IF NOT EXISTS so
-- re-runs are no-ops.
CREATE TABLE IF NOT EXISTS restores (
  id               TEXT PRIMARY KEY,
  target_id        TEXT NOT NULL,
  destination_id   TEXT NOT NULL,
  snapshot_id      TEXT NOT NULL,
  mode             TEXT NOT NULL,
  status           TEXT NOT NULL,
  location         TEXT NOT NULL DEFAULT '',
  checksum         TEXT NOT NULL DEFAULT '',
  last_verified_at TEXT NOT NULL DEFAULT '',
  requested_at     TEXT NOT NULL,
  finished_at      TEXT NOT NULL DEFAULT '',
  error            TEXT NOT NULL DEFAULT '',
  -- heartbeat: last time the record's status was persisted, so an in-flight or
  -- wedged restore is observable and startup reconciliation can age it
  updated_at       TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_restores_target ON restores(target_id, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_restores_requested ON restores(requested_at DESC);
