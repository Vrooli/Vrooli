-- Inventory domain — owned by internal/inventory/. Embedded by schema.go and
-- applied through modules.AllSchemas at boot.

CREATE TABLE IF NOT EXISTS device_groups (
  name       TEXT PRIMARY KEY,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS devices (
  id                   TEXT PRIMARY KEY,
  hostname             TEXT NOT NULL DEFAULT '',
  ip_address           TEXT NOT NULL DEFAULT '',
  mac_address          TEXT NOT NULL DEFAULT '',
  stable_id            TEXT NOT NULL DEFAULT '',
  resolver_client_id   TEXT NOT NULL DEFAULT '',
  group_name           TEXT NOT NULL DEFAULT 'unassigned',
  identity_confidence  TEXT NOT NULL,
  notes_json           TEXT NOT NULL DEFAULT '[]',
  last_seen            TEXT NOT NULL,
  created_at           TEXT NOT NULL,
  updated_at           TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_devices_group_updated
  ON devices(group_name, updated_at DESC, id);

CREATE INDEX IF NOT EXISTS idx_devices_identity
  ON devices(stable_id, resolver_client_id, mac_address, ip_address);
