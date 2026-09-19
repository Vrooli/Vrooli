CREATE TABLE IF NOT EXISTS profiles (
  workspace_id TEXT PRIMARY KEY, revision INTEGER NOT NULL, preset TEXT NOT NULL,
  preset_version INTEGER NOT NULL, active_rules_json TEXT NOT NULL,
  excluded_groups_json TEXT NOT NULL, allergies_json TEXT NOT NULL,
  appliances_json TEXT NOT NULL, cost_weight TEXT NOT NULL,
  effort_weight TEXT NOT NULL, variety_weight TEXT NOT NULL,
  draft_json TEXT NOT NULL, updated_at TEXT NOT NULL
);
