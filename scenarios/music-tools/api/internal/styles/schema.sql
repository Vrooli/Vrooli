CREATE TABLE IF NOT EXISTS style (
  id TEXT PRIMARY KEY,
  json TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS music_style_defaults (
  scope TEXT PRIMARY KEY,
  style_id TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
