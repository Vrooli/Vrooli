CREATE TABLE IF NOT EXISTS shopping_checks (
  workspace_id TEXT NOT NULL,
  line_key TEXT NOT NULL,
  checked INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL,
  PRIMARY KEY(workspace_id, line_key)
);
