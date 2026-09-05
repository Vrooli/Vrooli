CREATE TABLE IF NOT EXISTS provider_config (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  byok_enabled INTEGER NOT NULL DEFAULT 1,
  vrooli_enabled INTEGER NOT NULL DEFAULT 0,
  local_enabled INTEGER NOT NULL DEFAULT 1,
  whisper_url TEXT NOT NULL DEFAULT '',
  kokoro_url TEXT NOT NULL DEFAULT '',
  ollama_url TEXT NOT NULL DEFAULT '',
  lpbs_base_url TEXT NOT NULL DEFAULT '',
  lpbs_app_bundle_key TEXT NOT NULL DEFAULT '',
  avail_ttl_byok_seconds INTEGER NOT NULL DEFAULT 300,
  avail_ttl_vrooli_seconds INTEGER NOT NULL DEFAULT 30,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS voice_overrides (
  canonical_voice TEXT NOT NULL,
  tier_provider TEXT NOT NULL,
  adapter_voice TEXT NOT NULL,
  PRIMARY KEY (canonical_voice, tier_provider)
);
