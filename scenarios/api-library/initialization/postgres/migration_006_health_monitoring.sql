-- Migration: Add API health monitoring and uptime tracking
-- This adds tables and functionality for monitoring API availability and performance

-- Health check results table
CREATE TABLE IF NOT EXISTS api_health_checks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    check_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    response_time_ms INTEGER,
    status_code INTEGER,
    healthy BOOLEAN NOT NULL,
    error_message TEXT,
    
    -- Additional details
    endpoint_checked VARCHAR(500), -- Specific endpoint if not base URL
    headers_sent JSONB, -- Headers used in the check
    response_headers JSONB, -- Headers received in response
    
    -- Indexes for performance
    CONSTRAINT valid_response_time CHECK (response_time_ms >= 0)
);

-- Aggregated health metrics table
CREATE TABLE IF NOT EXISTS api_health_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID UNIQUE NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    uptime_percentage NUMERIC(5,2),
    avg_response_time_ms INTEGER,
    total_checks INTEGER DEFAULT 0,
    successful_checks INTEGER DEFAULT 0,
    consecutive_failures INTEGER DEFAULT 0,
    current_status VARCHAR(20) DEFAULT 'unknown' CHECK (current_status IN ('healthy', 'degraded', 'unhealthy', 'unknown')),
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Time-based metrics
    last_24h_uptime NUMERIC(5,2),
    last_7d_uptime NUMERIC(5,2),
    last_30d_uptime NUMERIC(5,2),
    
    -- Performance metrics
    p50_response_time_ms INTEGER, -- Median
    p95_response_time_ms INTEGER, -- 95th percentile
    p99_response_time_ms INTEGER  -- 99th percentile
);

-- Scheduled health check configurations
CREATE TABLE IF NOT EXISTS health_check_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID UNIQUE NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    enabled BOOLEAN DEFAULT true,
    check_interval_minutes INTEGER DEFAULT 5,
    timeout_seconds INTEGER DEFAULT 10,
    retry_count INTEGER DEFAULT 2,
    custom_endpoint VARCHAR(500), -- Use this instead of base_url if specified
    custom_headers JSONB, -- Additional headers to send
    expected_status_codes INTEGER[], -- List of acceptable status codes
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Outage tracking table
CREATE TABLE IF NOT EXISTS api_outages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,
    duration_minutes INTEGER,
    consecutive_failures INTEGER,
    error_summary TEXT,
    resolution_notes TEXT,
    severity VARCHAR(20) CHECK (severity IN ('minor', 'major', 'critical')),
    
    -- Calculate severity based on duration
    CONSTRAINT calculate_severity CHECK (
        severity = CASE 
            WHEN duration_minutes < 5 THEN 'minor'
            WHEN duration_minutes < 60 THEN 'major'
            ELSE 'critical'
        END OR duration_minutes IS NULL
    )
);

-- Create indexes for faster queries
CREATE INDEX IF NOT EXISTS idx_health_checks_api_time ON api_health_checks(api_id, check_time DESC);
CREATE INDEX IF NOT EXISTS idx_health_checks_healthy ON api_health_checks(api_id, healthy);
CREATE INDEX IF NOT EXISTS idx_health_metrics_status ON api_health_metrics(current_status);
CREATE INDEX IF NOT EXISTS idx_outages_api ON api_outages(api_id, started_at DESC);

-- Function to calculate uptime percentages
CREATE OR REPLACE FUNCTION calculate_uptime_percentage(
    p_api_id UUID,
    p_hours INTEGER DEFAULT 24
) RETURNS NUMERIC AS $$
DECLARE
    v_total_checks INTEGER;
    v_successful_checks INTEGER;
    v_uptime_percentage NUMERIC(5,2);
