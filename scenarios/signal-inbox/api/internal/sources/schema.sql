CREATE TABLE IF NOT EXISTS adapter_state (
  adapter_id TEXT PRIMARY KEY,
  risk_tier INTEGER NOT NULL,
  enabled INTEGER NOT NULL,
  last_run_at TEXT NOT NULL DEFAULT '',
  last_error TEXT NOT NULL DEFAULT '',
  disabled_reason TEXT NOT NULL DEFAULT ''
);
CREATE TABLE IF NOT EXISTS import_run (
  id TEXT PRIMARY KEY,
  adapter_id TEXT NOT NULL,
  created_count INTEGER NOT NULL,
  duplicate_count INTEGER NOT NULL,
  failed_count INTEGER NOT NULL,
  started_at TEXT NOT NULL,
  finished_at TEXT NOT NULL
);
