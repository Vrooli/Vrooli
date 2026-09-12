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

-- Model install state (IMG-P0-007). A model's weights are downloaded on opt-in
-- (`image-tools models install <id>`) into <models-root>/models/<id>; this table
-- records that the download completed, where it landed, the checksum pinned on
-- first real download (NEVER hand-written — see DECISIONS), and the on-disk
-- byte size. Absence of a row (or installed=0) means "not installed", and AI ops
-- refuse with an actionable hint rather than launching a doomed job. Forward-only
-- declarative; a new table (not an ALTER of model_state) so existing overlays are
-- untouched.
CREATE TABLE IF NOT EXISTS model_install (
  id           TEXT PRIMARY KEY,
  installed    INTEGER NOT NULL DEFAULT 0,
  path         TEXT NOT NULL DEFAULT '',
  checksum     TEXT NOT NULL DEFAULT '',
  size_bytes   INTEGER NOT NULL DEFAULT 0,
  installed_at TEXT NOT NULL DEFAULT ''
);

-- Custom / fine-tuned local model entries (IMG-P0-007). The seed catalog
-- (registry.seed.json) is the read-only baseline; operators register their own
-- local models here as a full JSON-serialized models.Model, merged on top of the
-- seed at read time. Keyed by id; an id colliding with the seed is rejected by
-- the management layer so custom entries can only ADD, never shadow the baseline.
CREATE TABLE IF NOT EXISTS custom_model (
  id   TEXT PRIMARY KEY,
  json TEXT NOT NULL
);

-- Per-operation default-model override (IMG-P0-007 settings). The seed declares
-- each op's default_for model; an operator can pin a different enabled model as
-- THE default for an op here, and selection applies it when no per-call override
-- is given. Keyed by operation; absence means "use the seed default". A row is
-- still validated at selection time (op-support, enabled, host-runnable), so a
-- stale pin degrades gracefully rather than breaking the op.
CREATE TABLE IF NOT EXISTS op_default (
  operation TEXT PRIMARY KEY,
  model_id  TEXT NOT NULL
);
