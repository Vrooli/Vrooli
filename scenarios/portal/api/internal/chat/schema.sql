-- Portal chat domain schema. Applied declaratively through
-- database.EnsureSchemas at boot; no migration ladder lives in this scenario.
-- Times are RFC3339Nano strings written by Go so wire and storage formats
-- round-trip cleanly.

CREATE TABLE IF NOT EXISTS chat_groups (
  id         TEXT PRIMARY KEY,
  name       TEXT NOT NULL,
  color      TEXT NOT NULL DEFAULT '#2563eb',
  collapsed  INTEGER NOT NULL DEFAULT 0,
  sort_order INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_chat_groups_sort ON chat_groups(sort_order, name);

CREATE TABLE IF NOT EXISTS chats (
  id                     TEXT PRIMARY KEY,
  title                  TEXT NOT NULL DEFAULT 'New chat',
  preview                TEXT NOT NULL DEFAULT '',
  group_id               TEXT REFERENCES chat_groups(id) ON DELETE SET NULL,
  sort_order             INTEGER NOT NULL DEFAULT 0,
  model                  TEXT NOT NULL DEFAULT 'anthropic/claude-3.5-sonnet',
  web_search_enabled     INTEGER NOT NULL DEFAULT 0,
  mode                   TEXT NOT NULL DEFAULT 'llm' CHECK (mode IN ('llm', 'agent')),
  agent_harness          TEXT NOT NULL DEFAULT 'claude-code' CHECK (agent_harness IN ('claude-code', 'codex', 'opencode', 'grok')),
  active_leaf_message_id TEXT,
  system_prompt          TEXT NOT NULL DEFAULT '',
  created_at             TEXT NOT NULL,
  updated_at             TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_chats_group_sort ON chats(group_id, sort_order, updated_at);
CREATE INDEX IF NOT EXISTS idx_chats_updated ON chats(updated_at);

CREATE TABLE IF NOT EXISTS messages (
  id                TEXT PRIMARY KEY,
  chat_id           TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
  parent_message_id TEXT REFERENCES messages(id) ON DELETE CASCADE,
  sibling_index     INTEGER NOT NULL DEFAULT 0,
  role              TEXT NOT NULL CHECK (role IN ('system', 'user', 'assistant', 'agent')),
  content           TEXT NOT NULL,
  model             TEXT NOT NULL DEFAULT '',
  token_count       INTEGER NOT NULL DEFAULT 0,
  response_id       TEXT NOT NULL DEFAULT '',
  finish_reason     TEXT NOT NULL DEFAULT '',
  web_search        INTEGER,
  created_at        TEXT NOT NULL,
  updated_at        TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON messages(chat_id);
CREATE INDEX IF NOT EXISTS idx_messages_parent ON messages(parent_message_id, sibling_index);
CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at);

CREATE TABLE IF NOT EXISTS usage_records (
  id                TEXT PRIMARY KEY,
  chat_id           TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
  message_id        TEXT REFERENCES messages(id) ON DELETE SET NULL,
  provider          TEXT NOT NULL DEFAULT 'openrouter',
  model             TEXT NOT NULL,
  prompt_tokens     INTEGER NOT NULL DEFAULT 0,
  completion_tokens INTEGER NOT NULL DEFAULT 0,
  total_tokens      INTEGER NOT NULL DEFAULT 0,
  cost_usd          REAL NOT NULL DEFAULT 0,
  created_at        TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_usage_records_chat_id ON usage_records(chat_id);
CREATE INDEX IF NOT EXISTS idx_usage_records_message_id ON usage_records(message_id);

CREATE TABLE IF NOT EXISTS search_attachments (
  id         TEXT PRIMARY KEY,
  chat_id    TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
  message_id TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
  query      TEXT NOT NULL,
  hits_json  TEXT NOT NULL DEFAULT '[]',
  degraded   INTEGER NOT NULL DEFAULT 0,
  reason     TEXT NOT NULL DEFAULT '',
  latency_ms INTEGER NOT NULL DEFAULT 0,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_search_attachments_message ON search_attachments(message_id, created_at);
CREATE INDEX IF NOT EXISTS idx_search_attachments_chat ON search_attachments(chat_id, created_at);

CREATE TABLE IF NOT EXISTS user_settings (
  id         TEXT PRIMARY KEY,
  key        TEXT NOT NULL UNIQUE,
  value      TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
  content,
  content=messages,
  content_rowid=rowid,
  tokenize='porter unicode61'
);

CREATE VIRTUAL TABLE IF NOT EXISTS chats_fts USING fts5(
  title,
  preview,
  content=chats,
  content_rowid=rowid,
  tokenize='porter unicode61'
);

CREATE TRIGGER IF NOT EXISTS messages_fts_insert AFTER INSERT ON messages BEGIN
  INSERT INTO messages_fts(rowid, content) VALUES (NEW.rowid, NEW.content);
END;

CREATE TRIGGER IF NOT EXISTS messages_fts_delete AFTER DELETE ON messages BEGIN
  INSERT INTO messages_fts(messages_fts, rowid, content) VALUES('delete', OLD.rowid, OLD.content);
END;

CREATE TRIGGER IF NOT EXISTS messages_fts_update AFTER UPDATE OF content ON messages BEGIN
  INSERT INTO messages_fts(messages_fts, rowid, content) VALUES('delete', OLD.rowid, OLD.content);
  INSERT INTO messages_fts(rowid, content) VALUES (NEW.rowid, NEW.content);
END;

CREATE TRIGGER IF NOT EXISTS chats_fts_insert AFTER INSERT ON chats BEGIN
  INSERT INTO chats_fts(rowid, title, preview) VALUES (NEW.rowid, NEW.title, NEW.preview);
END;

CREATE TRIGGER IF NOT EXISTS chats_fts_delete AFTER DELETE ON chats BEGIN
  INSERT INTO chats_fts(chats_fts, rowid, title, preview) VALUES('delete', OLD.rowid, OLD.title, OLD.preview);
END;

CREATE TRIGGER IF NOT EXISTS chats_fts_update AFTER UPDATE OF title, preview ON chats BEGIN
  INSERT INTO chats_fts(chats_fts, rowid, title, preview) VALUES('delete', OLD.rowid, OLD.title, OLD.preview);
  INSERT INTO chats_fts(rowid, title, preview) VALUES (NEW.rowid, NEW.title, NEW.preview);
END;
