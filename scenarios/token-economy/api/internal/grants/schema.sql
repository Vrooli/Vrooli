CREATE TABLE IF NOT EXISTS grants (
    id TEXT PRIMARY KEY,
    token_type_id TEXT NOT NULL REFERENCES token_types(id),
    grant_source_id TEXT NOT NULL,
    authorizer TEXT NOT NULL,
    holder_id TEXT NOT NULL,
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    allowed_catalog_scopes TEXT NOT NULL,
    denied_catalog_scopes TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    issued_at TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('draft', 'live', 'exhausted', 'expired', 'revoked')),
    idempotency_key TEXT NOT NULL UNIQUE,
    required_evidence TEXT NOT NULL,
    recurrence_seconds INTEGER NOT NULL DEFAULT 0 CHECK (recurrence_seconds >= 0),
    next_issue_at TEXT,
    cancelled_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_grants_holder_type
    ON grants(holder_id, token_type_id, issued_at, id);

CREATE TABLE IF NOT EXISTS grant_rules (
    id TEXT PRIMARY KEY,
    grant_id TEXT NOT NULL REFERENCES grants(id),
    position INTEGER NOT NULL CHECK (position >= 0),
    condition TEXT NOT NULL CHECK (condition IN ('catalog_scope_allowed', 'catalog_scope_denied', 'before_expiry', 'required_evidence', 'sufficient_balance')),
    operands TEXT NOT NULL,
    amount_limit INTEGER NOT NULL DEFAULT 0 CHECK (amount_limit >= 0),
    UNIQUE (grant_id, position)
);

CREATE TABLE IF NOT EXISTS grant_schedules (
    grant_id TEXT PRIMARY KEY REFERENCES grants(id),
    recurrence_seconds INTEGER NOT NULL CHECK (recurrence_seconds > 0),
    next_issue_at TEXT NOT NULL,
    cancelled_at TEXT
);

CREATE TABLE IF NOT EXISTS grant_revoke_receipts (
    idempotency_key TEXT PRIMARY KEY CHECK (length(trim(idempotency_key)) > 0),
    grant_id TEXT NOT NULL UNIQUE REFERENCES grants(id),
    reason TEXT NOT NULL CHECK (length(trim(reason)) > 0),
    created_at TEXT NOT NULL
);
