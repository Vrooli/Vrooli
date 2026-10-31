CREATE TABLE IF NOT EXISTS forecast_snapshots (
  id TEXT PRIMARY KEY,
  subject TEXT NOT NULL,
  input_fingerprint TEXT NOT NULL,
  generated_at TEXT NOT NULL,
  horizon_start TEXT NOT NULL,
  horizon_end TEXT NOT NULL,
  central_finish TEXT NOT NULL DEFAULT '',
  cautious_finish TEXT NOT NULL DEFAULT '',
  result_state TEXT NOT NULL,
  risk_state TEXT NOT NULL,
  explanation TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_forecast_snapshots_subject_fingerprint ON forecast_snapshots(subject, input_fingerprint);
CREATE INDEX IF NOT EXISTS idx_forecast_snapshots_subject_created ON forecast_snapshots(subject, created_at DESC);

CREATE TABLE IF NOT EXISTS forecast_change_records (
  id TEXT PRIMARY KEY,
  subject TEXT NOT NULL,
  previous_snapshot_id TEXT NOT NULL,
  snapshot_id TEXT NOT NULL,
  explanation TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_forecast_changes_subject_created ON forecast_change_records(subject, created_at DESC);
