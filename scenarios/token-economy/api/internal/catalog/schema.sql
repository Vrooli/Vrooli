CREATE TABLE IF NOT EXISTS catalog_entries (
    id TEXT PRIMARY KEY,
    token_type_id TEXT NOT NULL REFERENCES token_types(id),
    title TEXT NOT NULL CHECK (length(trim(title)) > 0),
    description TEXT NOT NULL DEFAULT '',
    cost_amount INTEGER NOT NULL CHECK (cost_amount > 0),
    available_from TEXT,
    available_until TEXT,
    remaining_quantity INTEGER CHECK (remaining_quantity IS NULL OR remaining_quantity >= 0),
    approval_posture TEXT NOT NULL CHECK (approval_posture IN ('immediate', 'requires_approval')),
    retired INTEGER NOT NULL DEFAULT 0 CHECK (retired IN (0, 1)),
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    retired_at TEXT
);

CREATE INDEX IF NOT EXISTS idx_catalog_entries_available
    ON catalog_entries(retired, available_from, available_until, title);

CREATE TABLE IF NOT EXISTS catalog_mutations (
    idempotency_key TEXT PRIMARY KEY,
    entry_id TEXT NOT NULL REFERENCES catalog_entries(id),
    operation TEXT NOT NULL CHECK (operation IN ('create', 'update', 'retire')),
    created_at TEXT NOT NULL
);
