-- Routes table — owned by internal/routes/. The exposure manifest (SSOT):
-- which scenario is exposed at which subdomain/port, in which tier, under
-- which lease. Embedded by schema.go and applied via database.EnsureSchemas
-- at boot (api/main.go through the modules.AllSchemas registry). Times are
-- stored as RFC3339Nano strings matching the wire format and the time.Time
-- round-trip in sqlite.go::scanRoute. Use CREATE TABLE IF NOT EXISTS so
-- re-runs are no-ops; add columns with ALTER TABLE ... ADD COLUMN (migrate,
-- never recreate).
CREATE TABLE IF NOT EXISTS routes (
  id          TEXT PRIMARY KEY,
  subdomain   TEXT NOT NULL UNIQUE,
  scenario    TEXT NOT NULL,
  domain      TEXT NOT NULL DEFAULT 'itsagitime.com',
  local_port  INTEGER NOT NULL,
  tier        TEXT NOT NULL DEFAULT 'leased',
  lease_id    TEXT NOT NULL DEFAULT '',
  enabled     INTEGER NOT NULL DEFAULT 1,
  health_path TEXT NOT NULL DEFAULT '/health',
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_routes_scenario ON routes(scenario);
CREATE INDEX IF NOT EXISTS idx_routes_tier ON routes(tier);
