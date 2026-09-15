-- Assets table — owned by internal/assets/. Embedded by schema.go and applied
-- via database.EnsureSchemas at boot (api/main.go through the modules.AllSchemas
-- registry). The row is a catalog entry; the file bytes live on disk under
-- file_path. (brand_id, filename) is unique so a brand cannot hold two files of
-- the same name — re-uploading replaces the row in place. Times are RFC3339
-- strings matching the wire format and the time.Time round-trip in sqlite.go.
-- Use CREATE TABLE IF NOT EXISTS so re-runs are no-ops.
CREATE TABLE IF NOT EXISTS assets (
  id         TEXT PRIMARY KEY,
  brand_id   TEXT NOT NULL,
  filename   TEXT NOT NULL,
  mime_type  TEXT NOT NULL,
  file_path  TEXT NOT NULL,
  size       INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  UNIQUE (brand_id, filename)
);

CREATE INDEX IF NOT EXISTS idx_assets_brand ON assets(brand_id, created_at DESC);
