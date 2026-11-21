CREATE TABLE IF NOT EXISTS supplement_schedules (
  id TEXT PRIMARY KEY,
  workspace_id TEXT NOT NULL,
  current_revision INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS supplement_schedule_revisions (
  schedule_id TEXT NOT NULL,
  revision INTEGER NOT NULL,
  workspace_id TEXT NOT NULL,
  product_revision_id TEXT NOT NULL,
  dose TEXT NOT NULL,
  dose_unit TEXT NOT NULL,
  weekdays_json TEXT NOT NULL,
  start_date TEXT NOT NULL,
  end_date TEXT NOT NULL DEFAULT '',
  paused INTEGER NOT NULL,
  confirmed INTEGER NOT NULL,
  created_at TEXT NOT NULL,
  PRIMARY KEY(schedule_id, revision),
  FOREIGN KEY(schedule_id) REFERENCES supplement_schedules(id)
);
CREATE INDEX IF NOT EXISTS idx_supplement_schedules_workspace ON supplement_schedules(workspace_id, updated_at DESC);
