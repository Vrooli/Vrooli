-- Agent Inbox SQLite Schema (Greenfield)
-- All tables use TEXT for UUIDs (generated in Go via google/uuid).
-- Timestamps stored as TEXT in ISO-8601 format via datetime('now').

CREATE TABLE IF NOT EXISTS chats (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL DEFAULT 'New Chat',
    preview TEXT NOT NULL DEFAULT '',
    model TEXT NOT NULL DEFAULT 'anthropic/claude-3.5-sonnet',
    system_prompt TEXT DEFAULT '',
    view_mode TEXT NOT NULL DEFAULT 'bubble' CHECK (view_mode IN ('bubble', 'compact')),
    is_read INTEGER NOT NULL DEFAULT 0,
    is_archived INTEGER NOT NULL DEFAULT 0,
    is_starred INTEGER NOT NULL DEFAULT 0,
    tools_enabled INTEGER DEFAULT 1,
    web_search_enabled INTEGER DEFAULT 0,
    active_leaf_message_id TEXT,
    active_template_id TEXT,
    active_template_tool_ids TEXT,
    chat_mode TEXT DEFAULT 'llm',
    agent_run_id TEXT,
    agent_task_id TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS messages (
    id TEXT PRIMARY KEY,
    chat_id TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    model TEXT,
    token_count INTEGER DEFAULT 0,
    tool_call_id TEXT,
    tool_calls TEXT,
    response_id TEXT,
    finish_reason TEXT,
    parent_message_id TEXT REFERENCES messages(id) ON DELETE CASCADE,
    sibling_index INTEGER DEFAULT 0,
    web_search INTEGER,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_messages_chat_id ON messages(chat_id);
CREATE INDEX IF NOT EXISTS idx_messages_created_at ON messages(created_at);
CREATE INDEX IF NOT EXISTS idx_messages_tool_call_id ON messages(tool_call_id) WHERE tool_call_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_messages_parent_id ON messages(parent_message_id);

-- Add FK from chats.active_leaf_message_id now that messages table exists.
-- SQLite does not support ADD CONSTRAINT, but the column was declared without
-- REFERENCES intentionally to avoid a circular-dependency DDL error when
-- foreign_keys pragma is ON.  Application code enforces referential integrity.

CREATE TABLE IF NOT EXISTS labels (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    color TEXT NOT NULL DEFAULT '#6366f1',
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS chat_labels (
    chat_id TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    label_id TEXT NOT NULL REFERENCES labels(id) ON DELETE CASCADE,
    PRIMARY KEY (chat_id, label_id)
);

CREATE INDEX IF NOT EXISTS idx_chat_labels_chat_id ON chat_labels(chat_id);
CREATE INDEX IF NOT EXISTS idx_chat_labels_label_id ON chat_labels(label_id);

CREATE TABLE IF NOT EXISTS tool_calls (
    id TEXT PRIMARY KEY,
    message_id TEXT NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    chat_id TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    tool_name TEXT NOT NULL,
    arguments TEXT NOT NULL DEFAULT '{}',
    result TEXT,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'pending_approval', 'approved', 'rejected', 'running', 'completed', 'failed', 'cancelled')),
    scenario_name TEXT,
    external_run_id TEXT,
    started_at TEXT NOT NULL DEFAULT (datetime('now')),
    completed_at TEXT,
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS idx_tool_calls_message_id ON tool_calls(message_id);
CREATE INDEX IF NOT EXISTS idx_tool_calls_chat_id ON tool_calls(chat_id);
CREATE INDEX IF NOT EXISTS idx_tool_calls_status ON tool_calls(status) WHERE status IN ('pending', 'running');
CREATE INDEX IF NOT EXISTS idx_tool_calls_pending_approval ON tool_calls(chat_id, status) WHERE status = 'pending_approval';

CREATE TABLE IF NOT EXISTS usage_records (
    id TEXT PRIMARY KEY,
    chat_id TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    message_id TEXT REFERENCES messages(id) ON DELETE SET NULL,
    model TEXT NOT NULL,
    prompt_tokens INTEGER NOT NULL DEFAULT 0,
    completion_tokens INTEGER NOT NULL DEFAULT 0,
    total_tokens INTEGER NOT NULL DEFAULT 0,
    prompt_cost REAL NOT NULL DEFAULT 0,
    completion_cost REAL NOT NULL DEFAULT 0,
    total_cost REAL NOT NULL DEFAULT 0,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_usage_records_chat_id ON usage_records(chat_id);
CREATE INDEX IF NOT EXISTS idx_usage_records_created_at ON usage_records(created_at);
CREATE INDEX IF NOT EXISTS idx_usage_records_model ON usage_records(model);

CREATE TABLE IF NOT EXISTS tool_configurations (
    id TEXT PRIMARY KEY,
    chat_id TEXT REFERENCES chats(id) ON DELETE CASCADE,
    scenario TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 1,
    approval_override TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    UNIQUE (chat_id, scenario, tool_name)
);

CREATE INDEX IF NOT EXISTS idx_tool_configurations_chat_id ON tool_configurations(chat_id);
CREATE INDEX IF NOT EXISTS idx_tool_configurations_global ON tool_configurations(scenario, tool_name) WHERE chat_id IS NULL;
CREATE UNIQUE INDEX IF NOT EXISTS idx_tool_configurations_global_unique ON tool_configurations(scenario, tool_name) WHERE chat_id IS NULL;

CREATE TABLE IF NOT EXISTS user_settings (
    id TEXT PRIMARY KEY,
    key TEXT NOT NULL UNIQUE,
    value TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS attachments (
    id TEXT PRIMARY KEY,
    message_id TEXT REFERENCES messages(id) ON DELETE CASCADE,
    file_name TEXT NOT NULL,
    content_type TEXT NOT NULL,
    file_size INTEGER NOT NULL,
    storage_path TEXT NOT NULL UNIQUE,
    width INTEGER,
    height INTEGER,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_attachments_message_id ON attachments(message_id);

CREATE TABLE IF NOT EXISTS async_operations (
    tool_call_id TEXT PRIMARY KEY,
    chat_id TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    tool_name TEXT NOT NULL,
    scenario_name TEXT NOT NULL,
    operation_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'polling', 'completed', 'failed', 'cancelled', 'timeout')),
    progress INTEGER,
    message TEXT,
    phase TEXT,
    result TEXT,
    error TEXT,
    async_behavior TEXT NOT NULL,
    started_at TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at TEXT NOT NULL DEFAULT (datetime('now')),
    completed_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_async_operations_chat_id ON async_operations(chat_id);
CREATE INDEX IF NOT EXISTS idx_async_operations_status ON async_operations(status) WHERE status NOT IN ('completed', 'failed', 'cancelled', 'timeout');

CREATE TABLE IF NOT EXISTS async_completion_events (
    id TEXT PRIMARY KEY,
    chat_id TEXT NOT NULL REFERENCES chats(id) ON DELETE CASCADE,
    tool_call_id TEXT NOT NULL,
    tool_name TEXT NOT NULL,
    status TEXT NOT NULL,
    result TEXT,
    error TEXT,
    created_at TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_completion_events_chat_created ON async_completion_events(chat_id, created_at);

-- =============================================================================
-- FTS5 Full-Text Search
-- =============================================================================

CREATE VIRTUAL TABLE IF NOT EXISTS messages_fts USING fts5(
    content,
    content=messages,
    content_rowid=rowid,
    tokenize='porter unicode61'
);

CREATE VIRTUAL TABLE IF NOT EXISTS chats_fts USING fts5(
    name,
    content=chats,
    content_rowid=rowid,
    tokenize='porter unicode61'
);

-- Messages FTS triggers
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

-- Chats FTS triggers
CREATE TRIGGER IF NOT EXISTS chats_fts_insert AFTER INSERT ON chats BEGIN
    INSERT INTO chats_fts(rowid, name) VALUES (NEW.rowid, NEW.name);
END;

CREATE TRIGGER IF NOT EXISTS chats_fts_delete AFTER DELETE ON chats BEGIN
    INSERT INTO chats_fts(chats_fts, rowid, name) VALUES('delete', OLD.rowid, OLD.name);
END;

CREATE TRIGGER IF NOT EXISTS chats_fts_update AFTER UPDATE OF name ON chats BEGIN
    INSERT INTO chats_fts(chats_fts, rowid, name) VALUES('delete', OLD.rowid, OLD.name);
    INSERT INTO chats_fts(rowid, name) VALUES (NEW.rowid, NEW.name);
END;
