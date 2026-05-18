-- Goldens table — owned by internal/golden/. Embedded by schema.go and
-- applied via database.EnsureSchemas at boot through the modules.AllSchemas
-- registry. Times are stored as RFC3339 strings matching the wire format and
-- the time.Time round-trip in sqlite.go::scanGolden. Use CREATE TABLE IF NOT
-- EXISTS so re-runs are no-ops.
CREATE TABLE IF NOT EXISTS goldens (
  id                      TEXT PRIMARY KEY,
  slug                    TEXT NOT NULL UNIQUE,
  template_id             TEXT NOT NULL,
  template_version_pinned TEXT NOT NULL,
  path                    TEXT NOT NULL,
  created_at              TEXT NOT NULL,
  last_regenerated_at     TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_goldens_slug ON goldens(slug);
