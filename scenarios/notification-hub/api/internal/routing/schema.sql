CREATE TABLE IF NOT EXISTS routing_decisions (
  id TEXT PRIMARY KEY,
  notification_id TEXT NOT NULL,
  channel TEXT NOT NULL,
  machine_id TEXT NOT NULL DEFAULT '',
  approved INTEGER NOT NULL,
  content_mode TEXT NOT NULL,
  reason TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS holds (
  id TEXT PRIMARY KEY,
  notification_id TEXT NOT NULL UNIQUE,
  release_at TEXT NOT NULL,
  released_at TEXT NOT NULL DEFAULT '',
  reason TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS suppressions (
  id TEXT PRIMARY KEY,
  notification_id TEXT NOT NULL,
  collapsed_into TEXT NOT NULL,
  dedupe_key TEXT NOT NULL,
  created_at TEXT NOT NULL
);
