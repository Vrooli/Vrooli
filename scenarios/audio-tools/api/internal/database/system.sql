-- System schema for audio-tools.
--
-- All durable tables live here (single-tenant SQLite). Per-domain
-- handlers keep an empty Schema() function and read/write via
-- internal/store/*. Colocating schema in one auditable file keeps
-- EnsureSchemas idempotent regardless of registration order.

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

CREATE TABLE IF NOT EXISTS provider_config (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  byok_enabled   INTEGER NOT NULL DEFAULT 1,
  vrooli_enabled INTEGER NOT NULL DEFAULT 0,
  local_enabled  INTEGER NOT NULL DEFAULT 1,
  whisper_url    TEXT NOT NULL DEFAULT '',
  kokoro_url     TEXT NOT NULL DEFAULT '',
  ollama_url     TEXT NOT NULL DEFAULT '',
  lpbs_base_url  TEXT NOT NULL DEFAULT '',
  lpbs_app_bundle_key TEXT NOT NULL DEFAULT '',
  avail_ttl_byok_seconds   INTEGER NOT NULL DEFAULT 300,
  avail_ttl_vrooli_seconds INTEGER NOT NULL DEFAULT 30,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS voice_overrides (
  canonical_voice TEXT NOT NULL,
  tier_provider   TEXT NOT NULL,
  adapter_voice   TEXT NOT NULL,
  PRIMARY KEY (canonical_voice, tier_provider)
);

CREATE TABLE IF NOT EXISTS usage_rows (
  operation_id   TEXT PRIMARY KEY,
  emitted_at     TEXT NOT NULL,
  capability     TEXT NOT NULL,
  operation      TEXT NOT NULL,
  provider_tier  TEXT NOT NULL,
  provider_id    TEXT NOT NULL,
  model_id       TEXT NOT NULL DEFAULT '',
  latency_ms     REAL NOT NULL,
  credits_charged INTEGER NOT NULL DEFAULT 0,
  prompt_tokens  INTEGER NOT NULL DEFAULT 0,
  output_tokens  INTEGER NOT NULL DEFAULT 0,
  audio_duration_seconds REAL NOT NULL DEFAULT 0,
  error          TEXT NOT NULL DEFAULT '',
  fallback_reason TEXT NOT NULL DEFAULT '',
  user_identity  TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_usage_emitted_at ON usage_rows(emitted_at DESC);
CREATE INDEX IF NOT EXISTS idx_usage_capability ON usage_rows(capability, emitted_at DESC);

CREATE TABLE IF NOT EXISTS wakeword_templates (
  id TEXT PRIMARY KEY,
  phrase TEXT NOT NULL,
  embedding BLOB,
  created_at TEXT NOT NULL
);

-- speaker_profiles holds profile metadata + the active-binding link only.
-- The speaker-verification RESOURCE owns the real ECAPA embedding (keyed by
-- profile id) and performs verification, so `embedding` is nullable here: the
-- embedding bytes never round-trip back to audio-tools.
CREATE TABLE IF NOT EXISTS speaker_profiles (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  embedding BLOB,
  bound_user_identity TEXT,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS stt_stream_config (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  config_json TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tts_config (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  config_json TEXT NOT NULL,
  summarize_json TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS playback_events (
  event_id TEXT PRIMARY KEY,
  emitted_at TEXT NOT NULL,
  kind TEXT NOT NULL,
  version INTEGER NOT NULL DEFAULT 0,
  voice TEXT,
  provider_tier TEXT,
  provider_id TEXT
);
