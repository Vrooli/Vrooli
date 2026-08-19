CREATE TABLE IF NOT EXISTS delivery_attempts (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  notification_id TEXT NOT NULL,
  channel TEXT NOT NULL,
  machine_id TEXT NOT NULL DEFAULT '',
  attempt_number INTEGER NOT NULL,
  outcome TEXT NOT NULL,
  reason TEXT NOT NULL,
  next_attempt_at TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_delivery_attempts_due ON delivery_attempts(next_attempt_at, outcome);
CREATE TABLE IF NOT EXISTS receipts (
  id TEXT PRIMARY KEY,
  notification_id TEXT NOT NULL,
  channel TEXT NOT NULL,
  machine_id TEXT NOT NULL DEFAULT '',
  provider_id TEXT NOT NULL DEFAULT '',
  delivered_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS machine_channel_status (
  machine_id TEXT NOT NULL,
  channel TEXT NOT NULL,
  disposition TEXT NOT NULL,
  reason TEXT NOT NULL,
  observed_at TEXT NOT NULL,
  PRIMARY KEY(machine_id, channel)
);
