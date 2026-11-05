-- manifest table — owned by internal/manifest/. Primary key is the
-- (skill_id, golden_slug) tuple. Times stored as RFC3339 strings to
-- match the wire format and the time.Time round-trip in sqlite.go.
-- allowed_paths_json and content_rules_json hold JSON arrays so the
-- schema stays normalised at the column level (no positional CSV
-- shenanigans) while keeping the storage model dead-simple.
CREATE TABLE IF NOT EXISTS manifests (
  skill_id                TEXT NOT NULL,
  golden_slug             TEXT NOT NULL,
  allowed_paths_json      TEXT NOT NULL,
  content_rules_json      TEXT NOT NULL,
  wildcard_allowed        INTEGER NOT NULL,
  convergence_target      INTEGER NOT NULL,
  template_version_pinned TEXT NOT NULL,
  skill_version_pinned    TEXT NOT NULL,
  updated_at              TEXT NOT NULL,
  PRIMARY KEY (skill_id, golden_slug)
);

CREATE INDEX IF NOT EXISTS idx_manifests_skill_id    ON manifests(skill_id);
CREATE INDEX IF NOT EXISTS idx_manifests_golden_slug ON manifests(golden_slug);

-- manifest_stale_overrides records manual ClearStale invocations.
-- Staleness derivation (domain/staleness) consults this table to
-- suppress drift reports until the next manifest upsert.
CREATE TABLE IF NOT EXISTS manifest_stale_overrides (
  skill_id    TEXT NOT NULL,
  golden_slug TEXT NOT NULL,
  cleared_at  TEXT NOT NULL,
  PRIMARY KEY (skill_id, golden_slug)
);
