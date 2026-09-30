CREATE TABLE IF NOT EXISTS wakeword_templates (
  id TEXT PRIMARY KEY,
  phrase TEXT NOT NULL,
  embedding BLOB,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS speaker_profiles (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  embedding BLOB,
  bound_user_identity TEXT,
  created_at TEXT NOT NULL,
  clip_count INTEGER NOT NULL DEFAULT 0,
  total_voiced_seconds REAL NOT NULL DEFAULT 0,
  sample_rate INTEGER NOT NULL DEFAULT 0,
  embedding_dim INTEGER NOT NULL DEFAULT 0,
  model_name TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS stt_stream_config (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  config_json TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS stt_speaker_config (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  config_json TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
