-- Templates domain schema — owned by internal/templates/.
-- Applied via database.EnsureSchemas at boot through modules.AllSchemas().
-- Idempotent: CREATE TABLE/INDEX IF NOT EXISTS so re-runs are no-ops.
--
-- A template is a shelf draft the studio authors before publishing a library
-- asset; it is not a governed release. palette_json and spec_json are opaque
-- JSON objects: the server never re-models the TemplateNode tree
-- (ui/src/workbench/templates/spec.ts owns that shape). spec_json also carries
-- the two authored copy fields the shelf renders (slotHint, idea) because the
-- table has no columns for them; see types.go for the exact envelope.
CREATE TABLE IF NOT EXISTS studio_templates (
  id            TEXT PRIMARY KEY,
  name          TEXT NOT NULL,
  tagline       TEXT NOT NULL DEFAULT '',
  category      TEXT NOT NULL DEFAULT '',
  palette_json  TEXT NOT NULL DEFAULT '{}',
  display       TEXT NOT NULL DEFAULT 'sans',
  radius        INTEGER NOT NULL DEFAULT 14,
  spec_json     TEXT NOT NULL DEFAULT '{}',
  created_at    TEXT NOT NULL,
  updated_at    TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS studio_templates_updated_at
  ON studio_templates (updated_at DESC);
