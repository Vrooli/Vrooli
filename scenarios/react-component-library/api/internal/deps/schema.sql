-- Deps domain schema — owned by internal/deps/.
-- Applied via database.EnsureSchemas at boot through modules.AllSchemas().
-- Idempotent: CREATE TABLE/INDEX IF NOT EXISTS so re-runs are no-ops.
--
-- component_dep_declarations carries the parsed @deps JSON field from a
-- component's header comment. One row per (component_id, dep_name).
-- component_id is a soft FK to components.id (no REFERENCES per
-- storage-steer per-domain rule).
CREATE TABLE IF NOT EXISTS component_dep_declarations (
  component_id  TEXT NOT NULL,
  library_id    TEXT NOT NULL DEFAULT '',
  dep_name      TEXT NOT NULL,
  version_range TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (component_id, dep_name)
);

CREATE INDEX IF NOT EXISTS idx_dep_decls_component_id
  ON component_dep_declarations(component_id);

CREATE INDEX IF NOT EXISTS idx_dep_decls_library_id
  ON component_dep_declarations(library_id);
