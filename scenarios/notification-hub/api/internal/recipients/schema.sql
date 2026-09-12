CREATE TABLE IF NOT EXISTS recipients (
  id TEXT PRIMARY KEY,
  subject TEXT NOT NULL UNIQUE,
  trust_posture TEXT NOT NULL DEFAULT 'personal',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS devices (
  id TEXT PRIMARY KEY,
  recipient_id TEXT NOT NULL REFERENCES recipients(id) ON DELETE CASCADE,
  name TEXT NOT NULL,
  machine_id TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  UNIQUE(recipient_id, name)
);
CREATE TABLE IF NOT EXISTS channel_addresses (
  id TEXT PRIMARY KEY,
  device_id TEXT NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
  channel TEXT NOT NULL,
  address TEXT NOT NULL,
  approved_labels TEXT NOT NULL DEFAULT '["public"]',
  created_at TEXT NOT NULL,
  UNIQUE(device_id, channel)
);
CREATE TABLE IF NOT EXISTS push_subscriptions (
  id TEXT PRIMARY KEY,
  recipient_id TEXT NOT NULL REFERENCES recipients(id) ON DELETE CASCADE,
  device_id TEXT NOT NULL DEFAULT '',
  endpoint TEXT NOT NULL,
  p256dh TEXT NOT NULL,
  auth TEXT NOT NULL,
  origin TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  UNIQUE(recipient_id, endpoint)
);
CREATE TABLE IF NOT EXISTS quiet_windows (
  id TEXT PRIMARY KEY,
  recipient_id TEXT NOT NULL REFERENCES recipients(id) ON DELETE CASCADE,
  weekday INTEGER NOT NULL CHECK(weekday BETWEEN 0 AND 6),
  start_time TEXT NOT NULL,
  end_time TEXT NOT NULL,
  timezone TEXT NOT NULL,
  critical_override INTEGER NOT NULL DEFAULT 1
);
CREATE TABLE IF NOT EXISTS escalation_chains (
  id TEXT PRIMARY KEY,
  recipient_id TEXT NOT NULL REFERENCES recipients(id) ON DELETE CASCADE,
  ordinal INTEGER NOT NULL,
  channel TEXT NOT NULL,
  UNIQUE(recipient_id, ordinal)
);
