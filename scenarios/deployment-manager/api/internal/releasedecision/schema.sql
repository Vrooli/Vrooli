CREATE TABLE IF NOT EXISTS release_decision_asks (
    id TEXT PRIMARY KEY,
    subject TEXT NOT NULL,
    state TEXT NOT NULL,
    raised_at TIMESTAMP NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    event_id TEXT NOT NULL DEFAULT '',
    published INTEGER NOT NULL DEFAULT 0,
    resolved_at TIMESTAMP,
    resolved_by TEXT,
    extra TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX IF NOT EXISTS idx_release_decision_asks_subject
    ON release_decision_asks (subject, state, expires_at);
