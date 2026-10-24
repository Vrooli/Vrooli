CREATE TABLE IF NOT EXISTS settlements (
    id TEXT PRIMARY KEY,
    authorization_id TEXT NOT NULL REFERENCES authorizations(id),
    mandate_id TEXT NOT NULL REFERENCES mandates(id),
    instrument_id TEXT NOT NULL REFERENCES instruments(id),
    rail TEXT NOT NULL CHECK (length(trim(rail)) > 0),
    idempotency_key TEXT NOT NULL UNIQUE,
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    currency TEXT NOT NULL CHECK (length(trim(currency)) > 0),
    counterparty TEXT NOT NULL CHECK (length(trim(counterparty)) > 0),
    outcome TEXT NOT NULL CHECK (outcome IN ('ready', 'calling', 'settled', 'failed', 'unknown')),
    external_id TEXT NOT NULL,
    receipt_reference TEXT NOT NULL,
    basis TEXT NOT NULL,
    detail TEXT NOT NULL,
    occurred_at TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    retain_until TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS settlements_authorization
ON settlements(authorization_id, created_at);

CREATE INDEX IF NOT EXISTS settlements_retention
ON settlements(retain_until);
