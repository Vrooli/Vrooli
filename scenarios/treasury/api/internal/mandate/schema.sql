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

CREATE TABLE IF NOT EXISTS mandate_recurrences (
    mandate_id TEXT PRIMARY KEY REFERENCES mandates(id) ON DELETE CASCADE,
    interval_seconds INTEGER NOT NULL CHECK (interval_seconds >= 60),
    next_charge_at TEXT NOT NULL,
    cancelled_at TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL,
    CHECK (cancelled_at = '' OR cancelled_at <= updated_at)
);

CREATE INDEX IF NOT EXISTS mandate_recurrences_due
ON mandate_recurrences(next_charge_at, mandate_id)
WHERE cancelled_at = '';

-- INVARIANT: onlyOperatorBeneficiaryCanBeRepresented
-- A mandate and its budget must remain in the same operator-owned book. These
-- triggers also upgrade greenfield development databases whose tables already
-- existed before this constraint was added.
CREATE TRIGGER IF NOT EXISTS mandates_book_budget_match_insert
BEFORE INSERT ON mandates
FOR EACH ROW
WHEN NOT EXISTS (
    SELECT 1 FROM budgets
    WHERE budgets.id = NEW.budget_id AND budgets.book_id = NEW.book_id
)
BEGIN
    SELECT RAISE(ABORT, 'mandate budget must belong to mandate book');
END;

CREATE TRIGGER IF NOT EXISTS mandates_book_budget_match_update
BEFORE UPDATE OF book_id, budget_id ON mandates
FOR EACH ROW
WHEN NOT EXISTS (
    SELECT 1 FROM budgets
    WHERE budgets.id = NEW.budget_id AND budgets.book_id = NEW.book_id
)
BEGIN
    SELECT RAISE(ABORT, 'mandate budget must belong to mandate book');
END;
