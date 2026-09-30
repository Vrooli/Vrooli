-- coverage_snapshots — owned by internal/coverage/. The short-TTL cache of the
-- computed readiness scoreboard (the numerator is NEVER stored as truth; this
-- only absorbs polling bursts). Embedded by schema.go and applied via
-- database.EnsureSchemas at boot through the modules.AllSchemas registry.
-- computed_at is RFC3339Nano matching internal/coverage/snapshot_sqlite.go.
-- Use CREATE TABLE IF NOT EXISTS so re-runs are no-ops.
CREATE TABLE IF NOT EXISTS coverage_snapshots (
  id          INTEGER PRIMARY KEY AUTOINCREMENT,
  computed_at TEXT NOT NULL,
  payload     TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_coverage_snapshots_computed_at
  ON coverage_snapshots(computed_at DESC);
