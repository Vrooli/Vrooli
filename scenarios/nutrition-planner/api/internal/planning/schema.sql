CREATE TABLE IF NOT EXISTS plan_state (
  workspace_id TEXT PRIMARY KEY,
  revision INTEGER NOT NULL,
  metadata_json TEXT NOT NULL,
  unresolved_json TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS plan_occurrences (
  workspace_id TEXT NOT NULL,
  date TEXT NOT NULL,
  slot_name TEXT NOT NULL,
  mode TEXT NOT NULL,
  quantity TEXT NOT NULL,
  recipe_id TEXT NOT NULL,
  recipe_revision INTEGER NOT NULL DEFAULT 0,
  recipe_name TEXT NOT NULL,
  reason TEXT NOT NULL,
  locked INTEGER NOT NULL DEFAULT 0,
  PRIMARY KEY (workspace_id, date, slot_name),
  FOREIGN KEY (workspace_id) REFERENCES plan_state(workspace_id) ON DELETE CASCADE
);
CREATE INDEX IF NOT EXISTS idx_plan_occurrences_date ON plan_occurrences(workspace_id, date);

CREATE TABLE IF NOT EXISTS plan_apply_receipts (
  workspace_id TEXT NOT NULL,
  expected_revision INTEGER NOT NULL,
  payload_hash TEXT NOT NULL,
  applied_revision INTEGER NOT NULL,
  PRIMARY KEY (workspace_id, expected_revision, payload_hash)
);

CREATE TABLE IF NOT EXISTS plan_migrations (
  name TEXT PRIMARY KEY,
  applied_at TEXT NOT NULL
);
