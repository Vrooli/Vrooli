-- Targets table — owned by internal/targets/. Embedded by schema.go and
-- applied via database.EnsureSchemas at boot through the modules.AllSchemas
-- registry. Keyed by (owner, name): an owning scenario re-registers
-- idempotently on boot, so the unique constraint is what makes the catalog
-- reconstructable. Times are RFC3339Nano strings matching the wire format and
-- the time.Time round-trip in sqlite.go::scanTarget. CREATE ... IF NOT EXISTS
-- so re-runs are no-ops.
CREATE TABLE IF NOT EXISTS targets (
  id          TEXT PRIMARY KEY,
  owner       TEXT NOT NULL,
  name        TEXT NOT NULL,
  source_kind TEXT NOT NULL,
  locator     TEXT NOT NULL,
  critical    INTEGER NOT NULL DEFAULT 0,
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL,
  UNIQUE (owner, name)
);

CREATE INDEX IF NOT EXISTS idx_targets_owner ON targets(owner, name);
