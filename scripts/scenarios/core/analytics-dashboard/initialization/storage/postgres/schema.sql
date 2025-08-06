-- Resource Monitoring Platform Database Schema
-- Provides resource registry, alert configuration, and audit logging

-- Create schema for monitoring platform
CREATE SCHEMA IF NOT EXISTS resource_monitoring;

-- Set search path
SET search_path TO resource_monitoring;

-- ============================================
-- Resource Registry Tables
-- ============================================

-- Discovered resources and their metadata
CREATE TABLE IF NOT EXISTS resources (
    id SERIAL PRIMARY KEY,
    resource_name VARCHAR(100) UNIQUE NOT NULL,
    resource_type VARCHAR(50) NOT NULL, -- e.g., 'ai', 'storage', 'automation', 'agent'
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Connection details
    host VARCHAR(255) DEFAULT 'localhost',
    port INTEGER,
    base_url VARCHAR(500),
    health_check_endpoint VARCHAR(500),
    
    -- Discovery metadata
    discovered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    auto_discovered BOOLEAN DEFAULT true,
    
    -- Status tracking
    is_enabled BOOLEAN DEFAULT true,
    is_critical BOOLEAN DEFAULT false, -- Critical resources trigger immediate alerts
    current_status VARCHAR(20) DEFAULT 'unknown', -- 'healthy', 'degraded', 'down', 'unknown'
    last_status_change TIMESTAMP WITH TIME ZONE,
    
    -- Configuration
    config JSONB DEFAULT '{}', -- Resource-specific configuration
    custom_labels JSONB DEFAULT '{}', -- User-defined labels
    
    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(100) DEFAULT 'system',
    updated_by VARCHAR(100) DEFAULT 'system'
);

-- Create indexes for performance
CREATE INDEX idx_resources_type ON resources(resource_type);
CREATE INDEX idx_resources_status ON resources(current_status);
CREATE INDEX idx_resources_enabled ON resources(is_enabled);
CREATE INDEX idx_resources_critical ON resources(is_critical);

-- ============================================
-- Alert Configuration Tables
-- ============================================

