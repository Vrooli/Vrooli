CREATE TABLE IF NOT EXISTS workspace_entitlements (
  workspace_id TEXT PRIMARY KEY,
  version INTEGER NOT NULL DEFAULT 1,
  optional_compute INTEGER NOT NULL DEFAULT 1,
  monthly_limit INTEGER NOT NULL DEFAULT 0,
  used_units INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS entitlement_reservations (
  workspace_id TEXT NOT NULL,
  operation_id TEXT NOT NULL,
  units INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY(workspace_id, operation_id)
);
