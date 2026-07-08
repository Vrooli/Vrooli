-- Components domain schema — owned by internal/components/.
-- Applied via database.EnsureSchemas at boot through the
-- modules.AllSchemas registry. Idempotent: CREATE TABLE/INDEX IF NOT
-- EXISTS so re-runs are no-ops.
--
-- `components` is the indexed registry row, sourced from
-- library/components/<slug>/component.json. Cross-domain references
-- (versions, adoptions, deps) store component_id as a soft FK — no
-- REFERENCES constraint, per storage-steer.
CREATE TABLE IF NOT EXISTS components (
  id            TEXT PRIMARY KEY,
  library_id    TEXT NOT NULL UNIQUE,
  slug          TEXT NOT NULL UNIQUE,
  display_name  TEXT NOT NULL DEFAULT '',
  description   TEXT NOT NULL DEFAULT '',
  slot          TEXT NOT NULL DEFAULT '',
  category      TEXT NOT NULL DEFAULT '',
  source_path   TEXT NOT NULL,
  version       TEXT NOT NULL DEFAULT '',
  latest_version TEXT NOT NULL DEFAULT '',
  draft_version TEXT NOT NULL DEFAULT '',
  manifest_path TEXT NOT NULL DEFAULT '',
  tags          TEXT NOT NULL DEFAULT '',
  indexed_at    TEXT NOT NULL,
  updated_at    TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_components_indexed_at
  ON components(indexed_at DESC);

CREATE INDEX IF NOT EXISTS idx_components_display_name
  ON components(display_name);

CREATE INDEX IF NOT EXISTS idx_components_category
  ON components(category);

-- Case-insensitive index supporting ORDER BY display_name COLLATE NOCASE,
-- the path req SF-001 takes when a substring match is present. Required
-- to meet the p95 < 100ms budget at 1k+ rows (req SF-003).
CREATE INDEX IF NOT EXISTS idx_components_display_name_nocase
  ON components(display_name COLLATE NOCASE);

CREATE TABLE IF NOT EXISTS component_versions (
  id              TEXT PRIMARY KEY,
  component_id    TEXT NOT NULL,
  library_id      TEXT NOT NULL,
  version         TEXT NOT NULL,
  status          TEXT NOT NULL,
  source_path     TEXT NOT NULL,
  content         TEXT NOT NULL DEFAULT '',
  content_sha256  TEXT NOT NULL,
  changelog_md    TEXT NOT NULL DEFAULT '',
  indexed_at      TEXT NOT NULL,
  released_at     TEXT NOT NULL DEFAULT '',
  UNIQUE(component_id, version)
);

CREATE INDEX IF NOT EXISTS idx_component_versions_component_status
  ON component_versions(component_id, status, version);

CREATE TABLE IF NOT EXISTS component_headers (
  component_id  TEXT NOT NULL,
  field         TEXT NOT NULL,
  value         TEXT NOT NULL,
  PRIMARY KEY (component_id, field)
);

CREATE INDEX IF NOT EXISTS idx_component_headers_field_value
  ON component_headers(field, value);

CREATE TABLE IF NOT EXISTS component_design_affinities (
  component_id  TEXT NOT NULL,
  style_id      TEXT NOT NULL,
  affinity      TEXT NOT NULL,
  reason        TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (component_id, style_id)
);

CREATE INDEX IF NOT EXISTS idx_component_design_affinities_style_affinity
  ON component_design_affinities(style_id, affinity);
