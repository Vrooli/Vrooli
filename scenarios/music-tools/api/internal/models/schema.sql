CREATE TABLE IF NOT EXISTS model_state (
  model_id TEXT PRIMARY KEY,
  enabled INTEGER NOT NULL DEFAULT 1,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS model_install (
  model_id TEXT PRIMARY KEY,
  installed_at TEXT NOT NULL,
  checksum TEXT NOT NULL,
  bytes INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS model_custom (
  model_id TEXT PRIMARY KEY,
  model_json TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS model_operation_default (
  operation TEXT PRIMARY KEY,
  model_id TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
