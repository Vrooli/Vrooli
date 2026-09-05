CREATE TABLE IF NOT EXISTS token_types (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL,
    symbol TEXT NOT NULL,
    color TEXT NOT NULL,
    supply_policy TEXT NOT NULL CHECK (supply_policy IN ('unbounded', 'capped', 'fixed')),
    cap_amount INTEGER NOT NULL DEFAULT 0 CHECK (cap_amount >= 0),
    minted_amount INTEGER NOT NULL DEFAULT 0 CHECK (minted_amount >= 0),
    retired INTEGER NOT NULL DEFAULT 0 CHECK (retired IN (0, 1)),
    created_at TEXT NOT NULL,
    retired_at TEXT,
    CHECK (
        (supply_policy = 'unbounded' AND cap_amount = 0) OR
        (supply_policy IN ('capped', 'fixed') AND cap_amount > 0)
    ),
    CHECK (supply_policy = 'unbounded' OR minted_amount <= cap_amount)
);

CREATE TABLE IF NOT EXISTS minter_authorities (
    token_type_id TEXT PRIMARY KEY REFERENCES token_types(id),
    subject TEXT NOT NULL CHECK (length(trim(subject)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_token_types_active_created
    ON token_types(retired, created_at, id);
