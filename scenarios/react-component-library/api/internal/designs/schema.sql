-- Designs domain schema — owned by internal/designs/.
-- Applied via database.EnsureSchemas at boot through modules.AllSchemas().
-- Idempotent: CREATE TABLE/INDEX IF NOT EXISTS so re-runs are no-ops.
--
-- A design is the durable object the studio edits. frames_json and theme_json
-- are stored as opaque JSON: the server never re-models the TemplateNode tree
-- (ui/src/workbench/templates/spec.ts owns that shape). The columns the server
-- does read — name, revision, frame_count, scenario, page, updated_at — are
-- materialised so list ordering, summaries and page binding filters do not
-- require parsing every document.
--
-- options_json holds the design-owned chrome (radius/display) as a small
-- validated JSON object. It is read by the renderer, never by list queries, so
-- it stays opaque to SQL and always defaults to {}.
--
-- Existing databases do not gain columns from CREATE TABLE IF NOT EXISTS;
-- internal/designs.Migrate adds scenario/page/options_json (and the asset table
-- above is created here because it is new, not a column on an existing table).
CREATE TABLE IF NOT EXISTS studio_designs (
  id           TEXT PRIMARY KEY,
  name         TEXT NOT NULL,
  revision     INTEGER NOT NULL DEFAULT 1,
  frame_count  INTEGER NOT NULL DEFAULT 0,
  frames_json  TEXT NOT NULL DEFAULT '[]',
  theme_json   TEXT NOT NULL DEFAULT '{}',
  options_json TEXT NOT NULL DEFAULT '{}',
  scenario     TEXT,
  page         TEXT,
  created_at   TEXT NOT NULL,
  updated_at   TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS studio_designs_updated_at
  ON studio_designs (updated_at DESC);

-- studio_designs_binding is created by Migrate after the scenario/page
-- columns exist: an old database has the table but not the columns, so
-- CREATE INDEX here would fail before Migrate could add them.

-- Design-scoped uploads. Bytes live in SQLite because every payload is an
-- image bounded at 8 MiB; there are no filesystem paths to port.
CREATE TABLE IF NOT EXISTS studio_design_assets (
  id          TEXT PRIMARY KEY,
  design_id   TEXT NOT NULL,
  name        TEXT NOT NULL,
  mime        TEXT NOT NULL,
  bytes       INTEGER NOT NULL DEFAULT 0,
  width       INTEGER NOT NULL DEFAULT 0,
  height      INTEGER NOT NULL DEFAULT 0,
  fidelity    TEXT NOT NULL DEFAULT 'reference',
  data        BLOB NOT NULL,
  created_at  TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS studio_design_assets_design
  ON studio_design_assets (design_id, created_at);
