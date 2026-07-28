CREATE TABLE IF NOT EXISTS credit_usage (
    id TEXT PRIMARY KEY,
    user_identity TEXT NOT NULL,
    billing_month TEXT NOT NULL,  -- Format: YYYY-MM

    -- Single unified credit pool totals
    total_credits_used INTEGER NOT NULL DEFAULT 0,
    total_operations INTEGER NOT NULL DEFAULT 0,

    -- Breakdown by operation type for analytics/UI display
    -- Format: {"ai.workflow_generate": 15, "execution.run": 42}
    credits_by_operation TEXT NOT NULL DEFAULT '{}',  -- JSON as TEXT
    operations_by_type TEXT NOT NULL DEFAULT '{}',  -- JSON as TEXT

    last_operation_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (user_identity, billing_month)
);

CREATE INDEX IF NOT EXISTS idx_credit_usage_user ON credit_usage(user_identity);
CREATE INDEX IF NOT EXISTS idx_credit_usage_month ON credit_usage(billing_month);

-- Unified Operation Log
-- Detailed audit log for all credit-consuming operations
CREATE TABLE IF NOT EXISTS operation_log (
    id TEXT PRIMARY KEY,
    user_identity TEXT NOT NULL,
    operation_type TEXT NOT NULL,  -- e.g., 'ai.workflow_generate', 'execution.run'
    credits_charged INTEGER NOT NULL DEFAULT 0,
    success INTEGER NOT NULL DEFAULT 1,  -- SQLite uses INTEGER for boolean

    -- Flexible metadata for operation-specific details
    metadata TEXT DEFAULT '{}',  -- JSON as TEXT

    error_message TEXT,
    duration_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_operation_log_user ON operation_log(user_identity);
CREATE INDEX IF NOT EXISTS idx_operation_log_type ON operation_log(operation_type);
CREATE INDEX IF NOT EXISTS idx_operation_log_created ON operation_log(created_at);

-- ============================================================================
-- UX METRICS: Per-step interaction traces, cursor paths, and execution-level scores
-- Backs services/uxmetrics. Repository code passes its own id only for traces;
-- cursor_paths and execution_metrics rely on the DEFAULT random hex id.
-- ============================================================================

