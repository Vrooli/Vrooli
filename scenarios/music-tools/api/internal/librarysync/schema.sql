-- Durable bridge to Music Library. Jobs are scanned after terminal success,
-- then their existing immutable result blob is retained here for retry.
CREATE TABLE IF NOT EXISTS music_library_sync_outbox (
  job_id TEXT PRIMARY KEY,
  track_sha256 TEXT NOT NULL DEFAULT '',
  result_ref TEXT NOT NULL,
  state TEXT NOT NULL CHECK (state IN ('pending','delivered','blocked')),
  attempts INTEGER NOT NULL DEFAULT 0,
  available_at TEXT NOT NULL,
  lease_token TEXT NOT NULL DEFAULT '',
  lease_until TEXT NOT NULL DEFAULT '',
  last_error TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  delivered_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS music_library_sync_due
  ON music_library_sync_outbox(state, available_at, lease_until, created_at);
