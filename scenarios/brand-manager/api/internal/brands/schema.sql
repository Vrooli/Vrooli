-- Brands tables — owned by internal/brands/. Embedded by schema.go and applied
-- via database.EnsureSchemas at boot (api/main.go through the modules.AllSchemas
-- registry). Facets (identity, colors, typography, voice) are stored as JSON
-- text columns; times are RFC3339 strings matching the wire format and the
-- time.Time round-trip in sqlite.go. Use CREATE TABLE IF NOT EXISTS so re-runs
-- are no-ops.
CREATE TABLE IF NOT EXISTS brands (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  identity    TEXT NOT NULL DEFAULT '{}',
  colors      TEXT NOT NULL DEFAULT '{}',
  typography  TEXT NOT NULL DEFAULT '{}',
  voice       TEXT NOT NULL DEFAULT '{}',
  notes       TEXT NOT NULL DEFAULT '',
  version     INTEGER NOT NULL DEFAULT 1,
  created_at  TEXT NOT NULL,
  updated_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_brands_updated_at ON brands(updated_at DESC);
CREATE INDEX IF NOT EXISTS idx_brands_name ON brands(name);

CREATE TABLE IF NOT EXISTS brand_versions (
  id         TEXT PRIMARY KEY,
  brand_id   TEXT NOT NULL,
  version    INTEGER NOT NULL,
  snapshot   TEXT NOT NULL,
  created_at TEXT NOT NULL,
  FOREIGN KEY (brand_id) REFERENCES brands(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_brand_versions_brand
  ON brand_versions(brand_id, version DESC);
