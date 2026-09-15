CREATE TABLE IF NOT EXISTS harness_import_runs (
  id TEXT PRIMARY KEY,
  runtime TEXT NOT NULL,
  source_root TEXT NOT NULL,
  status TEXT NOT NULL,
  total_sources INTEGER NOT NULL,
  processed_sources INTEGER NOT NULL DEFAULT 0,
  imported_count INTEGER NOT NULL DEFAULT 0,
  existing_count INTEGER NOT NULL DEFAULT 0,
  failed_count INTEGER NOT NULL DEFAULT 0,
  current_path TEXT NOT NULL DEFAULT '',
  error_message TEXT NOT NULL DEFAULT '',
  started_at TEXT NOT NULL,
  completed_at TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_harness_import_runs_runtime_updated ON harness_import_runs(runtime, updated_at DESC);

CREATE TABLE IF NOT EXISTS harness_import_cursors (
  runtime TEXT PRIMARY KEY,
  source_root TEXT NOT NULL,
  cursor_path TEXT NOT NULL DEFAULT '',
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS harness_projections (
  runtime TEXT PRIMARY KEY,
  target_path TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  size_bytes INTEGER NOT NULL,
  size_lines INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);
