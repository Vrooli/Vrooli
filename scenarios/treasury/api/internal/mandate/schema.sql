CREATE TABLE IF NOT EXISTS mandates (
    id TEXT PRIMARY KEY,
    idempotency_key TEXT NOT NULL UNIQUE,
    book_id TEXT NOT NULL REFERENCES books(id),
    budget_id TEXT NOT NULL REFERENCES budgets(id),
    authorizer TEXT NOT NULL CHECK (length(trim(authorizer)) > 0),
    cap_minor INTEGER NOT NULL CHECK (cap_minor > 0),
    currency TEXT NOT NULL CHECK (length(trim(currency)) > 0),
    allowed_counterparties_json TEXT NOT NULL,
    denied_counterparties_json TEXT NOT NULL,
    required_evidence_json TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    issued_at TEXT NOT NULL,
    signature BLOB NOT NULL CHECK (length(signature) > 0),
    status TEXT NOT NULL CHECK (status IN ('live', 'exhausted', 'expired', 'revoked'))
);
