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
  asset_kind    TEXT NOT NULL DEFAULT 'component',
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

CREATE INDEX IF NOT EXISTS idx_components_asset_kind
  ON components(asset_kind);

CREATE TABLE IF NOT EXISTS component_asset_dependencies (
  component_id TEXT NOT NULL,
  library_id   TEXT NOT NULL,
  version      TEXT NOT NULL,
  PRIMARY KEY (component_id, library_id, version)
);

CREATE INDEX IF NOT EXISTS idx_component_asset_dependencies_library
  ON component_asset_dependencies(library_id, version);

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
  created_at      TEXT NOT NULL DEFAULT '',
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

-- Derived styling contract for one immutable version unit. This is rebuilt
-- from source during indexing; it is never authored in component.json.
CREATE TABLE IF NOT EXISTS component_version_required_tokens (
  version_id TEXT NOT NULL,
  property   TEXT NOT NULL,
  PRIMARY KEY (version_id, property)
);

CREATE INDEX IF NOT EXISTS idx_component_version_required_tokens_version
  ON component_version_required_tokens(version_id, property);

CREATE TABLE IF NOT EXISTS component_version_required_token_patterns (
  version_id TEXT NOT NULL,
  pattern    TEXT NOT NULL,
  PRIMARY KEY (version_id, pattern)
);

CREATE INDEX IF NOT EXISTS idx_component_version_required_token_patterns_version
  ON component_version_required_token_patterns(version_id, pattern);

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

-- story.json replaces examples/control/setup metadata. It is stored as one
-- typed projection per asset version so all consumers share validated source.
CREATE TABLE IF NOT EXISTS component_stories (
  id               TEXT PRIMARY KEY,
  component_id     TEXT NOT NULL,
  library_id       TEXT NOT NULL,
  version          TEXT NOT NULL,
  schema_version   INTEGER NOT NULL,
  kind             TEXT NOT NULL,
  title            TEXT NOT NULL DEFAULT '',
  args_json        TEXT NOT NULL,
  environment_json TEXT NOT NULL,
  stories_json     TEXT NOT NULL,
  contract_json    TEXT NOT NULL,
  source_path      TEXT NOT NULL,
  indexed_at       TEXT NOT NULL,
  UNIQUE(component_id, version)
);

CREATE INDEX IF NOT EXISTS idx_component_stories_component_version
  ON component_stories(component_id, version);

-- Durable reports are intentionally separate from catalog source. Contracts
-- remain Git-tracked/versioned; reports are execution evidence with bounded
-- normalized JSON details.
CREATE TABLE IF NOT EXISTS component_test_reports (
  id TEXT PRIMARY KEY,
  component_id TEXT NOT NULL,
  root_library_id TEXT NOT NULL,
  root_version TEXT NOT NULL,
  include_closure INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  verdict TEXT NOT NULL,
  results_json TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_component_test_reports_component_created
  ON component_test_reports(component_id, created_at DESC);

-- Compact version-scoped test history. Payload rows are bounded, while these
-- counters preserve the complete verdict history for analytical consumers.
CREATE TABLE IF NOT EXISTS component_version_test_rollup (
  library_id TEXT NOT NULL,
  version TEXT NOT NULL,
  runs_total INTEGER NOT NULL DEFAULT 0,
  runs_passed INTEGER NOT NULL DEFAULT 0,
  runs_failed INTEGER NOT NULL DEFAULT 0,
  runs_blocked INTEGER NOT NULL DEFAULT 0,
  first_pass_report_id TEXT NOT NULL DEFAULT '',
  first_fail_report_id TEXT NOT NULL DEFAULT '',
  last_run_at TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (library_id, version)
);

-- Durable envelope for a corpus run. The per-version verdict map lets a
-- resumed request distinguish completed non-blocked work from versions that
-- were blocked by an unavailable browser boundary.
CREATE TABLE IF NOT EXISTS component_test_sweeps (
  id TEXT PRIMARY KEY,
  component_filter TEXT NOT NULL DEFAULT '',
  include_closure INTEGER NOT NULL DEFAULT 0,
  started_at TEXT NOT NULL,
  completed_at TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL,
  results_json TEXT NOT NULL DEFAULT '{}',
  errors_json TEXT NOT NULL DEFAULT '[]'
);
CREATE INDEX IF NOT EXISTS idx_component_test_sweeps_status_started
  ON component_test_sweeps(status, started_at DESC);
