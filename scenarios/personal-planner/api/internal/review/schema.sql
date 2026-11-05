CREATE TABLE IF NOT EXISTS review_reflections (
  local_date TEXT PRIMARY KEY,
  text TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS review_wins (
  local_date TEXT PRIMARY KEY,
  text TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS estimation_bias (
  period TEXT PRIMARY KEY,
  avg_error_percent REAL NOT NULL DEFAULT 0,
  over_ratio REAL NOT NULL DEFAULT 0,
  under_ratio REAL NOT NULL DEFAULT 0,
  by_category_json TEXT NOT NULL DEFAULT '{}',
  accuracy_trend_json TEXT NOT NULL DEFAULT '[]',
  sample_size INTEGER NOT NULL DEFAULT 0,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS reminder_preferences (
  id TEXT PRIMARY KEY,
  enabled INTEGER NOT NULL DEFAULT 1,
  quiet_start_minutes INTEGER NOT NULL DEFAULT 1320,
  quiet_end_minutes INTEGER NOT NULL DEFAULT 420,
  lead_minutes INTEGER NOT NULL DEFAULT 60,
  updated_at TEXT NOT NULL
);
INSERT OR IGNORE INTO reminder_preferences(id, updated_at) VALUES ('workspace', datetime('now'));
