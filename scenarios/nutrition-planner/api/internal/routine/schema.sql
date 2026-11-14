CREATE TABLE IF NOT EXISTS routine_templates (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  current_revision INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS routine_template_revisions (
  template_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  workspace_id TEXT NOT NULL,
  slot_name TEXT NOT NULL,
  recipe_id TEXT NOT NULL DEFAULT '',
  quantity TEXT NOT NULL,
  weekdays_json TEXT NOT NULL,
  start_date TEXT NOT NULL,
  end_date TEXT NOT NULL DEFAULT '',
  mode TEXT NOT NULL,
  active INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY(template_id, revision),
  FOREIGN KEY(template_id) REFERENCES routine_templates(id)
);
CREATE INDEX IF NOT EXISTS idx_routine_templates_workspace ON routine_templates(workspace_id, updated_at DESC);
