-- Model enabled-state overrides — owned by internal/models/. Embedded by
-- schema.go and applied via database.EnsureSchemas at boot through the
-- modules.AllSchemas registry.
--
-- The seed catalog (registry.seed.json) is the read-only baseline; this table
-- records ONLY explicit overrides of a model's enabled state, each with its
-- provenance (who, when, why, whether pinned) and the seed's enabled flag at the
-- moment it was written. A model with no row uses its seed default. The rule that
-- decides when a later seed change supersedes an override is documented in
-- internal/catalogstate/overlay.go and docs/reference/model-registry.md.
-- seed_enabled is -1 until a pre-provenance row is adopted at boot.
CREATE TABLE IF NOT EXISTS model_override (
  id              TEXT PRIMARY KEY,
  enabled         INTEGER NOT NULL,
  set_by          TEXT NOT NULL,
  set_at          TEXT NOT NULL DEFAULT '',
  reason          TEXT NOT NULL DEFAULT '',
  pinned          INTEGER NOT NULL DEFAULT 0,
  seed_enabled    INTEGER NOT NULL DEFAULT -1,
  superseded_at   TEXT NOT NULL DEFAULT '',
  superseded_note TEXT NOT NULL DEFAULT ''
);

-- Model install state (IMG-P0-007). A model's weights are downloaded on opt-in
-- (`image-tools models install <id>`) into <models-root>/models/<id>; this table
-- records that the download completed, where it landed, the checksum pinned on
-- first real download (NEVER hand-written — see DECISIONS), and the on-disk
-- byte size. Absence of a row (or installed=0) means "not installed", and AI ops
-- refuse with an actionable hint rather than launching a doomed job. Forward-only
-- declarative; a new table (not an ALTER of the override table) so existing overlays are
-- untouched.
CREATE TABLE IF NOT EXISTS model_install (
  id           TEXT PRIMARY KEY,
  installed    INTEGER NOT NULL DEFAULT 0,
  path         TEXT NOT NULL DEFAULT '',
  checksum     TEXT NOT NULL DEFAULT '',
  size_bytes   INTEGER NOT NULL DEFAULT 0,
  installed_at TEXT NOT NULL DEFAULT ''
);

-- Preparation phase is distinct from artifact acquisition. Readiness reconciles
-- this transition record with the current artifact/runtime smoke fingerprints.
CREATE TABLE IF NOT EXISTS model_lifecycle (
  id            TEXT PRIMARY KEY,
  state         TEXT NOT NULL,
  reason        TEXT NOT NULL,
  detail        TEXT NOT NULL DEFAULT '',
  artifact_hash TEXT NOT NULL DEFAULT '',
  runtime_hash  TEXT NOT NULL DEFAULT '',
  updated_at    TEXT NOT NULL
);

-- Build-operation intent is persisted before dispatch so an exporter service
-- remains attributable and recoverable across API restarts. It stores the
-- operation identity, never its environment, which can contain credentials.
CREATE TABLE IF NOT EXISTS model_build_operation (
  job_id        TEXT PRIMARY KEY,
  model_id      TEXT NOT NULL,
  intent_json   TEXT NOT NULL,
  state         TEXT NOT NULL,
  result        TEXT NOT NULL DEFAULT '',
  exit_status   INTEGER NOT NULL DEFAULT 0,
  updated_at    TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS model_build_operation_model_state
  ON model_build_operation(model_id, state);

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
