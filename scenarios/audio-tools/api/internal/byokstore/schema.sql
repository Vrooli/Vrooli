CREATE TABLE IF NOT EXISTS byok_credentials (
  provider_id   TEXT NOT NULL,
  capability    TEXT NOT NULL CHECK (capability IN ('stt','tts','summarize')),
  secret_kind   TEXT NOT NULL DEFAULT 'api_key',
  secret_cipher BLOB NOT NULL,
  fingerprint   TEXT NOT NULL,
  created_at    TEXT NOT NULL,
  last_used_at  TEXT,
  PRIMARY KEY (provider_id, capability)
);
