CREATE TABLE IF NOT EXISTS component_versions (
  id              TEXT PRIMARY KEY,
  component_id    TEXT NOT NULL,
  library_id      TEXT NOT NULL DEFAULT '',
  version         TEXT NOT NULL DEFAULT '',
  status          TEXT NOT NULL DEFAULT '',
  source_path     TEXT NOT NULL DEFAULT '',
  content         TEXT NOT NULL,
  content_sha256  TEXT NOT NULL,
  changelog_md    TEXT NOT NULL DEFAULT '',
  indexed_at      TEXT NOT NULL,
  released_at     TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_component_versions_component_recorded
  ON component_versions(component_id, indexed_at DESC);

CREATE INDEX IF NOT EXISTS idx_component_versions_component_version
  ON component_versions(component_id, version);
