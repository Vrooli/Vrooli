-- Nodes table — owned by internal/registry/. Embedded by schema.go and applied
-- via database.EnsureSchemas at boot (api/main.go through modules.AllSchemas).
-- Durable identity of trusted Vrooli nodes (OT-P0-001). Times are RFC3339Nano
-- strings matching the wire format and the time.Time round-trip in
-- sqlite.go::scanNode. Capabilities and scopes are JSON-encoded string arrays.
-- Use CREATE TABLE IF NOT EXISTS so re-runs are no-ops (migrate, never
-- recreate). Postgres-compatible column types for forward scale.
CREATE TABLE IF NOT EXISTS nodes (
  id           TEXT PRIMARY KEY,
  name         TEXT NOT NULL,
  os           TEXT NOT NULL DEFAULT '',
  arch         TEXT NOT NULL DEFAULT '',
  revision     TEXT NOT NULL DEFAULT '',
  endpoint     TEXT NOT NULL DEFAULT '',
  capabilities TEXT NOT NULL DEFAULT '[]',
  scopes       TEXT NOT NULL DEFAULT '[]',
  created_at   TEXT NOT NULL,
  updated_at   TEXT NOT NULL,
  last_seen_at TEXT NOT NULL DEFAULT '',
  revoked_at   TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_nodes_created_at ON nodes(created_at DESC);
