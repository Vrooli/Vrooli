CREATE TABLE IF NOT EXISTS event_outbox (
  event_id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL DEFAULT '',
  program_id TEXT NOT NULL DEFAULT '',
  kind INTEGER NOT NULL,
  occurred_at TEXT NOT NULL,
  payload TEXT NOT NULL,
  attempt_count INTEGER NOT NULL DEFAULT 0,
  last_error TEXT NOT NULL DEFAULT '',
  next_attempt_at TEXT NOT NULL,
  state TEXT NOT NULL CHECK (state IN ('pending', 'delivered', 'dead'))
);

CREATE INDEX IF NOT EXISTS idx_event_outbox_pending ON event_outbox(state, next_attempt_at, occurred_at);
CREATE INDEX IF NOT EXISTS idx_event_outbox_session ON event_outbox(session_id, occurred_at);
CREATE INDEX IF NOT EXISTS idx_event_outbox_kind ON event_outbox(kind, occurred_at);
