CREATE TABLE IF NOT EXISTS jobs (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  type TEXT NOT NULL,
  dedup_key TEXT NOT NULL,
  request_hash TEXT NOT NULL DEFAULT '',
  state TEXT NOT NULL,
  input_revisions_json TEXT NOT NULL DEFAULT '[]',
  attempts INTEGER NOT NULL DEFAULT 0,
  provider TEXT NOT NULL DEFAULT '',
  budget_units INTEGER NOT NULL DEFAULT 0,
  result_reference TEXT NOT NULL DEFAULT '',
  error_code TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(workspace_id, type, dedup_key)
);
CREATE INDEX IF NOT EXISTS idx_jobs_workspace_updated ON jobs(workspace_id, updated_at DESC);
