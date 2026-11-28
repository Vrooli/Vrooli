CREATE TABLE IF NOT EXISTS briefs (
  id TEXT PRIMARY KEY,
  consumer TEXT NOT NULL,
  verdict TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  chat_id TEXT,
  message_id TEXT,
  harness TEXT NOT NULL DEFAULT '',
  session_ref TEXT NOT NULL DEFAULT '',
  prompt_digest TEXT NOT NULL,
  effective_query TEXT NOT NULL DEFAULT '',
  rendered TEXT NOT NULL DEFAULT '',
  queried_providers_json TEXT NOT NULL DEFAULT '[]',
  max_trust_class TEXT NOT NULL DEFAULT '',
  degraded INTEGER NOT NULL DEFAULT 0,
  latency_ms INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS brief_items (
  brief_id TEXT NOT NULL REFERENCES briefs(id) ON DELETE CASCADE,
  item_index INTEGER NOT NULL,
  provider_id TEXT NOT NULL,
  type TEXT NOT NULL,
  title TEXT NOT NULL DEFAULT '',
  snippet TEXT NOT NULL DEFAULT '',
  path TEXT NOT NULL DEFAULT '',
  score REAL NOT NULL DEFAULT 0,
  rerank_score REAL NOT NULL DEFAULT 0,
  trust_class TEXT NOT NULL DEFAULT '',
  suggested_command TEXT NOT NULL DEFAULT '',
  PRIMARY KEY (brief_id, item_index)
);

CREATE TABLE IF NOT EXISTS brief_uses (
  brief_id TEXT NOT NULL REFERENCES briefs(id) ON DELETE CASCADE,
  item_index INTEGER NOT NULL,
  kind TEXT NOT NULL,
  occurred_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_briefs_consumer_created ON briefs(consumer, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_briefs_session ON briefs(session_ref, created_at DESC);
