-- Routes table — owned by internal/routes/. The exposure manifest (SSOT):
-- which scenario is exposed at which subdomain/port, in which tier, under
-- which lease. Embedded by schema.go and applied via database.EnsureSchemas
-- at boot (api/main.go through the modules.AllSchemas registry). Times are
-- stored as RFC3339Nano strings matching the wire format and the time.Time
-- round-trip in sqlite.go::scanRoute. Use CREATE TABLE IF NOT EXISTS so
-- re-runs are no-ops; add columns with ALTER TABLE ... ADD COLUMN (migrate,
-- never recreate).
CREATE TABLE IF NOT EXISTS routes (
  id             TEXT PRIMARY KEY,
  subdomain      TEXT NOT NULL UNIQUE,
  scenario       TEXT NOT NULL,
  domain         TEXT NOT NULL DEFAULT 'itsagitime.com',
  local_port     INTEGER NOT NULL,
  tier           TEXT NOT NULL DEFAULT 'leased',
  lease_id       TEXT NOT NULL DEFAULT '',
  enabled        INTEGER NOT NULL DEFAULT 1,
  health_path    TEXT NOT NULL DEFAULT '/health',
  -- Provenance (scenario|external). Defaults to scenario for rows written
  -- before the column existed. External routes carry service_target instead
  -- of deriving http://localhost:<local_port>.
  source         TEXT NOT NULL DEFAULT 'scenario',
  service_target TEXT NOT NULL DEFAULT '',
  created_at     TEXT NOT NULL,
  updated_at     TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_routes_scenario ON routes(scenario);
CREATE INDEX IF NOT EXISTS idx_routes_tier ON routes(tier);
-- No index on source: it is a low-cardinality column on a tiny table, and a
-- CREATE INDEX on it would crash with a cryptic "no such column" on a
-- pre-existing DB that predates the column — masking EnsureSchemas' helpful
-- drift-check guidance (storage-steer §5: apply a one-shot ALTER TABLE ...
-- ADD COLUMN migration). source/service_target stay declared in the CREATE
-- TABLE above so fresh DBs get them and the drift check covers existing ones.
