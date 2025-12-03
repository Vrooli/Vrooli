-- Vrooli Autoheal database schema
-- [REQ:PERSIST-STORE-001] [REQ:PERSIST-QUERY-001] [REQ:PERSIST-QUERY-002]
-- [REQ:PERSIST-HISTORY-001] [REQ:PERSIST-HISTORY-002] [REQ:PERSIST-ACTIONS-001]

-- Health check results table
CREATE TABLE IF NOT EXISTS health_results (
    id SERIAL PRIMARY KEY,
    check_id VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('ok', 'warning', 'critical')),
    message TEXT,
    details JSONB DEFAULT '{}',
    duration_ms BIGINT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),

    -- Index for querying by check_id and time
    CONSTRAINT valid_status CHECK (status IN ('ok', 'warning', 'critical'))
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_health_results_check_id ON health_results(check_id);
CREATE INDEX IF NOT EXISTS idx_health_results_created_at ON health_results(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_health_results_status ON health_results(status);
CREATE INDEX IF NOT EXISTS idx_health_results_check_status ON health_results(check_id, status, created_at DESC);

-- Auto-heal actions log
CREATE TABLE IF NOT EXISTS autoheal_actions (
    id SERIAL PRIMARY KEY,
    check_id VARCHAR(100) NOT NULL,
    action_type VARCHAR(50) NOT NULL,
    target VARCHAR(255) NOT NULL,
    success BOOLEAN NOT NULL DEFAULT false,
    message TEXT,
    details JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_autoheal_actions_check_id ON autoheal_actions(check_id);
CREATE INDEX IF NOT EXISTS idx_autoheal_actions_created_at ON autoheal_actions(created_at DESC);

-- Configuration table
CREATE TABLE IF NOT EXISTS autoheal_config (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Insert default configuration
INSERT INTO autoheal_config (key, value, description) VALUES
    ('tick_interval_seconds', '60', 'Default interval between health check cycles'),
    ('history_retention_hours', '24', 'How long to retain health check history'),
    ('auto_heal_enabled', 'true', 'Whether auto-healing actions are enabled'),
    ('max_restart_attempts', '3', 'Maximum restart attempts per check per hour')
ON CONFLICT (key) DO NOTHING;

-- Function to clean up old health results (24-hour retention by default)
-- [REQ:PERSIST-HISTORY-002]
CREATE OR REPLACE FUNCTION cleanup_old_health_results(retention_hours INTEGER DEFAULT 24)
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM health_results
    WHERE created_at < NOW() - (retention_hours || ' hours')::INTERVAL;
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- View for latest result per check
-- [REQ:PERSIST-QUERY-001]
CREATE OR REPLACE VIEW latest_health_results AS
SELECT DISTINCT ON (check_id)
    id,
    check_id,
    status,
    message,
    details,
    duration_ms,
    created_at
FROM health_results
ORDER BY check_id, created_at DESC;

-- View for health summary
-- [REQ:PERSIST-QUERY-002]
CREATE OR REPLACE VIEW health_summary AS
SELECT
    COUNT(*) FILTER (WHERE status = 'ok') AS ok_count,
    COUNT(*) FILTER (WHERE status = 'warning') AS warning_count,
    COUNT(*) FILTER (WHERE status = 'critical') AS critical_count,
    COUNT(*) AS total_count,
    CASE
        WHEN COUNT(*) FILTER (WHERE status = 'critical') > 0 THEN 'critical'
        WHEN COUNT(*) FILTER (WHERE status = 'warning') > 0 THEN 'warning'
        ELSE 'ok'
    END AS overall_status
FROM latest_health_results;
