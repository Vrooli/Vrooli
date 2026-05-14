CREATE TABLE IF NOT EXISTS component_versions (
  id              TEXT PRIMARY KEY,
  component_id    TEXT NOT NULL,
  version         TEXT NOT NULL DEFAULT '',
  content         TEXT NOT NULL,
  content_sha256  TEXT NOT NULL,
  changelog_md    TEXT NOT NULL DEFAULT '',
  recorded_at     TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_component_versions_component_recorded
  ON component_versions(component_id, recorded_at DESC);

CREATE INDEX IF NOT EXISTS idx_component_versions_component_version
  ON component_versions(component_id, version);
