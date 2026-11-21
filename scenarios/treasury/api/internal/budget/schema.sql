CREATE TABLE IF NOT EXISTS budgets (
    id TEXT PRIMARY KEY,
    book_id TEXT NOT NULL REFERENCES books(id),
    currency TEXT NOT NULL CHECK (length(trim(currency)) > 0),
    total_cap_minor INTEGER NOT NULL CHECK (total_cap_minor > 0),
    periodic_cap_minor INTEGER NOT NULL CHECK (periodic_cap_minor > 0),
    per_transaction_cap_minor INTEGER NOT NULL CHECK (per_transaction_cap_minor > 0),
    period_seconds INTEGER NOT NULL CHECK (period_seconds > 0),
    requires_approval INTEGER NOT NULL CHECK (requires_approval IN (0, 1)),
    frozen INTEGER NOT NULL CHECK (frozen IN (0, 1)),
    created_at TEXT NOT NULL,
    CHECK (per_transaction_cap_minor <= periodic_cap_minor),
    CHECK (periodic_cap_minor <= total_cap_minor)
);

CREATE TABLE IF NOT EXISTS budget_scope_entries (
    budget_id TEXT NOT NULL REFERENCES budgets(id) ON DELETE CASCADE,
    counterparty TEXT NOT NULL CHECK (length(trim(counterparty)) > 0),
    effect TEXT NOT NULL CHECK (effect IN ('allow', 'deny')),
    PRIMARY KEY (budget_id, counterparty, effect)
);

-- Book and scenario freezes are domain-owned controls with soft book IDs.
-- Budget-local freeze remains on budgets because it is part of that entity.
CREATE TABLE IF NOT EXISTS freeze_controls (
    scope TEXT NOT NULL CHECK (scope IN ('book', 'scenario')),
    scope_id TEXT NOT NULL CHECK (length(trim(scope_id)) > 0),
    frozen INTEGER NOT NULL CHECK (frozen IN (0, 1)),
    updated_at TEXT NOT NULL,
    PRIMARY KEY (scope, scope_id),
    CHECK ((scope = 'scenario' AND scope_id = '*') OR (scope = 'book' AND scope_id <> '*'))
);

-- A receipt-funded budget keeps its policy caps. Accepted receipts provide a
-- second, independent limit; they never raise an operator's spending policy.
CREATE TABLE IF NOT EXISTS budget_funding_bindings (
    budget_id TEXT PRIMARY KEY REFERENCES budgets(id),
    network TEXT NOT NULL,
    asset_id TEXT NOT NULL,
    decimals INTEGER NOT NULL CHECK (decimals >= 0 AND decimals <= 255),
    created_at TEXT NOT NULL
);

-- One immutable delta per accepted Money Ledger receipt. Token quantity is
-- decimal text so no floating point or int64 conversion can lose atomic units.
CREATE TABLE IF NOT EXISTS budget_funding_credits (
    ledger_receipt_id TEXT PRIMARY KEY,
    budget_id TEXT NOT NULL REFERENCES budget_funding_bindings(budget_id),
    posting_id TEXT NOT NULL,
    source_check_id TEXT NOT NULL,
    atomic_delta TEXT NOT NULL,
    valuation_delta_minor INTEGER NOT NULL,
    valuation_currency TEXT NOT NULL,
    accepted_at TEXT NOT NULL,
    payload_sha256 TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_budget_funding_credits_budget
    ON budget_funding_credits(budget_id, accepted_at, ledger_receipt_id);
