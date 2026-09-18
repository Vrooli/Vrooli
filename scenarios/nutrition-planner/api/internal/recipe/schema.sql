CREATE TABLE IF NOT EXISTS recipes (
  id TEXT PRIMARY KEY, workspace_id TEXT NOT NULL, current_revision INTEGER NOT NULL,
  created_at TEXT NOT NULL, updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS recipe_revisions (
  recipe_id TEXT NOT NULL, revision INTEGER NOT NULL, workspace_id TEXT NOT NULL,
  name TEXT NOT NULL, notes TEXT NOT NULL DEFAULT '', source_url TEXT NOT NULL DEFAULT '',
  source_type TEXT NOT NULL DEFAULT '', original_text TEXT NOT NULL DEFAULT '',
  status TEXT NOT NULL, methods_json TEXT NOT NULL DEFAULT '[]', groups_json TEXT NOT NULL DEFAULT '[]', required_appliances_json TEXT NOT NULL DEFAULT '[]', allergen_evidence_json TEXT NOT NULL DEFAULT '{}', created_at TEXT NOT NULL,
  PRIMARY KEY(recipe_id, revision), FOREIGN KEY(recipe_id) REFERENCES recipes(id)
);
CREATE INDEX IF NOT EXISTS idx_recipes_workspace ON recipes(workspace_id, updated_at DESC);
CREATE TABLE IF NOT EXISTS recipe_idempotency (
  workspace_id TEXT NOT NULL, idempotency_key TEXT NOT NULL, request_hash TEXT NOT NULL,
  recipe_id TEXT NOT NULL, created_at TEXT NOT NULL,
  PRIMARY KEY(workspace_id, idempotency_key), FOREIGN KEY(recipe_id) REFERENCES recipes(id)
);
