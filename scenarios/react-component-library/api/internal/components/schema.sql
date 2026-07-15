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

-- Supports the versions domain's history reads (Latest/List order by
-- indexed_at DESC per component). The UNIQUE(component_id, version)
-- constraint above already provides the (component_id, version) lookup
-- index, so no separate one is declared. component_versions is owned
-- solely by this schema — the versions domain is a read/append consumer.
CREATE INDEX IF NOT EXISTS idx_component_versions_component_recorded
  ON component_versions(component_id, indexed_at DESC);

-- A version is a file set. component_versions retains the entry-file mirror
-- for existing query paths; this child table is authoritative for companions.
CREATE TABLE IF NOT EXISTS component_version_files (
  version_id      TEXT NOT NULL,
  path            TEXT NOT NULL,
  content         TEXT NOT NULL DEFAULT '',
  content_sha256  TEXT NOT NULL,
  is_entry        INTEGER NOT NULL DEFAULT 0,
  -- Explicit per-file placement slot (e.g. "hook" for a companion). Empty
  -- means "unspecified": the adoption path resolver derives the slot from an
  -- extension heuristic (use*.ts -> hook) or the component's declared slot.
  -- Authored via component.json `fileSlots`. Explicit metadata wins.
  slot            TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (version_id, path)
);

CREATE INDEX IF NOT EXISTS idx_component_version_files_version
  ON component_version_files(version_id, is_entry DESC, path);

CREATE TABLE IF NOT EXISTS component_version_parity_reports (
  version_id TEXT PRIMARY KEY,
  report_json TEXT NOT NULL
);

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

CREATE TABLE IF NOT EXISTS component_examples (
  id            TEXT PRIMARY KEY,
  component_id  TEXT NOT NULL,
  library_id    TEXT NOT NULL,
  version       TEXT NOT NULL,
  name          TEXT NOT NULL,
  display_name  TEXT NOT NULL DEFAULT '',
  props_json    TEXT NOT NULL DEFAULT '{}',
  setup_json    TEXT NOT NULL DEFAULT '{}',
  expect_json   TEXT NOT NULL DEFAULT '[]',
  source_path   TEXT NOT NULL,
  indexed_at    TEXT NOT NULL,
  UNIQUE(component_id, version, name)
);

CREATE INDEX IF NOT EXISTS idx_component_examples_component_version
  ON component_examples(component_id, version, name);
