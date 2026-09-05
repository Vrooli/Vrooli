-- user_settings — owned by internal/settings/. Embedded by schema.go and
-- applied via database.EnsureSchemas at boot through modules.AllSchemas.
--
-- Single-row table keyed by principal_id ('local' in v1; multi-tenant
-- support is explicitly out of scope per the flow-verifier PRD). Every
-- column is NOT NULL with a sensible default so the row can be created
-- by an UPSERT without specifying anything. JSON-shaped columns
-- (inventory_filters) are stored as TEXT and re-validated by the
-- service before write.
CREATE TABLE IF NOT EXISTS user_settings (
  principal_id      TEXT PRIMARY KEY NOT NULL DEFAULT 'local',
  theme             TEXT NOT NULL DEFAULT 'system'
                    CHECK (theme IN ('light','dark','system')),
  font_scale        TEXT NOT NULL DEFAULT 'md'
                    CHECK (font_scale IN ('sm','md','lg')),
  reduced_motion    INTEGER NOT NULL DEFAULT 0
                    CHECK (reduced_motion IN (0,1)),
  rtl               INTEGER NOT NULL DEFAULT 0
                    CHECK (rtl IN (0,1)),
  default_root      TEXT NOT NULL DEFAULT '.',
  density           TEXT NOT NULL DEFAULT 'comfortable'
                    CHECK (density IN ('comfortable','compact')),
  sidebar_width     INTEGER NOT NULL DEFAULT 320,
  inventory_filters TEXT NOT NULL DEFAULT '{"search":"","language":"all","status":[],"sort":{"key":"flowId","dir":"asc"}}',
  updated_at        TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);
