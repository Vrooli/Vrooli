CREATE TABLE IF NOT EXISTS work_items (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  remaining_minutes INTEGER NOT NULL DEFAULT 0,
  original_estimate_minutes INTEGER NOT NULL DEFAULT 0,
  status TEXT NOT NULL DEFAULT 'open',
  completed_at TEXT NOT NULL DEFAULT '',
  snoozed_until TEXT NOT NULL DEFAULT '',
  source_label TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_work_items_created_at ON work_items(created_at DESC);

CREATE TABLE IF NOT EXISTS work_item_estimation_changes (
  id TEXT PRIMARY KEY,
  work_item_id TEXT NOT NULL,
  previous_minutes INTEGER NOT NULL,
  new_minutes INTEGER NOT NULL,
  reason_code TEXT NOT NULL DEFAULT '',
  changed_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_work_item_estimation_changes_item ON work_item_estimation_changes(work_item_id, changed_at DESC);

CREATE TABLE IF NOT EXISTS work_item_completion_history (
  id TEXT PRIMARY KEY,
  work_item_id TEXT NOT NULL,
  original_estimate_minutes INTEGER NOT NULL,
  final_actual_minutes INTEGER NOT NULL,
  variance_minutes INTEGER NOT NULL,
  created_date TEXT NOT NULL,
  completed_date TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_work_item_completion_history_item ON work_item_completion_history(work_item_id, completed_date DESC);

CREATE TABLE IF NOT EXISTS work_item_snoozes (
  id TEXT PRIMARY KEY,
  work_item_id TEXT NOT NULL,
  until_date TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_work_item_snoozes_item ON work_item_snoozes(work_item_id, created_at DESC);
