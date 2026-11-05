CREATE TABLE IF NOT EXISTS holders (
    id TEXT PRIMARY KEY,
    display_name TEXT NOT NULL CHECK (length(trim(display_name)) > 0),
    authenticator_subject TEXT NOT NULL UNIQUE CHECK (length(trim(authenticator_subject)) > 0),
    created_at TEXT NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_holders_authenticator_subject
    ON holders(authenticator_subject);

CREATE TABLE IF NOT EXISTS holder_create_receipts (
    idempotency_key TEXT PRIMARY KEY CHECK (length(trim(idempotency_key)) > 0),
    holder_id TEXT NOT NULL UNIQUE REFERENCES holders(id),
    created_at TEXT NOT NULL
);
