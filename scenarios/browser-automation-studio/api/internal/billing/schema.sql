CREATE TABLE IF NOT EXISTS credit_usage (
    id TEXT PRIMARY KEY,
    user_identity TEXT NOT NULL,
    billing_month TEXT NOT NULL,
    total_credits_used INTEGER NOT NULL DEFAULT 0,
    total_operations INTEGER NOT NULL DEFAULT 0,
    credits_by_operation TEXT NOT NULL DEFAULT '{}',
    operations_by_type TEXT NOT NULL DEFAULT '{}',
    last_operation_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (user_identity, billing_month)
);

CREATE INDEX IF NOT EXISTS idx_credit_usage_user ON credit_usage(user_identity);
CREATE INDEX IF NOT EXISTS idx_credit_usage_month ON credit_usage(billing_month);

CREATE TABLE IF NOT EXISTS operation_log (
    id TEXT PRIMARY KEY,
    user_identity TEXT NOT NULL,
    operation_type TEXT NOT NULL,
    credits_charged INTEGER NOT NULL DEFAULT 0,
    success INTEGER NOT NULL DEFAULT 1,
    metadata TEXT DEFAULT '{}',
    error_message TEXT,
    duration_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_operation_log_user ON operation_log(user_identity);
CREATE INDEX IF NOT EXISTS idx_operation_log_type ON operation_log(operation_type);
CREATE INDEX IF NOT EXISTS idx_operation_log_created ON operation_log(created_at);

-- Durable LPBS usage delivery. operation_id is the authority idempotency key;
-- delivery retries never create a second usage record upstream.
CREATE TABLE IF NOT EXISTS usage_outbox (
    operation_id TEXT PRIMARY KEY,
    user_identity TEXT NOT NULL,
    payload TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_error TEXT,
    delivered_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_usage_outbox_pending
    ON usage_outbox(status, next_attempt_at);
