CREATE TABLE IF NOT EXISTS monitoring_schedules (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  profile TEXT NOT NULL,
  baseline_snapshot_id TEXT NOT NULL,
  interval_minutes INTEGER NOT NULL,
  enabled INTEGER NOT NULL,
  latency_threshold_ms INTEGER NOT NULL,
  unavailable_threshold INTEGER NOT NULL,
  effects_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS monitoring_runs (
  id TEXT PRIMARY KEY,
  schedule_id TEXT NOT NULL,
  snapshot_id TEXT NOT NULL,
  status TEXT NOT NULL,
  summary TEXT NOT NULL,
  regression_detected INTEGER NOT NULL,
  effects_json TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_monitoring_runs_schedule_created
  ON monitoring_runs(schedule_id, created_at DESC);

CREATE TABLE IF NOT EXISTS monitoring_alerts (
  id TEXT PRIMARY KEY,
  schedule_id TEXT NOT NULL,
  snapshot_id TEXT NOT NULL,
  severity TEXT NOT NULL,
  status TEXT NOT NULL,
  summary TEXT NOT NULL,
  evidence_json TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_monitoring_alerts_schedule_created
  ON monitoring_alerts(schedule_id, created_at DESC);
