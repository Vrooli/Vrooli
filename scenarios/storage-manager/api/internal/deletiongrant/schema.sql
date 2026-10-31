-- Deletion-grant asks. One row is one operator decision about whether an agent
-- may delete a set of paths. It doubles as the operatordecision PendingStore
-- (the Raiser reads/writes it) and as the source the policy snapshot projects
-- approved, unexpired grants from. There is no separate "grant" table: an
-- approved, unexpired ask IS the grant.
CREATE TABLE IF NOT EXISTS deletion_asks (
    id            TEXT PRIMARY KEY,          -- correlation id (stable per path set)
    subject       TEXT NOT NULL,             -- flood-cap subject
    state         TEXT NOT NULL,             -- operatordecision pending state, or 'resolved'
    raised_at     TEXT NOT NULL,
    expires_at    TEXT NOT NULL,             -- the ASK's outstanding lifetime (flood/dedupe)
    event_id      TEXT NOT NULL DEFAULT '',
    published     INTEGER NOT NULL DEFAULT 0,
    resolved_at   TEXT,
    resolved_by   TEXT NOT NULL DEFAULT '',  -- the approving human on resolution
    extra         TEXT NOT NULL DEFAULT '{}' -- JSON: paths, answer, grant_expires_at, requested_by
);

CREATE INDEX IF NOT EXISTS idx_deletion_asks_subject ON deletion_asks(subject, state, expires_at);
CREATE INDEX IF NOT EXISTS idx_deletion_asks_state ON deletion_asks(state);
