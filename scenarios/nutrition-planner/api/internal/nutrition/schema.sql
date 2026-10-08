CREATE TABLE IF NOT EXISTS nutrition_targets (
  id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  workspace_id TEXT NOT NULL,
  nutrient_id TEXT NOT NULL,
  lower_bound TEXT NOT NULL,
  upper_bound TEXT NOT NULL,
  period TEXT NOT NULL,
  scope TEXT NOT NULL,
  enforcement TEXT NOT NULL,
  provenance TEXT NOT NULL,
  effective_from TEXT NOT NULL,
  effective_to TEXT,
  active INTEGER NOT NULL,
  PRIMARY KEY(id, revision)
);
CREATE INDEX IF NOT EXISTS idx_nutrition_targets_workspace ON nutrition_targets(workspace_id, active, effective_from DESC);
CREATE TABLE IF NOT EXISTS nutrition_intake_events (
  workspace_id TEXT NOT NULL,
  event_id TEXT NOT NULL,
  event_date TEXT NOT NULL,
  recipe_id TEXT NOT NULL DEFAULT '',
  recipe_revision INTEGER NOT NULL DEFAULT 0,
  nutrient_id TEXT NOT NULL,
  amount TEXT NOT NULL,
  unit TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  correction_of TEXT NOT NULL DEFAULT '',
  recorded_at TEXT NOT NULL,
  PRIMARY KEY(workspace_id, event_id)
);
CREATE INDEX IF NOT EXISTS idx_nutrition_intake_workspace_date ON nutrition_intake_events(workspace_id, event_date, recorded_at);
