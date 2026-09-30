CREATE TABLE IF NOT EXISTS portability_restore_checkpoints (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  expected_revision INTEGER NOT NULL,
  previous_state_json TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS portability_restore_operations (
  workspace_id TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  request_hash TEXT NOT NULL,
  workspace_revision INTEGER NOT NULL,
  checkpoint_id TEXT NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY(workspace_id, idempotency_key),
  FOREIGN KEY(checkpoint_id) REFERENCES portability_restore_checkpoints(id)
);
CREATE TABLE IF NOT EXISTS portability_recipe_import_operations (
  workspace_id TEXT NOT NULL,
  idempotency_key TEXT NOT NULL,
  request_hash TEXT NOT NULL,
  workspace_revision INTEGER NOT NULL,
  recipes_applied INTEGER NOT NULL,
  recipes_skipped INTEGER NOT NULL,
  remapped_ids_json TEXT NOT NULL DEFAULT '[]',
  created_at TEXT NOT NULL,
  PRIMARY KEY(workspace_id, idempotency_key)
);
