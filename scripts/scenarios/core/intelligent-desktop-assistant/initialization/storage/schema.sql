-- Database schema template for scenario applications
-- This creates the core database structure for the application

-- Create application schema
CREATE SCHEMA IF NOT EXISTS intelligent-desktop-assistant;

-- Core application tables
CREATE TABLE IF NOT EXISTS intelligent-desktop-assistant.app_metadata (
    id SERIAL PRIMARY KEY,
    app_name VARCHAR(255) NOT NULL DEFAULT 'Intelligent Desktop Assistant',
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    deployed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    configuration JSONB,
    
    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User sessions and activity tracking
CREATE TABLE IF NOT EXISTS intelligent-desktop-assistant.user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id VARCHAR(255) UNIQUE NOT NULL,
    user_identifier VARCHAR(255),
    
    -- Session metadata
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_activity TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true,
    
    -- Context and preferences
    context_data JSONB,
    preferences JSONB,
    
    -- Performance tracking
    request_count INTEGER DEFAULT 0,
    total_processing_time_ms INTEGER DEFAULT 0
);

-- Application events and audit log
CREATE TABLE IF NOT EXISTS intelligent-desktop-assistant.activity_log (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Event details
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB,
    
    -- User and session context
    session_id UUID REFERENCES intelligent-desktop-assistant.user_sessions(id),
    user_identifier VARCHAR(255),
    
    -- Performance metrics
    processing_time_ms INTEGER,
    resource_usage JSONB,
    
    -- Status tracking
    status VARCHAR(50) DEFAULT 'completed', -- pending, completed, failed
    error_message TEXT
);

-- Resource usage tracking
CREATE TABLE IF NOT EXISTS intelligent-desktop-assistant.resource_metrics (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Resource identification
    resource_type VARCHAR(50) NOT NULL, -- ollama, n8n, whisper, etc.
    resource_name VARCHAR(100) NOT NULL,
    operation VARCHAR(100) NOT NULL,
    
    -- Performance metrics
    duration_ms INTEGER NOT NULL,
    success BOOLEAN NOT NULL DEFAULT true,
    
    -- Usage details
    input_size_bytes INTEGER,
    output_size_bytes INTEGER,
    tokens_used INTEGER, -- For AI resources
    
    -- Cost tracking
    estimated_cost_usd DECIMAL(10,6),
    
    -- Context
    session_id UUID REFERENCES intelligent-desktop-assistant.user_sessions(id),
    request_metadata JSONB
);

-- Configuration and feature flags
CREATE TABLE IF NOT EXISTS intelligent-desktop-assistant.app_config (
    id SERIAL PRIMARY KEY,
    config_key VARCHAR(255) UNIQUE NOT NULL,
    config_value JSONB NOT NULL,
    config_type VARCHAR(50) NOT NULL DEFAULT 'setting', -- setting, feature_flag, resource_config
    
    -- Metadata
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_user_sessions_session_id ON intelligent-desktop-assistant.user_sessions(session_id);
CREATE INDEX idx_user_sessions_active ON intelligent-desktop-assistant.user_sessions(is_active, last_activity);
CREATE INDEX idx_activity_log_timestamp ON intelligent-desktop-assistant.activity_log(timestamp DESC);
CREATE INDEX idx_activity_log_session ON intelligent-desktop-assistant.activity_log(session_id);
CREATE INDEX idx_activity_log_event_type ON intelligent-desktop-assistant.activity_log(event_type);
CREATE INDEX idx_resource_metrics_timestamp ON intelligent-desktop-assistant.resource_metrics(timestamp DESC);
CREATE INDEX idx_resource_metrics_resource ON intelligent-desktop-assistant.resource_metrics(resource_type, resource_name);
CREATE INDEX idx_app_config_key ON intelligent-desktop-assistant.app_config(config_key);

-- Useful views for monitoring
CREATE OR REPLACE VIEW intelligent-desktop-assistant.session_summary AS
SELECT 
    DATE_TRUNC('hour', started_at) as hour,
    COUNT(*) as total_sessions,
    COUNT(DISTINCT user_identifier) as unique_users,
    AVG(EXTRACT(EPOCH FROM (last_activity - started_at))) as avg_session_duration_seconds,
    SUM(request_count) as total_requests
FROM intelligent-desktop-assistant.user_sessions
GROUP BY DATE_TRUNC('hour', started_at)
ORDER BY hour DESC;

CREATE OR REPLACE VIEW intelligent-desktop-assistant.resource_performance AS
SELECT 
    resource_type,
    resource_name,
    COUNT(*) as total_calls,
    AVG(duration_ms) as avg_duration_ms,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95_duration_ms,
    SUM(CASE WHEN success THEN 1 ELSE 0 END)::FLOAT / COUNT(*) * 100 as success_rate_percent,
    SUM(COALESCE(estimated_cost_usd, 0)) as total_estimated_cost_usd
FROM intelligent-desktop-assistant.resource_metrics
WHERE timestamp > NOW() - INTERVAL '24 hours'
GROUP BY resource_type, resource_name
ORDER BY total_calls DESC;

-- Grant permissions (adjust based on your user setup)
GRANT USAGE ON SCHEMA intelligent-desktop-assistant TO PUBLIC;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA intelligent-desktop-assistant TO PUBLIC;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA intelligent-desktop-assistant TO PUBLIC;
GRANT SELECT ON ALL TABLES IN SCHEMA intelligent-desktop-assistant TO PUBLIC;