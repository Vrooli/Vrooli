CREATE TABLE IF NOT EXISTS research_attempt_outbox (
  attempt_id TEXT PRIMARY KEY,
  task_id TEXT NOT NULL DEFAULT '',
  payload BLOB NOT NULL,
  payload_sha256 TEXT NOT NULL,
  delivery_identity TEXT NOT NULL UNIQUE,
  state TEXT NOT NULL DEFAULT 'pending',
  attempts INTEGER NOT NULL DEFAULT 0,
  next_attempt_at TEXT NOT NULL,
  lease_until TEXT NOT NULL DEFAULT '',
  last_error TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  delivered_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_research_attempt_outbox_ready
  ON research_attempt_outbox(state, next_attempt_at, lease_until);
