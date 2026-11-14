-- Adapter runtime-state overlay — owned by internal/adapters/. Embedded by
-- schema.go and applied via database.EnsureSchemas at boot through the
-- modules.AllSchemas registry. Mirrors the model catalog's overlay tables
-- (internal/models/schema.sql) for the sibling conditioning-adapter catalog.
--
-- The seed catalog (adapters.seed.json) is the read-only baseline; these tables
-- record ONLY explicit operator state: enabled overrides, completed installs,
-- and add-only custom/imported adapters. An adapter with no row uses its seed
-- default. Forward-only declarative; re-runs are no-ops.

-- Enabled-state override over the seed default (one row per explicit toggle).
CREATE TABLE IF NOT EXISTS adapter_state (
  id      TEXT PRIMARY KEY,
  enabled INTEGER NOT NULL
);

-- Install state: an adapter's weights are downloaded on opt-in into
-- <root>/adapters/<id>; this records the completed download, where it landed,
-- the checksum pinned on first real download (NEVER hand-written), and the size.
CREATE TABLE IF NOT EXISTS adapter_install (
  id           TEXT PRIMARY KEY,
  installed    INTEGER NOT NULL DEFAULT 0,
  path         TEXT NOT NULL DEFAULT '',
  checksum     TEXT NOT NULL DEFAULT '',
  size_bytes   INTEGER NOT NULL DEFAULT 0,
  installed_at TEXT NOT NULL DEFAULT ''
);

-- Custom / imported adapter entries, stored as a full JSON-serialized
-- adapters.Adapter and merged on top of the seed at read time. Keyed by id; an
-- id colliding with the seed is rejected by the management layer so custom
-- entries can only ADD, never shadow the baseline (decision D5).
CREATE TABLE IF NOT EXISTS custom_adapter (
  id   TEXT PRIMARY KEY,
  json TEXT NOT NULL
);
