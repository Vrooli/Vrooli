-- Audits — owned by internal/audits/. Records generic snapshot audit runs and
-- their evidence. The live/snapshot inventories and the comparison are stored
-- as JSON blobs (generic counts, hashes, and relative paths only — never file
-- contents or secrets). Times are RFC3339Nano strings. CREATE ... IF NOT EXISTS
-- so re-runs are no-ops.
CREATE TABLE IF NOT EXISTS audits (
  id                   TEXT PRIMARY KEY,
  target_id            TEXT NOT NULL,
  destination_id       TEXT NOT NULL,
  snapshot_id          TEXT NOT NULL,
  status               TEXT NOT NULL,
  include_content_hash INTEGER NOT NULL DEFAULT 1,
  include_sqlite_check INTEGER NOT NULL DEFAULT 1,
  restorable           INTEGER NOT NULL DEFAULT 0,
  live_json            TEXT NOT NULL DEFAULT '',
  snapshot_json        TEXT NOT NULL DEFAULT '',
  comparison_json      TEXT NOT NULL DEFAULT '',
  snapshot_time        TEXT NOT NULL DEFAULT '',
  requested_at         TEXT NOT NULL,
  finished_at          TEXT NOT NULL DEFAULT '',
  error                TEXT NOT NULL DEFAULT '',
  -- heartbeat: last time the record's status was persisted, so an in-flight or
  -- wedged audit is observable and startup reconciliation can age it
  updated_at           TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_audits_target ON audits(target_id, requested_at DESC);
CREATE INDEX IF NOT EXISTS idx_audits_requested ON audits(requested_at DESC);
