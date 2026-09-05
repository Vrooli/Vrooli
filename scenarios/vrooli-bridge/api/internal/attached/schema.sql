-- Durable attached-device projections owned by the attached domain.
CREATE TABLE IF NOT EXISTS bridge_attached_devices (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL DEFAULT '',
  host_node_id TEXT NOT NULL,
  kind TEXT NOT NULL,
  transport TEXT NOT NULL DEFAULT '',
  serial TEXT NOT NULL DEFAULT '',
  os_version TEXT NOT NULL DEFAULT '',
  trust_state TEXT NOT NULL,
  reachability TEXT NOT NULL,
  health_reason TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  revoked_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_attached_devices_host ON bridge_attached_devices(host_node_id);
