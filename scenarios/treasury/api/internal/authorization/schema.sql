CREATE TABLE IF NOT EXISTS authorizations (
    id TEXT PRIMARY KEY,
    idempotency_key TEXT NOT NULL UNIQUE,
    mandate_id TEXT NOT NULL REFERENCES mandates(id),
    budget_id TEXT NOT NULL REFERENCES budgets(id),
    requesting_agent TEXT NOT NULL CHECK (length(trim(requesting_agent)) > 0),
    amount_minor INTEGER NOT NULL CHECK (amount_minor > 0),
    currency TEXT NOT NULL CHECK (length(trim(currency)) > 0),
    counterparty TEXT NOT NULL CHECK (length(trim(counterparty)) > 0),
    verdict TEXT NOT NULL CHECK (verdict IN ('refused', 'pending', 'approved', 'released', 'settled')),
    violated_constraint TEXT NOT NULL,
    remediation TEXT NOT NULL,
    hold_minor INTEGER NOT NULL CHECK (hold_minor >= 0),
    created_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    CHECK ((verdict IN ('pending', 'approved') AND hold_minor = amount_minor) OR (verdict NOT IN ('pending', 'approved') AND hold_minor = 0))
);

-- A committed write to this row serializes authorization admission against
-- budget/book/scenario freeze writes in SQLite. The repository acquires it as
-- the first transaction write, then reads the switches and persists the
-- authorization in that same transaction.
CREATE TABLE IF NOT EXISTS authorization_admission_gate (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    generation INTEGER NOT NULL CHECK (generation >= 0)
);
INSERT OR IGNORE INTO authorization_admission_gate(id,generation) VALUES(1,0);

-- Settlement terms are part of the approved spend intent. This additive table
-- preserves upgrades while preventing an approved authorization from being
-- settled against a different instrument or replay key.
CREATE TABLE IF NOT EXISTS authorization_spend_intents (
    authorization_id TEXT PRIMARY KEY REFERENCES authorizations(id) ON DELETE CASCADE,
    settlement_id TEXT NOT NULL CHECK (length(trim(settlement_id)) > 0),
    instrument_id TEXT NOT NULL CHECK (length(trim(instrument_id)) > 0),
    settlement_idempotency_key TEXT NOT NULL UNIQUE CHECK (length(trim(settlement_idempotency_key)) > 0)
);

-- Book ownership is a domain-owned binding rather than a column migration on
-- the existing authorization ledger. This keeps upgrades safe while making
-- the isolation boundary mandatory for every newly persisted authorization.
CREATE TABLE IF NOT EXISTS authorization_book_bindings (
    authorization_id TEXT PRIMARY KEY REFERENCES authorizations(id),
    book_id TEXT NOT NULL CHECK (length(trim(book_id)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_authorization_book_bindings_book
ON authorization_book_bindings(book_id, authorization_id);

CREATE TRIGGER IF NOT EXISTS authorization_book_binding_insert_guard
BEFORE INSERT ON authorization_book_bindings
FOR EACH ROW
WHEN EXISTS (
    SELECT 1
    FROM authorizations a
    LEFT JOIN mandates m ON m.id = a.mandate_id
    LEFT JOIN budgets b ON b.id = a.budget_id
    WHERE a.id = NEW.authorization_id
      AND ((m.id IS NOT NULL AND m.book_id <> NEW.book_id)
        OR (b.id IS NOT NULL AND b.book_id <> NEW.book_id))
)
BEGIN
    SELECT RAISE(ABORT, 'authorization book must match mandate and budget books');
END;

CREATE TRIGGER IF NOT EXISTS authorization_book_binding_update_guard
BEFORE UPDATE OF book_id, authorization_id ON authorization_book_bindings
FOR EACH ROW
WHEN EXISTS (
    SELECT 1
    FROM authorizations a
    LEFT JOIN mandates m ON m.id = a.mandate_id
    LEFT JOIN budgets b ON b.id = a.budget_id
    WHERE a.id = NEW.authorization_id
      AND ((m.id IS NOT NULL AND m.book_id <> NEW.book_id)
        OR (b.id IS NOT NULL AND b.book_id <> NEW.book_id))
)
BEGIN
    SELECT RAISE(ABORT, 'authorization book must match mandate and budget books');
END;

CREATE INDEX IF NOT EXISTS authorizations_budget_usage
ON authorizations(budget_id, created_at, verdict, expires_at);

CREATE INDEX IF NOT EXISTS authorizations_mandate_usage
ON authorizations(mandate_id, verdict, expires_at);
