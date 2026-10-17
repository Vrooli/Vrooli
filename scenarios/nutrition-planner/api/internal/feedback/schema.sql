CREATE TABLE IF NOT EXISTS meal_feedback (
  workspace_id TEXT NOT NULL,
  date TEXT NOT NULL,
  recipe_id TEXT NOT NULL,
  portion TEXT NOT NULL DEFAULT '',
  minutes INTEGER NOT NULL DEFAULT 0,
  recorded_at TEXT NOT NULL,
  PRIMARY KEY(workspace_id, date)
);
