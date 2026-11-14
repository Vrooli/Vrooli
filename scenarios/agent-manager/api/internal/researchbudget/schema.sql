CREATE TABLE IF NOT EXISTS research_run_budgets (
    run_id TEXT PRIMARY KEY REFERENCES runs(id) ON DELETE RESTRICT,
    treasury_budget_id TEXT NOT NULL,
    mandate_id TEXT NOT NULL,
    agent_subject TEXT NOT NULL,
    cap_minor INTEGER NOT NULL CHECK(cap_minor >= 0),
    currency TEXT NOT NULL,
    unit_code TEXT NOT NULL,
    policy_revision TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS research_run_spend (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES research_run_budgets(run_id) ON DELETE RESTRICT,
    settlement_id TEXT NOT NULL UNIQUE,
    treasury_receipt_id TEXT NOT NULL UNIQUE,
    integration_id TEXT NOT NULL,
    category TEXT NOT NULL CHECK(category IN ('x402_call','compute','dataset')),
    amount_minor INTEGER NOT NULL CHECK(amount_minor >= 0),
    currency TEXT NOT NULL,
    unit_code TEXT NOT NULL,
    result_ref TEXT NOT NULL DEFAULT '',
    benefit_state TEXT NOT NULL CHECK(benefit_state IN ('measured','pending','none')),
    benefit_summary TEXT NOT NULL DEFAULT '',
    settled_at TEXT NOT NULL,
    recorded_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_research_run_spend_run_time
    ON research_run_spend(run_id, settled_at, id);

CREATE TRIGGER IF NOT EXISTS research_run_budgets_no_update
BEFORE UPDATE ON research_run_budgets BEGIN
    SELECT RAISE(ABORT, 'research run budget snapshots are immutable');
END;
CREATE TRIGGER IF NOT EXISTS research_run_budgets_no_delete
BEFORE DELETE ON research_run_budgets BEGIN
    SELECT RAISE(ABORT, 'research run budget snapshots are immutable');
END;
CREATE TRIGGER IF NOT EXISTS research_run_spend_no_update
BEFORE UPDATE ON research_run_spend BEGIN
    SELECT RAISE(ABORT, 'research spend evidence is immutable');
END;
CREATE TRIGGER IF NOT EXISTS research_run_spend_no_delete
BEFORE DELETE ON research_run_spend BEGIN
    SELECT RAISE(ABORT, 'research spend evidence is immutable');
END;
