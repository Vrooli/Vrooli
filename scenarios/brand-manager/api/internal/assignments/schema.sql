-- Assignments table — owned by internal/assignments/. Embedded by schema.go and
-- applied via database.EnsureSchemas at boot (api/main.go through the
-- modules.AllSchemas registry). A scenario carries at most one brand, so
-- scenario_name is the natural unique key and AssignBrand upserts on it.
-- elements is a JSON text array; applied_at is an RFC3339 string matching the
-- wire format and the time.Time round-trip in sqlite.go. Use CREATE TABLE IF
-- NOT EXISTS so re-runs are no-ops.
CREATE TABLE IF NOT EXISTS assignments (
  id            TEXT PRIMARY KEY,
  brand_id      TEXT NOT NULL,
  scenario_name TEXT NOT NULL UNIQUE,
  brand_version INTEGER NOT NULL DEFAULT 0,
  elements      TEXT NOT NULL DEFAULT '[]',
  applied_at    TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_assignments_brand ON assignments(brand_id);
CREATE INDEX IF NOT EXISTS idx_assignments_applied_at ON assignments(applied_at DESC);
