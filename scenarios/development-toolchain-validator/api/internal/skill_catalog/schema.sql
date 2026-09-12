-- skill_catalog table — owned by internal/skill_catalog/. Mirrors the
-- prompt-manager skill catalog. Times are stored as RFC3339 strings
-- matching the wire format and the time.Time round-trip in sqlite.go.
-- CREATE TABLE IF NOT EXISTS so re-runs are no-ops.
CREATE TABLE IF NOT EXISTS skill_catalog (
  id           TEXT PRIMARY KEY,
  version      TEXT NOT NULL,
  content_hash TEXT NOT NULL,
  synced_at    TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_skill_catalog_id ON skill_catalog(id);
