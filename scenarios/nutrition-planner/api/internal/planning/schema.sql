CREATE TABLE IF NOT EXISTS plans (
  workspace_id TEXT PRIMARY KEY,
  revision INTEGER NOT NULL,
  plan_json TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
