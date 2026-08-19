CREATE TABLE IF NOT EXISTS approval_requests (
    id TEXT PRIMARY KEY,
    authorization_id TEXT NOT NULL UNIQUE REFERENCES authorizations(id),
    mandate_id TEXT NOT NULL,
    requesting_agent TEXT NOT NULL,
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    currency TEXT NOT NULL,
    counterparty TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued','approved','declined','expired')),
    resolver_identity TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    resolved_at TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS approval_relay_attempts (
    id TEXT PRIMARY KEY,
    approval_id TEXT NOT NULL REFERENCES approval_requests(id),
    outcome TEXT NOT NULL CHECK (outcome IN ('skipped','sent','failed')),
    error TEXT NOT NULL DEFAULT '',
    attempted_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_approval_queue ON approval_requests(status, created_at);
CREATE INDEX IF NOT EXISTS idx_approval_relay_attempts ON approval_relay_attempts(approval_id, attempted_at);
