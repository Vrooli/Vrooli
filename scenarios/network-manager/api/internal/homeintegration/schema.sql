CREATE TABLE IF NOT EXISTS home_action_invocations (
    id TEXT PRIMARY KEY,
    action_name TEXT NOT NULL,
    status TEXT NOT NULL,
    approved INTEGER NOT NULL,
    message TEXT NOT NULL,
    params_json TEXT NOT NULL,
    event_id TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_home_action_invocations_action
ON home_action_invocations(action_name, created_at DESC);

CREATE TABLE IF NOT EXISTS home_events (
    id TEXT PRIMARY KEY,
    type TEXT NOT NULL,
    summary TEXT NOT NULL,
    occurred_at TEXT NOT NULL,
    publish_status TEXT NOT NULL,
    publish_error TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_home_events_recent
ON home_events(occurred_at DESC);
