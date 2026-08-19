CREATE TABLE IF NOT EXISTS notifications (
  id TEXT PRIMARY KEY,
  requested_by TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  body TEXT NOT NULL,
  urgency TEXT NOT NULL,
  sensitivity_label TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  dedupe_key TEXT NOT NULL DEFAULT '',
  dedupe_window_seconds INTEGER NOT NULL DEFAULT 0,
  scheduled_at TEXT NOT NULL DEFAULT '',
  digest_key TEXT NOT NULL DEFAULT '',
  digest_window_seconds INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(requested_by, idempotency_key)
);
CREATE INDEX IF NOT EXISTS idx_notifications_dedupe ON notifications(requested_by, dedupe_key, created_at);
CREATE TABLE IF NOT EXISTS notification_events (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  notification_id TEXT NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
  state TEXT NOT NULL,
  reason TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_notification_events_notification ON notification_events(notification_id, id);
