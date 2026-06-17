-- Model runtime-state overlay — owned by internal/models/. Embedded by
-- schema.go and applied via database.EnsureSchemas at boot through the
-- modules.AllSchemas registry.
--
-- The seed catalog (registry.seed.json) is the read-only baseline; this table
-- records ONLY explicit operator overrides of a model's enabled state. A model
-- with no row here uses its seed default — so an empty table means "ship the
-- seed defaults", and toggling a model writes exactly one row. Heavier runtime
-- state (installed/checksum) joins this table in a later phase.
CREATE TABLE IF NOT EXISTS model_state (
  id      TEXT PRIMARY KEY,
  enabled INTEGER NOT NULL
);
