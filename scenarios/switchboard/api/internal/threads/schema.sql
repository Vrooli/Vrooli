CREATE TABLE IF NOT EXISTS switchboard_threads (
 id TEXT PRIMARY KEY, channel_id TEXT NOT NULL, thread_key TEXT NOT NULL,
 is_group INTEGER NOT NULL DEFAULT 0, position INTEGER NOT NULL DEFAULT 0,
 turn_budget INTEGER NOT NULL DEFAULT 20, spend_cap_cents INTEGER NOT NULL DEFAULT 0,
 created_at TEXT NOT NULL, updated_at TEXT NOT NULL,
 UNIQUE(channel_id, thread_key)
);
CREATE TABLE IF NOT EXISTS switchboard_messages (
 id INTEGER PRIMARY KEY AUTOINCREMENT, thread_id TEXT NOT NULL REFERENCES switchboard_threads(id),
 channel_id TEXT NOT NULL, remote_id TEXT NOT NULL, author_kind TEXT NOT NULL,
 sender_address TEXT NOT NULL, text TEXT NOT NULL, reply_to_remote_id TEXT NOT NULL DEFAULT '',
 received_at TEXT NOT NULL, media_json TEXT NOT NULL DEFAULT '[]', UNIQUE(channel_id, remote_id)
);
CREATE INDEX IF NOT EXISTS switchboard_messages_thread ON switchboard_messages(thread_id, received_at, id);
CREATE TABLE IF NOT EXISTS switchboard_thread_runs (
 thread_id TEXT PRIMARY KEY REFERENCES switchboard_threads(id),
 run_id TEXT NOT NULL UNIQUE,
 updated_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS switchboard_thread_budget (
 thread_id TEXT PRIMARY KEY REFERENCES switchboard_threads(id),
 window_started_at TEXT NOT NULL,
 used INTEGER NOT NULL DEFAULT 0,
 spent_cents INTEGER NOT NULL DEFAULT 0,
 owner_notified INTEGER NOT NULL DEFAULT 0
);
CREATE TABLE IF NOT EXISTS switchboard_turn_events (
 id TEXT PRIMARY KEY,
 thread_id TEXT NOT NULL,
 agent_id TEXT NOT NULL DEFAULT '',
 channel_id TEXT NOT NULL,
 sender_address TEXT NOT NULL DEFAULT '',
 outcome TEXT NOT NULL,
 reason TEXT NOT NULL DEFAULT '',
 created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS switchboard_turn_events_thread ON switchboard_turn_events(thread_id, created_at);
CREATE INDEX IF NOT EXISTS switchboard_turn_events_agent ON switchboard_turn_events(agent_id, created_at);
