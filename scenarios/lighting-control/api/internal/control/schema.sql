CREATE TABLE IF NOT EXISTS lighting_controllers (
  controller_id TEXT PRIMARY KEY,
  backend_id TEXT NOT NULL,
  name TEXT NOT NULL DEFAULT '',
  vendor TEXT NOT NULL DEFAULT '',
  description TEXT NOT NULL DEFAULT '',
  identity_source TEXT NOT NULL DEFAULT '',
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

-- Older desired rows mixed automatic output and user intent. Only explicit
-- new requests establish ownership; legacy bytes remain available for review.
CREATE TABLE IF NOT EXISTS lighting_zone_ownership (
  controller_id TEXT NOT NULL,
  zone_id TEXT NOT NULL,
  source TEXT NOT NULL CHECK (source IN ('manual', 'policy')),
  PRIMARY KEY (controller_id, zone_id)
);

CREATE TABLE IF NOT EXISTS lighting_previous_zone_state (
  controller_id TEXT NOT NULL,
  zone_id TEXT NOT NULL,
  state BLOB NOT NULL,
  last_apply BLOB,
  source TEXT NOT NULL,
  archived_at TEXT NOT NULL,
  PRIMARY KEY (controller_id, zone_id)
);

CREATE TABLE IF NOT EXISTS lighting_manual_override (
  singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
  all_enrolled INTEGER NOT NULL,
  controller_ids BLOB NOT NULL,
  created_at TEXT NOT NULL,
  expires_at TEXT
);

-- Only one policy request can be pending. Its completed control transition
-- survives a lost policy checkpoint without repeating destructive adoption.
CREATE TABLE IF NOT EXISTS lighting_policy_selection (
  singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
  request_id TEXT NOT NULL,
  requested_at TEXT NOT NULL
);

-- Causal override order is independent of wall-clock corrections. The cause
-- identifies a selection's own publication when its policy checkpoint retries.
CREATE TABLE IF NOT EXISTS lighting_override_order (
  singleton INTEGER PRIMARY KEY CHECK (singleton = 1),
  revision INTEGER NOT NULL CHECK (typeof(revision) = 'integer' AND revision >= 0),
  selection_id TEXT NOT NULL
);
