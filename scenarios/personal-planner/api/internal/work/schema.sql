CREATE TABLE IF NOT EXISTS work_items (
  id TEXT PRIMARY KEY,
  title TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  remaining_minutes INTEGER NOT NULL DEFAULT 0,
  source_label TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_work_items_created_at ON work_items(created_at DESC);
