CREATE TABLE IF NOT EXISTS programs (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  source TEXT NOT NULL,
  provenance TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  completed_at TEXT NOT NULL DEFAULT '',
  stdout TEXT NOT NULL DEFAULT '',
  context_bytes INTEGER NOT NULL DEFAULT 0,
  output_limit_bytes INTEGER NOT NULL DEFAULT 0,
  failure_detail TEXT NOT NULL DEFAULT '',
  failure_shape TEXT NOT NULL DEFAULT '',
  failure_location TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_programs_session ON programs(session_id);
CREATE INDEX IF NOT EXISTS idx_programs_created ON programs(created_at);
CREATE INDEX IF NOT EXISTS idx_programs_failure_shape ON programs(failure_shape);