-- Alert rules for resources
CREATE TABLE IF NOT EXISTS alert_rules (
    id SERIAL PRIMARY KEY,
    rule_name VARCHAR(100) UNIQUE NOT NULL,
    description TEXT,
    
    -- Rule configuration
    resource_id INTEGER REFERENCES resources(id) ON DELETE CASCADE,
    resource_type VARCHAR(50), -- Apply to all resources of this type
    metric_type VARCHAR(50) NOT NULL, -- 'availability', 'response_time', 'error_rate', 'custom'
    
    -- Thresholds
    threshold_value NUMERIC,
    threshold_operator VARCHAR(10), -- 'gt', 'lt', 'gte', 'lte', 'eq', 'neq'
    threshold_duration_seconds INTEGER DEFAULT 0, -- How long condition must persist
    
    -- Alert configuration
    severity VARCHAR(20) DEFAULT 'warning', -- 'info', 'warning', 'critical'
    is_enabled BOOLEAN DEFAULT true,
    
    -- Notification settings
    notification_channels JSONB DEFAULT '["email"]', -- ['sms', 'email', 'webhook', 'slack']
    cooldown_minutes INTEGER DEFAULT 15, -- Minimum time between alerts
    max_alerts_per_hour INTEGER DEFAULT 4,
    
    -- Custom logic
    custom_condition TEXT, -- Optional SQL/JavaScript condition
    custom_message_template TEXT,
    
    -- Statistics
    last_triggered TIMESTAMP WITH TIME ZONE,
    trigger_count INTEGER DEFAULT 0,
    
    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_alert_rules_resource ON alert_rules(resource_id);
CREATE INDEX idx_alert_rules_enabled ON alert_rules(is_enabled);
CREATE INDEX idx_alert_rules_severity ON alert_rules(severity);

-- ============================================
-- Alert History and Throttling
-- ============================================

-- Alert history for audit and analysis
CREATE TABLE IF NOT EXISTS alert_history (
    id SERIAL PRIMARY KEY,
    alert_rule_id INTEGER REFERENCES alert_rules(id) ON DELETE SET NULL,
    resource_id INTEGER REFERENCES resources(id) ON DELETE SET NULL,
    
    -- Alert details
    alert_time TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    severity VARCHAR(20),
    metric_type VARCHAR(50),
    metric_value NUMERIC,
    threshold_value NUMERIC,
    
    -- Notification details
    notification_sent BOOLEAN DEFAULT false,
    notification_channels JSONB,
    notification_time TIMESTAMP WITH TIME ZONE,
    notification_response JSONB,
    
    -- Alert message
    message TEXT,
    context JSONB, -- Additional context data
    
    -- Resolution tracking
    is_resolved BOOLEAN DEFAULT false,
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolution_notes TEXT,
    
    -- Indexing for queries
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_alert_history_time ON alert_history(alert_time DESC);
CREATE INDEX idx_alert_history_resource ON alert_history(resource_id);
CREATE INDEX idx_alert_history_rule ON alert_history(alert_rule_id);
CREATE INDEX idx_alert_history_severity ON alert_history(severity);
CREATE INDEX idx_alert_history_resolved ON alert_history(is_resolved);

-- ============================================
-- Resource Metrics Configuration
-- ============================================

-- Define which metrics to collect for each resource
CREATE TABLE IF NOT EXISTS metric_configurations (
    id SERIAL PRIMARY KEY,
    resource_id INTEGER REFERENCES resources(id) ON DELETE CASCADE,
    
    -- Metric definition
    metric_name VARCHAR(100) NOT NULL,
    metric_type VARCHAR(50) NOT NULL, -- 'gauge', 'counter', 'histogram'
    metric_unit VARCHAR(50), -- 'ms', 'bytes', 'percent', etc.
    
    -- Collection settings
    collection_interval_seconds INTEGER DEFAULT 30,
    is_enabled BOOLEAN DEFAULT true,
    
    -- Processing
    aggregation_method VARCHAR(20) DEFAULT 'avg', -- 'avg', 'sum', 'min', 'max', 'last'
    retention_days INTEGER DEFAULT 30,
    
    -- Custom collection logic
    custom_query TEXT, -- Optional custom collection query
    custom_parser TEXT, -- Optional custom parsing logic
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(resource_id, metric_name)
);

CREATE INDEX idx_metric_config_resource ON metric_configurations(resource_id);
CREATE INDEX idx_metric_config_enabled ON metric_configurations(is_enabled);

-- ============================================
-- Notification Configuration
-- ============================================

-- Store notification channel configurations
CREATE TABLE IF NOT EXISTS notification_channels (
    id SERIAL PRIMARY KEY,
    channel_name VARCHAR(100) UNIQUE NOT NULL,
    channel_type VARCHAR(50) NOT NULL, -- 'sms', 'email', 'webhook', 'slack'
    
    -- Configuration (encrypted via Vault reference)
    vault_path VARCHAR(255), -- Path to credentials in Vault
    config JSONB DEFAULT '{}', -- Non-sensitive configuration
    
    -- Channel settings
    is_enabled BOOLEAN DEFAULT true,
    is_default BOOLEAN DEFAULT false,
    priority INTEGER DEFAULT 0, -- Higher priority channels tried first
    
    -- Rate limiting
    max_messages_per_hour INTEGER DEFAULT 10,
    current_hour_count INTEGER DEFAULT 0,
    current_hour_start TIMESTAMP WITH TIME ZONE,
    
    -- Test status
    last_test_time TIMESTAMP WITH TIME ZONE,
    last_test_success BOOLEAN,
    last_test_message TEXT,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_notification_channels_type ON notification_channels(channel_type);
CREATE INDEX idx_notification_channels_enabled ON notification_channels(is_enabled);

-- ============================================
-- Resource Actions and Management
-- ============================================

-- Available actions for each resource type
CREATE TABLE IF NOT EXISTS resource_actions (
    id SERIAL PRIMARY KEY,
    resource_type VARCHAR(50) NOT NULL,
    action_name VARCHAR(100) NOT NULL,
    action_command TEXT NOT NULL, -- Shell command or API call
    
    -- Action metadata
    description TEXT,
    requires_confirmation BOOLEAN DEFAULT true,
    is_dangerous BOOLEAN DEFAULT false,
    
    -- Permissions
    min_role VARCHAR(50) DEFAULT 'operator', -- 'viewer', 'operator', 'admin'
    
    -- UI configuration
    icon VARCHAR(50),
    button_color VARCHAR(20) DEFAULT 'primary',
    display_order INTEGER DEFAULT 0,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(resource_type, action_name)
);

-- Action execution history
CREATE TABLE IF NOT EXISTS action_history (
    id SERIAL PRIMARY KEY,
    resource_id INTEGER REFERENCES resources(id) ON DELETE SET NULL,
    action_name VARCHAR(100),
    
    -- Execution details
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    executed_by VARCHAR(100),
    execution_time_ms INTEGER,
    
    -- Results
    success BOOLEAN,
    output TEXT,
    error_message TEXT,
    
    -- Context
    context JSONB DEFAULT '{}'
);

CREATE INDEX idx_action_history_resource ON action_history(resource_id);
CREATE INDEX idx_action_history_time ON action_history(executed_at DESC);

-- ============================================
-- Dashboard Configuration
-- ============================================

-- User dashboard preferences and layouts
CREATE TABLE IF NOT EXISTS dashboard_configs (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(100) UNIQUE NOT NULL,
    
    -- Layout configuration
    layout JSONB DEFAULT '{}', -- Grid layout, widget positions
    widgets JSONB DEFAULT '[]', -- Active widgets and their configs
    
    -- Preferences
    refresh_interval_seconds INTEGER DEFAULT 30,
    theme VARCHAR(20) DEFAULT 'light',
    timezone VARCHAR(50) DEFAULT 'UTC',
    
    -- Filters
    resource_filters JSONB DEFAULT '{}', -- Which resources to show/hide
    severity_filters JSONB DEFAULT '["warning", "critical"]',
    
    -- Saved views
    saved_views JSONB DEFAULT '[]',
    default_view VARCHAR(100),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================
-- System Configuration
-- ============================================

-- Global system configuration
CREATE TABLE IF NOT EXISTS system_config (
    key VARCHAR(100) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    is_sensitive BOOLEAN DEFAULT false,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_by VARCHAR(100) DEFAULT 'system'
);

-- Insert default configuration
INSERT INTO system_config (key, value, description) VALUES
    ('auto_discovery', '{"enabled": true, "interval_minutes": 60, "port_range": "1000-65535"}', 'Auto-discovery settings'),
    ('monitoring', '{"default_interval_seconds": 30, "retention_days": 30}', 'Default monitoring settings'),
    ('alerting', '{"default_cooldown_minutes": 15, "max_alerts_per_hour": 10}', 'Default alerting settings'),
    ('ui', '{"default_theme": "light", "enable_animations": true}', 'UI preferences')
ON CONFLICT (key) DO NOTHING;

-- ============================================
-- Functions and Triggers
-- ============================================

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply update trigger to relevant tables
CREATE TRIGGER update_resources_updated_at BEFORE UPDATE ON resources
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
    
CREATE TRIGGER update_alert_rules_updated_at BEFORE UPDATE ON alert_rules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
    
CREATE TRIGGER update_metric_configurations_updated_at BEFORE UPDATE ON metric_configurations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
    
CREATE TRIGGER update_notification_channels_updated_at BEFORE UPDATE ON notification_channels
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
    
CREATE TRIGGER update_dashboard_configs_updated_at BEFORE UPDATE ON dashboard_configs
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Function to get resource health summary
CREATE OR REPLACE FUNCTION get_resource_health_summary()
RETURNS TABLE (
    total_resources INTEGER,
    healthy_count INTEGER,
    degraded_count INTEGER,
    down_count INTEGER,
    unknown_count INTEGER,
    critical_down INTEGER,
    last_update TIMESTAMP WITH TIME ZONE
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        COUNT(*)::INTEGER AS total_resources,
        COUNT(CASE WHEN current_status = 'healthy' THEN 1 END)::INTEGER AS healthy_count,
        COUNT(CASE WHEN current_status = 'degraded' THEN 1 END)::INTEGER AS degraded_count,
        COUNT(CASE WHEN current_status = 'down' THEN 1 END)::INTEGER AS down_count,
        COUNT(CASE WHEN current_status = 'unknown' THEN 1 END)::INTEGER AS unknown_count,
        COUNT(CASE WHEN current_status = 'down' AND is_critical = true THEN 1 END)::INTEGER AS critical_down,
        MAX(last_seen) AS last_update
    FROM resource_monitoring.resources
    WHERE is_enabled = true;
END;
$$ LANGUAGE plpgsql;

-- Function to check alert throttling
CREATE OR REPLACE FUNCTION should_send_alert(
    p_rule_id INTEGER,
    p_resource_id INTEGER
) RETURNS BOOLEAN AS $$
DECLARE
    v_last_alert TIMESTAMP WITH TIME ZONE;
    v_cooldown_minutes INTEGER;
    v_recent_count INTEGER;
    v_max_per_hour INTEGER;
BEGIN
    -- Get alert rule configuration
    SELECT cooldown_minutes, max_alerts_per_hour
    INTO v_cooldown_minutes, v_max_per_hour
    FROM alert_rules
    WHERE id = p_rule_id;
    
    -- Check cooldown period
    SELECT MAX(alert_time)
    INTO v_last_alert
    FROM alert_history
    WHERE alert_rule_id = p_rule_id
      AND resource_id = p_resource_id
      AND notification_sent = true;
    
    IF v_last_alert IS NOT NULL AND 
       v_last_alert + (v_cooldown_minutes || ' minutes')::INTERVAL > NOW() THEN
        RETURN FALSE;
    END IF;
    
    -- Check hourly limit
    SELECT COUNT(*)
    INTO v_recent_count
    FROM alert_history
    WHERE alert_rule_id = p_rule_id
      AND resource_id = p_resource_id
      AND notification_sent = true
      AND alert_time > NOW() - INTERVAL '1 hour';
    
    IF v_recent_count >= v_max_per_hour THEN
        RETURN FALSE;
    END IF;
    
    RETURN TRUE;
END;
$$ LANGUAGE plpgsql;

-- ============================================
-- Views for Easy Querying
-- ============================================

-- Current resource status with alert counts
CREATE OR REPLACE VIEW resource_status_summary AS
SELECT 
    r.id,
    r.resource_name,
    r.resource_type,
    r.display_name,
    r.current_status,
    r.last_status_change,
    COUNT(DISTINCT ah.id) FILTER (WHERE ah.alert_time > NOW() - INTERVAL '24 hours' AND NOT ah.is_resolved) AS active_alerts,
    COUNT(DISTINCT ah.id) FILTER (WHERE ah.alert_time > NOW() - INTERVAL '24 hours') AS alerts_24h,
    MAX(ah.alert_time) AS last_alert_time
FROM resources r
LEFT JOIN alert_history ah ON r.id = ah.resource_id
WHERE r.is_enabled = true
GROUP BY r.id;

-- Alert rule effectiveness
CREATE OR REPLACE VIEW alert_rule_stats AS
SELECT 
    ar.id,
    ar.rule_name,
    ar.severity,
    ar.is_enabled,
    COUNT(ah.id) AS total_triggers,
    COUNT(ah.id) FILTER (WHERE ah.notification_sent) AS notifications_sent,
    AVG(EXTRACT(EPOCH FROM (ah.resolved_at - ah.alert_time))/60)::INTEGER AS avg_resolution_minutes,
    ar.last_triggered
FROM alert_rules ar
LEFT JOIN alert_history ah ON ar.id = ah.alert_rule_id
GROUP BY ar.id;

-- Grant permissions (adjust as needed)
GRANT ALL ON SCHEMA resource_monitoring TO vrooli_app;
GRANT ALL ON ALL TABLES IN SCHEMA resource_monitoring TO vrooli_app;
GRANT ALL ON ALL SEQUENCES IN SCHEMA resource_monitoring TO vrooli_app;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA resource_monitoring TO vrooli_app;