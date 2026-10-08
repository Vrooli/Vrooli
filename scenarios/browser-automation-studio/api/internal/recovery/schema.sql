-- Browser Automation Studio sidecar checkpoint storage.
-- This SQLite store is private to the recovery sidecar, separate from the
-- scenario's PostgreSQL domain schema.
CREATE TABLE IF NOT EXISTS session_checkpoints (
    id TEXT PRIMARY KEY,
    session_id TEXT UNIQUE NOT NULL,
    workflow_id TEXT,
    actions TEXT NOT NULL DEFAULT '[]',
    current_url TEXT NOT NULL DEFAULT '',
    browser_config TEXT NOT NULL DEFAULT '{}',
    created_at DATETIME NOT NULL,
    updated_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_checkpoints_session ON session_checkpoints(session_id);
CREATE INDEX IF NOT EXISTS idx_checkpoints_updated ON session_checkpoints(updated_at);
