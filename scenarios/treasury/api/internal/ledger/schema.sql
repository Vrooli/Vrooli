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
    accepted_at TEXT NOT NULL DEFAULT '',
    posting_id TEXT NOT NULL DEFAULT '',
    receipt_id TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_ledger_emissions_pending ON ledger_emissions(status, created_at, id);

-- The operator's chosen Money Ledger destination for settlements: at most one
-- row. Treasury never guesses or creates it, and settlements stay queued until
-- it exists.
CREATE TABLE IF NOT EXISTS ledger_destination (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    book_id TEXT NOT NULL,
    account_id TEXT NOT NULL,
    book_name TEXT NOT NULL,
    account_name TEXT NOT NULL,
    currency TEXT NOT NULL,
    set_at TEXT NOT NULL
);
