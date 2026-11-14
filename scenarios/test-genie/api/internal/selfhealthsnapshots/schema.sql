-- selfhealth_snapshots persists timestamped rollups of Test Genie's own
-- reliability + conformance so trend deltas ("is our test infrastructure
-- getting better or worse?") are answerable over time. It is written ONLY by
-- the background self-health sweeper (single writer); every read path is
-- compute-on-read and never writes here.
--
-- digest dedups identical snapshots: the sweeper computes a content digest over
-- the rollup payload (excluding the capture timestamp) and INSERT OR IGNOREs,
-- so an idle period with no change does not accumulate rows.
CREATE TABLE IF NOT EXISTS selfhealth_snapshots (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  captured_at TEXT NOT NULL,
  window_days INTEGER NOT NULL DEFAULT 0,
  run_count INTEGER NOT NULL DEFAULT 0,
  availability REAL NOT NULL DEFAULT 0,
  hard_violations INTEGER NOT NULL DEFAULT 0,
  metrics_adopted INTEGER NOT NULL DEFAULT 0,
  providers_total INTEGER NOT NULL DEFAULT 0,
  payload_json TEXT NOT NULL DEFAULT '{}',
  digest TEXT NOT NULL,
  source TEXT NOT NULL DEFAULT 'sweeper',
  UNIQUE (digest)
);

CREATE INDEX IF NOT EXISTS idx_selfhealth_snapshots_captured_at
  ON selfhealth_snapshots (captured_at DESC);

CREATE INDEX IF NOT EXISTS idx_selfhealth_snapshots_digest
  ON selfhealth_snapshots (digest);
