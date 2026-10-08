CREATE TABLE IF NOT EXISTS switchboard_bindings (
 id TEXT PRIMARY KEY,
 agent_id TEXT NOT NULL,
 channel_id TEXT NOT NULL,
 address TEXT NOT NULL,
 thread_key TEXT NOT NULL DEFAULT '',
 created_at TEXT NOT NULL,
 UNIQUE(agent_id, channel_id, address, thread_key)
);
CREATE INDEX IF NOT EXISTS switchboard_bindings_lookup
 ON switchboard_bindings(channel_id, thread_key, address);
