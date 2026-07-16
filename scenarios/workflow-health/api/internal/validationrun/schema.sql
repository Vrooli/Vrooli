CREATE TABLE IF NOT EXISTS validation_runs (
  id TEXT PRIMARY KEY,
  scenario TEXT NOT NULL,
  target_path TEXT NOT NULL,
  idempotency_key TEXT NOT NULL UNIQUE,
  parent_run_id TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL,
  created_at TEXT NOT NULL,
  started_at TEXT NOT NULL DEFAULT '',
  completed_at TEXT NOT NULL DEFAULT '',
  eta_seconds INTEGER NOT NULL DEFAULT 0,
  preliminary_result BLOB NOT NULL,
  terminal_result BLOB,
  error_code TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  artifact_refs BLOB NOT NULL DEFAULT '[]',
  cancellation_requested INTEGER NOT NULL DEFAULT 0,
  version INTEGER NOT NULL DEFAULT 1
);
CREATE INDEX IF NOT EXISTS idx_validation_runs_parent_run_id ON validation_runs(parent_run_id);
CREATE INDEX IF NOT EXISTS idx_validation_runs_state ON validation_runs(state);
