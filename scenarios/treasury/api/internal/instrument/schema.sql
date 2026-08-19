CREATE TABLE IF NOT EXISTS instruments (
    id TEXT PRIMARY KEY,
    book_id TEXT NOT NULL REFERENCES books(id),
    mandate_id TEXT NOT NULL REFERENCES mandates(id),
    rail TEXT NOT NULL CHECK (length(trim(rail)) > 0),
    credential_reference TEXT NOT NULL CHECK (length(trim(credential_reference)) > 0),
    cap_minor INTEGER NOT NULL CHECK (cap_minor > 0),
    currency TEXT NOT NULL CHECK (length(trim(currency)) > 0),
    counterparty TEXT NOT NULL CHECK (length(trim(counterparty)) > 0),
    expires_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);
