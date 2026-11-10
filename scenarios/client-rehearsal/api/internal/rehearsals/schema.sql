CREATE TABLE IF NOT EXISTS rehearsals (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  source_format TEXT NOT NULL,
  source_sha256 TEXT NOT NULL,
  credential_replacements INTEGER NOT NULL,
  simulator_profile TEXT NOT NULL,
  simulator_source_sha256 TEXT NOT NULL,
  state TEXT NOT NULL,
  trace_json TEXT NOT NULL,
  fault_summary TEXT NOT NULL,
  repair_summary TEXT NOT NULL,
  handoff_markdown TEXT NOT NULL,
  review_decision TEXT NOT NULL DEFAULT '',
  review_note TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  reviewed_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_rehearsals_created ON rehearsals(created_at DESC, id DESC);
