CREATE TABLE IF NOT EXISTS coding_jobs (
  id TEXT PRIMARY KEY,
  replay_key TEXT NOT NULL DEFAULT '',
  case_id TEXT NOT NULL,
  project TEXT NOT NULL,
  title TEXT NOT NULL,
  task TEXT NOT NULL,
  state TEXT NOT NULL,
  source_seal_ref TEXT NOT NULL,
  solver_run_ref TEXT NOT NULL DEFAULT '',
  evaluator_receipt_ref TEXT NOT NULL DEFAULT '',
  evaluator_receipt_sha256 TEXT NOT NULL DEFAULT '',
  score TEXT NOT NULL DEFAULT '',
  revision INTEGER NOT NULL DEFAULT 1,
  events_json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_coding_jobs_created ON coding_jobs(created_at DESC, id DESC);
