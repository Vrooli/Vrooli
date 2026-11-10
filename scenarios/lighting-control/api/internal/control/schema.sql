CREATE TABLE IF NOT EXISTS lighting_controllers (
  controller_id TEXT PRIMARY KEY,
  backend_id TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  vendor TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  enrolled INTEGER NOT NULL DEFAULT 0,
  last_seen_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS lighting_zones (
  controller_id TEXT NOT NULL,
  zone_id TEXT NOT NULL,
  fingerprint TEXT NOT NULL,
  name TEXT NOT NULL,
  led_count INTEGER NOT NULL,
  supports_static_color INTEGER NOT NULL DEFAULT 1,
  PRIMARY KEY (controller_id, zone_id),
  UNIQUE (controller_id, fingerprint)
);

CREATE TABLE IF NOT EXISTS lighting_desired_state (
  controller_id TEXT NOT NULL,
  zone_id TEXT NOT NULL,
  state BLOB NOT NULL,
  last_apply BLOB,
  updated_at TEXT NOT NULL,
  PRIMARY KEY (controller_id, zone_id)
);

CREATE TABLE IF NOT EXISTS lighting_manual_override (
  singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
  all_enrolled INTEGER NOT NULL,
  controller_ids BLOB NOT NULL,
  created_at TEXT NOT NULL,
  expires_at TEXT
);
