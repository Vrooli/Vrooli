-- Vrooli Autoheal Database Schema
-- [REQ:PERSIST-STORE-001] [REQ:PERSIST-QUERY-001] [REQ:PERSIST-HISTORY-001]

-- Health check results table
CREATE TABLE IF NOT EXISTS health_results (
    id SERIAL PRIMARY KEY,
    check_id VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('ok', 'warning', 'critical')),
    message TEXT NOT NULL,
    details JSONB DEFAULT '{}',
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for querying latest results per check
CREATE INDEX IF NOT EXISTS idx_health_results_check_id_created
    ON health_results (check_id, created_at DESC);

-- Index for time-based queries (history cleanup)
CREATE INDEX IF NOT EXISTS idx_health_results_created_at
    ON health_results (created_at);

-- Auto-heal action log
CREATE TABLE IF NOT EXISTS autoheal_actions (
    id SERIAL PRIMARY KEY,
    check_id VARCHAR(100) NOT NULL,
    action_type VARCHAR(50) NOT NULL,
    target VARCHAR(255) NOT NULL,
    success BOOLEAN NOT NULL DEFAULT false,
    message TEXT,
    details JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for querying recent actions
CREATE INDEX IF NOT EXISTS idx_autoheal_actions_created_at
    ON autoheal_actions (created_at DESC);

-- Recovery action logs table (for UI-triggered actions)
-- [REQ:HEAL-ACTION-001]
CREATE TABLE IF NOT EXISTS action_logs (
    id SERIAL PRIMARY KEY,
    check_id VARCHAR(100) NOT NULL,
    action_id VARCHAR(50) NOT NULL,
    success BOOLEAN NOT NULL DEFAULT false,
    message TEXT NOT NULL,
    output TEXT,
    error TEXT,
    duration_ms INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Index for querying action logs by check
CREATE INDEX IF NOT EXISTS idx_action_logs_check_id_created
    ON action_logs (check_id, created_at DESC);

-- Index for querying recent action logs
CREATE INDEX IF NOT EXISTS idx_action_logs_created_at
    ON action_logs (created_at DESC);

-- Configuration table for check intervals and settings
CREATE TABLE IF NOT EXISTS autoheal_config (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- View for latest result per check
CREATE OR REPLACE VIEW latest_health_results AS
SELECT DISTINCT ON (check_id) *
FROM health_results
ORDER BY check_id, created_at DESC;

-- View for health summary
CREATE OR REPLACE VIEW health_summary AS
SELECT
    COUNT(*) as total_checks,
    COUNT(*) FILTER (WHERE status = 'ok') as ok_count,
    COUNT(*) FILTER (WHERE status = 'warning') as warning_count,
    COUNT(*) FILTER (WHERE status = 'critical') as critical_count,
    CASE
        WHEN COUNT(*) FILTER (WHERE status = 'critical') > 0 THEN 'critical'
        WHEN COUNT(*) FILTER (WHERE status = 'warning') > 0 THEN 'warning'
        ELSE 'ok'
    END as overall_status
FROM latest_health_results;

-- Function to cleanup old results (keep 24 hours by default)
CREATE OR REPLACE FUNCTION cleanup_old_results(retention_hours INTEGER DEFAULT 24)
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
