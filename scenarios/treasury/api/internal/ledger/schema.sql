CREATE TABLE IF NOT EXISTS ledger_emissions (
    id TEXT PRIMARY KEY,
    settlement_id TEXT NOT NULL UNIQUE,
    external_id TEXT NOT NULL,
    adapter_id TEXT NOT NULL,
    account_id TEXT NOT NULL,
    book_id TEXT NOT NULL,
    amount_minor INTEGER NOT NULL CHECK (amount_minor != 0),
    currency TEXT NOT NULL,
    basis TEXT NOT NULL,
    occurred_at TEXT NOT NULL,
    fetched_at TEXT NOT NULL,
    description TEXT NOT NULL,
    status TEXT NOT NULL CHECK (status IN ('queued','accepted')),
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    accepted_at TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_ledger_emissions_pending ON ledger_emissions(status, created_at, id);
