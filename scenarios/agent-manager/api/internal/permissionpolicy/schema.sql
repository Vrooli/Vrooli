-- ============================================================================
-- Permission policy reconcile audit - Agent Manager-owned metadata only
-- ============================================================================
CREATE TABLE IF NOT EXISTS permission_policy_reconcile_audit (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    catalog_digest TEXT NOT NULL,
    started_at TEXT NOT NULL,
    finished_at TEXT NOT NULL,
    explicitly_authorized INTEGER NOT NULL,
    success INTEGER NOT NULL,
    hard_enforcement_satisfied INTEGER NOT NULL,
    missing_hard_enforcement_rule_ids TEXT NOT NULL,
    resource_results TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_permission_policy_reconcile_audit_finished
    ON permission_policy_reconcile_audit(finished_at DESC);
