package monetization

// SQLiteSchema creates the shared Class B outbox table. Scenario migrations
// should append this schema to their domain bootstrap rather than reimplement
// the table or retry state locally.
const SQLiteSchema = `
CREATE TABLE IF NOT EXISTS monetization_usage_outbox (
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

CREATE INDEX IF NOT EXISTS idx_monetization_usage_outbox_pending
    ON monetization_usage_outbox(status, next_attempt_at);
`

// PostgresSchema is the equivalent shared outbox migration for LPBS-backed
// PostgreSQL scenario databases.
const PostgresSchema = `
CREATE TABLE IF NOT EXISTS monetization_usage_outbox (
    operation_id TEXT PRIMARY KEY,
    user_identity TEXT NOT NULL,
    payload JSONB NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_error TEXT,
    delivered_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_monetization_usage_outbox_pending
    ON monetization_usage_outbox(status, next_attempt_at);
`
