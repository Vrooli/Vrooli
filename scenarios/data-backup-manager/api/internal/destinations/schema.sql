-- Destinations table — owned by internal/destinations/. Embedded by schema.go
-- and applied via database.EnsureSchemas at boot through the modules.AllSchemas
-- registry. Keyed by name: a destination is a unique kopia repository. Times
-- are RFC3339Nano strings matching the wire format and the time.Time round-trip
-- in sqlite.go::scanDest. CREATE ... IF NOT EXISTS so re-runs are no-ops.
CREATE TABLE IF NOT EXISTS destinations (
  id                   TEXT PRIMARY KEY,
  name                 TEXT NOT NULL UNIQUE,
  backend_kind         TEXT NOT NULL,
  -- location is the operator-facing bundle root; repository_location is the
  -- concrete kopia repository path nested inside it (filesystem) or the same
  -- bucket/prefix (s3). Split so a detached drive shows the human bundle root.
  location             TEXT NOT NULL,
  repository_location  TEXT NOT NULL DEFAULT '',
  cap_bytes            INTEGER NOT NULL DEFAULT 0,
  cap_policy           TEXT NOT NULL DEFAULT 'alert_block',
  encryption_algorithm TEXT NOT NULL DEFAULT '',
  secret_ref           TEXT NOT NULL DEFAULT '',
  created_at           TEXT NOT NULL,
  updated_at           TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_destinations_name ON destinations(name);
