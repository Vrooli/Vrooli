CREATE TABLE IF NOT EXISTS packaged_launch_samples (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  scenario TEXT NOT NULL,
  ramp TEXT NOT NULL,
  strategy TEXT NOT NULL,
  platform TEXT NOT NULL,
  experiment_id TEXT NOT NULL,
  host_fingerprint TEXT NOT NULL,
  artifact_digest TEXT NOT NULL,
  condition TEXT NOT NULL,
  display TEXT NOT NULL,
  profiler_mode TEXT NOT NULL,
  power_state TEXT NOT NULL,
  captured_at TEXT NOT NULL,
  sample_json TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_packaged_samples_report
  ON packaged_launch_samples (scenario, ramp, strategy, condition, captured_at DESC);
CREATE INDEX IF NOT EXISTS idx_packaged_samples_experiment
  ON packaged_launch_samples (experiment_id, captured_at DESC);
CREATE TABLE IF NOT EXISTS packaged_launch_attempts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  scenario TEXT NOT NULL,
  ramp TEXT NOT NULL,
  strategy TEXT NOT NULL,
  experiment_id TEXT NOT NULL,
  condition TEXT NOT NULL,
  toggle_identity TEXT NOT NULL,
  attempted_at TEXT NOT NULL,
  attempt_json TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_packaged_attempts_experiment
  ON packaged_launch_attempts (experiment_id, scenario, strategy, condition, attempted_at);
