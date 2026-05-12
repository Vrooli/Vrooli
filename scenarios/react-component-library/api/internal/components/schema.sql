-- Components domain schema — owned by internal/components/.
-- Applied via database.EnsureSchemas at boot through the
-- modules.AllSchemas registry. Idempotent: CREATE TABLE/INDEX IF NOT
-- EXISTS so re-runs are no-ops.
--
-- `components` is the indexed registry row, sourced from on-disk TSX
-- files with @libraryId headers. `component_headers` stores the raw
-- header field/value pairs so future filters can query without
-- reparsing files. Cross-domain references (versions, adoptions, deps)
-- store component_id as a soft FK — no REFERENCES constraint, per
-- storage-steer.
CREATE TABLE IF NOT EXISTS components (
  id            TEXT PRIMARY KEY,
  library_id    TEXT NOT NULL UNIQUE,
  display_name  TEXT NOT NULL DEFAULT '',
  description   TEXT NOT NULL DEFAULT '',
  source_path   TEXT NOT NULL,
  version       TEXT NOT NULL DEFAULT '',
  tags          TEXT NOT NULL DEFAULT '',
  indexed_at    TEXT NOT NULL,
  updated_at    TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_components_indexed_at
  ON components(indexed_at DESC);

CREATE INDEX IF NOT EXISTS idx_components_display_name
  ON components(display_name);

CREATE TABLE IF NOT EXISTS component_headers (
  component_id  TEXT NOT NULL,
  field         TEXT NOT NULL,
  value         TEXT NOT NULL,
  PRIMARY KEY (component_id, field)
);