BEGIN
    SELECT 
        COUNT(*),
        COUNT(*) FILTER (WHERE healthy = true)
    INTO v_total_checks, v_successful_checks
    FROM api_health_checks
    WHERE api_id = p_api_id
    AND check_time > NOW() - (p_hours || ' hours')::INTERVAL;
    
    IF v_total_checks > 0 THEN
        v_uptime_percentage := (v_successful_checks::NUMERIC / v_total_checks) * 100;
    ELSE
        v_uptime_percentage := NULL;
    END IF;
    
    RETURN v_uptime_percentage;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update metrics after each health check
CREATE OR REPLACE FUNCTION update_health_metrics_trigger()
RETURNS TRIGGER AS $$
BEGIN
    -- Update or insert health metrics
    INSERT INTO api_health_metrics (
        api_id,
        uptime_percentage,
        last_24h_uptime,
        last_7d_uptime,
        last_30d_uptime,
        total_checks,
        successful_checks,
        consecutive_failures,
        current_status,
        last_updated
    )
    VALUES (
        NEW.api_id,
        calculate_uptime_percentage(NEW.api_id, 24*7), -- 7 day average
        calculate_uptime_percentage(NEW.api_id, 24),
        calculate_uptime_percentage(NEW.api_id, 24*7),
        calculate_uptime_percentage(NEW.api_id, 24*30),
        1,
        CASE WHEN NEW.healthy THEN 1 ELSE 0 END,
        CASE WHEN NEW.healthy THEN 0 ELSE 1 END,
        CASE WHEN NEW.healthy THEN 'healthy' ELSE 'degraded' END,
        NOW()
    )
    ON CONFLICT (api_id) DO UPDATE SET
        uptime_percentage = calculate_uptime_percentage(NEW.api_id, 24*7),
        last_24h_uptime = calculate_uptime_percentage(NEW.api_id, 24),
        last_7d_uptime = calculate_uptime_percentage(NEW.api_id, 24*7),
        last_30d_uptime = calculate_uptime_percentage(NEW.api_id, 24*30),
        total_checks = api_health_metrics.total_checks + 1,
        successful_checks = api_health_metrics.successful_checks + CASE WHEN NEW.healthy THEN 1 ELSE 0 END,
        consecutive_failures = CASE 
            WHEN NEW.healthy THEN 0 
            ELSE api_health_metrics.consecutive_failures + 1 
        END,
        current_status = CASE
            WHEN NEW.healthy THEN 'healthy'
            WHEN api_health_metrics.consecutive_failures >= 2 THEN 'unhealthy'
            ELSE 'degraded'
        END,
        last_updated = NOW();
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create the trigger
CREATE TRIGGER update_health_metrics
AFTER INSERT ON api_health_checks
FOR EACH ROW EXECUTE FUNCTION update_health_metrics_trigger();

-- Sample health check configurations for testing
-- INSERT INTO health_check_configs (api_id, check_interval_minutes, custom_headers)
-- SELECT id, 5, '{"User-Agent": "API-Library-Health-Monitor/1.0"}'::jsonb
-- FROM apis
-- WHERE status = 'active' AND base_url IS NOT NULL
-- LIMIT 3;

-- View for current API health status
CREATE OR REPLACE VIEW api_health_status AS
SELECT 
    a.id,
    a.name,
    a.provider,
    a.category,
    m.current_status,
    m.uptime_percentage,
    m.avg_response_time_ms,
    m.consecutive_failures,
    m.last_updated,
    CASE 
        WHEN m.consecutive_failures >= 3 THEN 'Alert: API is down'
        WHEN m.consecutive_failures >= 1 THEN 'Warning: API may be experiencing issues'
        ELSE 'OK'
    END AS alert_status
FROM apis a
LEFT JOIN api_health_metrics m ON a.id = m.api_id
WHERE a.status IN ('active', 'beta');

COMMENT ON TABLE api_health_checks IS 'Stores individual health check results for APIs';
COMMENT ON TABLE api_health_metrics IS 'Aggregated health metrics and uptime statistics for APIs';
COMMENT ON TABLE health_check_configs IS 'Configuration for automated health monitoring of APIs';
COMMENT ON TABLE api_outages IS 'Tracks API outages and downtime incidents';