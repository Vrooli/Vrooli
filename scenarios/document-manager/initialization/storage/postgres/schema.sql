-- Document Manager Database Schema
-- Core tables for documentation health management

-- Applications being monitored
CREATE TABLE applications (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    repository_url TEXT,
    documentation_path TEXT DEFAULT '/docs',
    health_score DECIMAL(3,2) DEFAULT 0.0 CHECK (health_score >= 0.0 AND health_score <= 1.0),
    last_scan_result JSONB,
    notification_settings JSONB DEFAULT '{"email": false, "slack": true, "severity_threshold": "medium"}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    active BOOLEAN DEFAULT true
);

-- Documentation agents configuration
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('drift_detector', 'link_validator', 'organization_analyzer', 'coverage_checker', 'meta_optimizer', 'custom')),
    application_id UUID REFERENCES applications(id),
    config JSONB DEFAULT '{}',
    schedule_cron VARCHAR(100),
    auto_apply_threshold DECIMAL(3,2) DEFAULT 0.0 CHECK (auto_apply_threshold >= 0.0 AND auto_apply_threshold <= 1.0),
    last_performance_score DECIMAL(3,2),
    enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_run TIMESTAMP,
    next_run TIMESTAMP
);

-- Improvement suggestions queue
CREATE TABLE improvement_queue (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID REFERENCES agents(id),
    application_id UUID REFERENCES applications(id),
    type VARCHAR(50) NOT NULL,
    severity VARCHAR(20) CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    title TEXT NOT NULL,
    description TEXT,
    suggested_fix JSONB,
    user_feedback TEXT,
    applied_result JSONB,
    revert_possible BOOLEAN DEFAULT false,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'denied', 'revision_requested', 'applied', 'reverted')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    reviewed_at TIMESTAMP,
    reviewed_by VARCHAR(255),
    review_notes TEXT,
    applied_at TIMESTAMP,
    applied_by VARCHAR(255)
);

-- Action history tracking
CREATE TABLE action_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    queue_item_id UUID REFERENCES improvement_queue(id),
    agent_id UUID REFERENCES agents(id),
    action_type VARCHAR(50) NOT NULL,
    action_data JSONB,
    result VARCHAR(20) CHECK (result IN ('success', 'failure', 'partial')),
    error_message TEXT,
    execution_time_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Agent performance metrics
CREATE TABLE agent_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID REFERENCES agents(id),
    metric_date DATE NOT NULL,
    runs_count INTEGER DEFAULT 0,
    suggestions_count INTEGER DEFAULT 0,
    approved_count INTEGER DEFAULT 0,
    denied_count INTEGER DEFAULT 0,
    revision_count INTEGER DEFAULT 0,
    avg_execution_time_ms INTEGER,
    error_count INTEGER DEFAULT 0,
    UNIQUE(agent_id, metric_date)
);

-- Documentation coverage tracking
CREATE TABLE documentation_coverage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID REFERENCES applications(id),
    scan_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_files INTEGER,
    documented_files INTEGER,
    coverage_percentage DECIMAL(5,2),
    missing_docs JSONB,
    outdated_docs JSONB,
    quality_score DECIMAL(5,2)
);

-- Per-application configuration settings
CREATE TABLE app_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    application_id UUID REFERENCES applications(id) UNIQUE,
    auto_apply_enabled BOOLEAN DEFAULT false,
    auto_apply_max_severity VARCHAR(20) DEFAULT 'low',
    notification_channels JSONB DEFAULT '["slack"]',
    scan_frequency_hours INTEGER DEFAULT 24,
    retention_days INTEGER DEFAULT 90,
    custom_rules JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Reusable agent configuration templates
CREATE TABLE agent_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50) NOT NULL,
    default_config JSONB NOT NULL,
    default_schedule VARCHAR(100),
    is_system_template BOOLEAN DEFAULT false,
    created_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    active BOOLEAN DEFAULT true
);

-- User feedback on applied improvements
CREATE TABLE improvement_feedback (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    improvement_id UUID REFERENCES improvement_queue(id),
    feedback_type VARCHAR(20) CHECK (feedback_type IN ('positive', 'negative', 'neutral')),
    effectiveness_rating INTEGER CHECK (effectiveness_rating >= 1 AND effectiveness_rating <= 5),
    feedback_text TEXT,
    would_revert BOOLEAN,
    additional_suggestions TEXT,
    created_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User notification preferences
CREATE TABLE notification_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    application_id UUID REFERENCES applications(id),
    channel VARCHAR(20) CHECK (channel IN ('email', 'slack', 'webhook', 'ui')),
    severity_threshold VARCHAR(20) DEFAULT 'medium' CHECK (severity_threshold IN ('low', 'medium', 'high', 'critical')),
    enabled BOOLEAN DEFAULT true,
    schedule_window_start TIME,
    schedule_window_end TIME,
    timezone VARCHAR(50) DEFAULT 'UTC',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, application_id, channel)
);

-- Comprehensive audit log for compliance
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_type VARCHAR(50) NOT NULL,
    entity_id UUID NOT NULL,
    action VARCHAR(50) NOT NULL,
    old_values JSONB,
    new_values JSONB,
    performed_by VARCHAR(255),
    ip_address INET,
    user_agent TEXT,
    session_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_agents_application ON agents(application_id);
CREATE INDEX idx_queue_status ON improvement_queue(status);
CREATE INDEX idx_queue_application ON improvement_queue(application_id);
CREATE INDEX idx_history_agent ON action_history(agent_id);
CREATE INDEX idx_metrics_agent_date ON agent_metrics(agent_id, metric_date);
CREATE INDEX idx_coverage_application ON documentation_coverage(application_id);

-- Indexes for new tables
CREATE INDEX idx_app_settings_application ON app_settings(application_id);
CREATE INDEX idx_agent_templates_type ON agent_templates(type);
CREATE INDEX idx_agent_templates_active ON agent_templates(active);
CREATE INDEX idx_improvement_feedback_improvement ON improvement_feedback(improvement_id);
CREATE INDEX idx_notification_prefs_user ON notification_preferences(user_id);
CREATE INDEX idx_notification_prefs_app ON notification_preferences(application_id);
CREATE INDEX idx_audit_log_entity ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_log_timestamp ON audit_log(created_at);
CREATE INDEX idx_audit_log_user ON audit_log(performed_by);