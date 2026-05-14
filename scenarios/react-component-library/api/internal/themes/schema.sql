-- Themes domain schema — owned by internal/themes/.
-- Applied via database.EnsureSchemas at boot through modules.AllSchemas().
-- Idempotent: CREATE TABLE/INDEX IF NOT EXISTS so re-runs are no-ops.
--
-- builtin_themes carries the seed-on-empty registry of themes shipped
-- with the library (vrooli-default, neutral-light, neutral-dark,
-- high-contrast). Scenario-derived themes are computed on demand and
-- not persisted (per req 12).
CREATE TABLE IF NOT EXISTS builtin_themes (
  id          TEXT PRIMARY KEY,
  name        TEXT NOT NULL,
  tokens_json TEXT NOT NULL DEFAULT '{}',
  created_at  TEXT NOT NULL DEFAULT (datetime('now'))
);
