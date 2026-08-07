CREATE TABLE IF NOT EXISTS maintenance_runs (
  id TEXT PRIMARY KEY,
  started_at TEXT NOT NULL,
  completed_at TEXT NOT NULL DEFAULT ''
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
