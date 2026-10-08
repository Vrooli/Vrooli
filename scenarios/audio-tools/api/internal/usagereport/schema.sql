CREATE TABLE IF NOT EXISTS usage_rows (
  operation_id TEXT PRIMARY KEY,
  emitted_at TEXT NOT NULL,
  capability TEXT NOT NULL,
  operation TEXT NOT NULL,
  provider_tier TEXT NOT NULL,
  provider_id TEXT NOT NULL,
  model_id TEXT NOT NULL DEFAULT '',
  latency_ms REAL NOT NULL,
  credits_charged INTEGER NOT NULL DEFAULT 0,
  prompt_tokens INTEGER NOT NULL DEFAULT 0,
  output_tokens INTEGER NOT NULL DEFAULT 0,
  audio_duration_seconds REAL NOT NULL DEFAULT 0,
  error TEXT NOT NULL DEFAULT '',
  fallback_reason TEXT NOT NULL DEFAULT '',
  user_identity TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_usage_emitted_at ON usage_rows(emitted_at DESC);
CREATE INDEX IF NOT EXISTS idx_usage_capability ON usage_rows(capability, emitted_at DESC);
