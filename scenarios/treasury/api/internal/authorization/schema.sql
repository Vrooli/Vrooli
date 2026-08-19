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

CREATE INDEX IF NOT EXISTS authorizations_budget_usage
ON authorizations(budget_id, created_at, verdict, expires_at);

CREATE INDEX IF NOT EXISTS authorizations_mandate_usage
ON authorizations(mandate_id, verdict, expires_at);
