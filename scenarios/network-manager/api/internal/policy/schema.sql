CREATE TABLE IF NOT EXISTS policy_change_plans (
    id TEXT PRIMARY KEY,
    target TEXT NOT NULL,
    action TEXT NOT NULL,
    status TEXT NOT NULL,
    values_json TEXT NOT NULL,
    effects_json TEXT NOT NULL,
    rollback_supported INTEGER NOT NULL,
    rollback_handle TEXT NOT NULL,
    approval_id TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_policy_change_plans_status
ON policy_change_plans(status, updated_at DESC);

CREATE TABLE IF NOT EXISTS approval_records (
    id TEXT PRIMARY KEY,
    change_id TEXT NOT NULL,
    approved INTEGER NOT NULL,
    note TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_approval_records_change
ON approval_records(change_id, created_at DESC);

CREATE TABLE IF NOT EXISTS rollback_records (
    id TEXT PRIMARY KEY,
    change_id TEXT NOT NULL,
    status TEXT NOT NULL,
    details_json TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_rollback_records_change
ON rollback_records(change_id, created_at DESC);
