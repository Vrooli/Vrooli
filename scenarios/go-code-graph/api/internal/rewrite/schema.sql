-- rewrite domain schema — REQ-P1-002 Persistent Plan Store + Operation Log.
--
-- `rewrite_plans` is the durable PlanStore. Plans survive scenario
-- restarts; Apply may be called against a plan minutes or hours after
-- it was authored.
--
-- `rewrite_operation_log` records the per-operation outcome of every
-- non-dry-run Apply. Operators can audit historical applies after a
-- scenario restart; the row count for a single Apply equals the number
-- of operations in the plan.
CREATE TABLE IF NOT EXISTS rewrite_plans (
    id            TEXT PRIMARY KEY,
    scenario_path TEXT NOT NULL,
    operations    TEXT NOT NULL,
    created_at    INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS rewrite_operation_log (
    plan_id    TEXT NOT NULL,
    op_index   INTEGER NOT NULL,
    kind       TEXT NOT NULL,
    status     INTEGER NOT NULL,
    message    TEXT NOT NULL,
    applied_at INTEGER NOT NULL,
    PRIMARY KEY (plan_id, op_index, applied_at)
);

CREATE INDEX IF NOT EXISTS idx_rewrite_operation_log_plan_id
    ON rewrite_operation_log(plan_id);
