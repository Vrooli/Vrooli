CREATE TABLE IF NOT EXISTS authoring_sessions (
  id TEXT PRIMARY KEY,
  scenario TEXT NOT NULL,
  target_path TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS authoring_pages (
  session_id TEXT NOT NULL,
  page_id TEXT NOT NULL,
  path TEXT NOT NULL,
  title TEXT NOT NULL,
  status TEXT NOT NULL,
  json TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (session_id, page_id),
  FOREIGN KEY (session_id) REFERENCES authoring_sessions(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_authoring_sessions_scenario
  ON authoring_sessions (scenario, updated_at);
