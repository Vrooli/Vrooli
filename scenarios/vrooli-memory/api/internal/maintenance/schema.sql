CREATE TABLE IF NOT EXISTS maintenance_runs (
  id TEXT PRIMARY KEY,
  started_at TEXT NOT NULL,
  completed_at TEXT NOT NULL DEFAULT '',
  -- Compaction is recorded on the run, not on maintenance_outcomes, because the
  -- canopy belongs to the corpus rather than to any one coding harness.
  compaction_status TEXT NOT NULL DEFAULT '',
  compaction_error TEXT NOT NULL DEFAULT '',
  compacted_count INTEGER NOT NULL DEFAULT 0,
  frontier_before INTEGER NOT NULL DEFAULT 0,
  frontier_after INTEGER NOT NULL DEFAULT 0,
  frontier_target INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_maintenance_runs_started ON maintenance_runs(started_at DESC);
CREATE TABLE IF NOT EXISTS maintenance_outcomes (
  run_id TEXT NOT NULL REFERENCES maintenance_runs(id) ON DELETE CASCADE,
  runtime TEXT NOT NULL,
  import_status TEXT NOT NULL,
  import_error TEXT NOT NULL DEFAULT '',
  projection_status TEXT NOT NULL,
  projection_error TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL,
  completed_at TEXT NOT NULL DEFAULT '',
  PRIMARY KEY(run_id,runtime)
);
