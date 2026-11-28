-- Look / Style Library custom entries (IMG-P1-012) — owned by internal/looks/.
-- Embedded by schema.go and applied via database.EnsureSchemas at boot through
-- the modules.AllSchemas registry.
--
-- Built-in Looks (seed.go) are the read-only baseline merged at read time; this
-- table records ONLY operator-created custom Looks, serialized as protojson of
-- the looks.v1.Look message. Keyed by id; an id colliding with a built-in is
-- rejected by the store so custom entries can only ADD, never shadow a built-in.
-- thumbnail_ref is denormalized into its own column so RenderPreview can update
-- a Look's preview without rewriting the whole JSON blob.
CREATE TABLE IF NOT EXISTS look (
  id            TEXT PRIMARY KEY,
  json          TEXT NOT NULL,
  thumbnail_ref TEXT NOT NULL DEFAULT '',
  created_at    TEXT NOT NULL DEFAULT '',
  updated_at    TEXT NOT NULL DEFAULT ''
);
