-- INVARIANT: onlyOperatorBeneficiaryCanBeRepresented
-- Holding value for a second beneficiary would cross into third-party custody,
-- a regulated activity Treasury is deliberately unable to represent.
CREATE TABLE IF NOT EXISTS treasury_beneficiary (
    singleton_key INTEGER PRIMARY KEY CHECK (singleton_key = 1),
    identity TEXT NOT NULL UNIQUE
);

CREATE TABLE IF NOT EXISTS books (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL CHECK (length(trim(name)) > 0),
    beneficiary_identity TEXT NOT NULL REFERENCES treasury_beneficiary(identity),
    created_at TEXT NOT NULL
);
