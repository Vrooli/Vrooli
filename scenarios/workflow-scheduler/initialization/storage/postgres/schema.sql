-- Workflow Scheduler Database Schema
-- Enterprise-grade scheduling platform for Vrooli workflows

-- Create database if not exists (handled by setup process)
-- Database name: workflow_scheduler

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- For text search optimization

-- Enum types for better data integrity
CREATE TYPE schedule_status AS ENUM ('active', 'paused', 'disabled', 'archived');
CREATE TYPE execution_status AS ENUM ('pending', 'running', 'success', 'failed', 'skipped', 'cancelled', 'timeout');
CREATE TYPE overlap_policy AS ENUM ('skip', 'queue', 'allow');
CREATE TYPE retry_strategy AS ENUM ('exponential', 'linear', 'fixed');
CREATE TYPE notification_channel AS ENUM ('webhook', 'email', 'slack', 'discord', 'ui');
CREATE TYPE severity_level AS ENUM ('low', 'medium', 'high', 'critical');

-- Core schedule definitions table
CREATE TABLE schedules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    cron_expression VARCHAR(100) NOT NULL,
    timezone VARCHAR(50) DEFAULT 'UTC',
    
    -- Target configuration
    target_type VARCHAR(50) NOT NULL DEFAULT 'webhook', -- webhook, workflow_id, windmill_job
    target_url TEXT,
    target_workflow_id VARCHAR(255),
    target_method VARCHAR(10) DEFAULT 'POST',
    target_headers JSONB DEFAULT '{}',
    target_payload JSONB DEFAULT '{}',
    
    -- Schedule configuration
    status schedule_status DEFAULT 'active',
    enabled BOOLEAN DEFAULT true,
    start_date TIMESTAMP WITH TIME ZONE,
    end_date TIMESTAMP WITH TIME ZONE,
    
    -- Execution policies
    overlap_policy overlap_policy DEFAULT 'skip',
    max_retries INTEGER DEFAULT 3,
    retry_strategy retry_strategy DEFAULT 'exponential',
    retry_delay_seconds INTEGER DEFAULT 300,
    timeout_seconds INTEGER DEFAULT 300,
    catch_up_missed BOOLEAN DEFAULT true,
    
    -- Notification settings
    notification_on_failure BOOLEAN DEFAULT true,
    notification_on_success BOOLEAN DEFAULT false,
    notification_channels notification_channel[] DEFAULT '{}',
    failure_webhook TEXT,
    
    -- Metadata
    created_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_executed_at TIMESTAMP WITH TIME ZONE,
    next_execution_at TIMESTAMP WITH TIME ZONE,
    
    -- Organization
    tags TEXT[] DEFAULT '{}',
    priority INTEGER DEFAULT 5 CHECK (priority >= 1 AND priority <= 10),
    owner VARCHAR(255),
    team VARCHAR(255),
    
    -- Constraints
    CONSTRAINT valid_cron CHECK (cron_expression ~ '^(\*|([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9])|\*\/([0-9]|1[0-9]|2[0-9]|3[0-9]|4[0-9]|5[0-9])) (\*|([0-9]|1[0-9]|2[0-3])|\*\/([0-9]|1[0-9]|2[0-3])) (\*|([1-9]|1[0-9]|2[0-9]|3[0-1])|\*\/([1-9]|1[0-9]|2[0-9]|3[0-1])) (\*|([1-9]|1[0-2])|\*\/([1-9]|1[0-2])) (\*|([0-6])|\*\/([0-6]))$' OR cron_expression IN ('@hourly', '@daily', '@weekly', '@monthly', '@yearly')),
    CONSTRAINT valid_dates CHECK (start_date IS NULL OR end_date IS NULL OR start_date < end_date)
);

-- Execution history table
CREATE TABLE executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    schedule_id UUID NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    execution_number SERIAL,
    
    -- Timing information
    scheduled_time TIMESTAMP WITH TIME ZONE NOT NULL,
    start_time TIMESTAMP WITH TIME ZONE,
    end_time TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    
    -- Execution details
    status execution_status NOT NULL DEFAULT 'pending',
    attempt_count INTEGER DEFAULT 1,
    is_manual_trigger BOOLEAN DEFAULT false,
    is_catch_up BOOLEAN DEFAULT false,
    
    -- Response information
    response_code INTEGER,
    response_body TEXT,
    response_headers JSONB,
    error_message TEXT,
    error_details JSONB,
    
    -- Metadata
    triggered_by VARCHAR(255),
    execution_context JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Performance metrics
    queue_time_ms INTEGER,
    network_time_ms INTEGER,
    processing_time_ms INTEGER
);

-- Execution locks to prevent overlaps
CREATE TABLE execution_locks (
    schedule_id UUID PRIMARY KEY REFERENCES schedules(id) ON DELETE CASCADE,
    execution_id UUID REFERENCES executions(id) ON DELETE CASCADE,
    locked_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    locked_until TIMESTAMP WITH TIME ZONE NOT NULL,
    lock_token UUID DEFAULT uuid_generate_v4()
);

-- Schedule metrics for performance tracking
CREATE TABLE schedule_metrics (
    schedule_id UUID PRIMARY KEY REFERENCES schedules(id) ON DELETE CASCADE,
    
    -- Execution counts
    total_executions INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    skipped_count INTEGER DEFAULT 0,
    timeout_count INTEGER DEFAULT 0,
    
    -- Performance metrics
    avg_duration_ms INTEGER,
    min_duration_ms INTEGER,
    max_duration_ms INTEGER,
    p95_duration_ms INTEGER,
    p99_duration_ms INTEGER,
    
    -- Success metrics
    success_rate DECIMAL(5,2),
    consecutive_failures INTEGER DEFAULT 0,
    consecutive_successes INTEGER DEFAULT 0,
    
    -- Timing
    last_success_at TIMESTAMP WITH TIME ZONE,
    last_failure_at TIMESTAMP WITH TIME ZONE,
    last_timeout_at TIMESTAMP WITH TIME ZONE,
    
    -- Health score (0-100)
    health_score INTEGER DEFAULT 100,
    
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Notification channels configuration
CREATE TABLE notification_channels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    channel_type notification_channel NOT NULL,
    
    -- Channel-specific configuration
    webhook_url TEXT,
    email_addresses TEXT[],
    slack_webhook TEXT,
    slack_channel VARCHAR(255),
    discord_webhook TEXT,
    
    -- Settings
    enabled BOOLEAN DEFAULT true,
    batch_notifications BOOLEAN DEFAULT false,
    batch_window_minutes INTEGER DEFAULT 5,
    min_severity severity_level DEFAULT 'low',
    
    -- Rate limiting
    max_notifications_per_hour INTEGER DEFAULT 100,
    current_hour_count INTEGER DEFAULT 0,
    current_hour_start TIMESTAMP WITH TIME ZONE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Schedule-to-notification channel mapping
CREATE TABLE schedule_notifications (
    schedule_id UUID REFERENCES schedules(id) ON DELETE CASCADE,
    channel_id UUID REFERENCES notification_channels(id) ON DELETE CASCADE,
    severity_threshold severity_level DEFAULT 'medium',
    PRIMARY KEY (schedule_id, channel_id)
);

-- Cron expression presets for common patterns
CREATE TABLE cron_presets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    expression VARCHAR(100) NOT NULL,
    category VARCHAR(50),
    is_system BOOLEAN DEFAULT false,
    usage_count INTEGER DEFAULT 0
);

-- Audit log for schedule changes
CREATE TABLE audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    schedule_id UUID REFERENCES schedules(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL, -- created, updated, deleted, enabled, disabled, triggered
    performed_by VARCHAR(255),
    changes JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_schedules_status ON schedules(status) WHERE enabled = true;
CREATE INDEX idx_schedules_next_execution ON schedules(next_execution_at) WHERE status = 'active' AND enabled = true;
CREATE INDEX idx_schedules_owner ON schedules(owner);
CREATE INDEX idx_schedules_tags ON schedules USING gin(tags);
CREATE INDEX idx_schedules_created_at ON schedules(created_at DESC);

CREATE INDEX idx_executions_schedule_id ON executions(schedule_id);
CREATE INDEX idx_executions_status ON executions(status);
CREATE INDEX idx_executions_scheduled_time ON executions(scheduled_time DESC);
CREATE INDEX idx_executions_created_at ON executions(created_at DESC);
CREATE INDEX idx_executions_schedule_status ON executions(schedule_id, status);

CREATE INDEX idx_metrics_health_score ON schedule_metrics(health_score);
CREATE INDEX idx_metrics_success_rate ON schedule_metrics(success_rate);

CREATE INDEX idx_audit_schedule_id ON audit_log(schedule_id);
CREATE INDEX idx_audit_created_at ON audit_log(created_at DESC);

-- Create views for common queries
CREATE OR REPLACE VIEW active_schedules AS
SELECT 
    s.*,
    m.success_rate,
    m.health_score,
    m.total_executions,
    m.last_success_at,
    m.last_failure_at
FROM schedules s
LEFT JOIN schedule_metrics m ON s.id = m.schedule_id
WHERE s.status = 'active' AND s.enabled = true
  AND (s.start_date IS NULL OR s.start_date <= CURRENT_TIMESTAMP)
  AND (s.end_date IS NULL OR s.end_date > CURRENT_TIMESTAMP);

CREATE OR REPLACE VIEW recent_executions AS
SELECT 
    e.*,
    s.name as schedule_name,
    s.owner as schedule_owner
FROM executions e
JOIN schedules s ON e.schedule_id = s.id
WHERE e.created_at > CURRENT_TIMESTAMP - INTERVAL '24 hours'
ORDER BY e.created_at DESC;

CREATE OR REPLACE VIEW schedule_health AS
SELECT 
    s.id,
    s.name,
    s.status,
    m.health_score,
    m.success_rate,
    m.consecutive_failures,
    m.last_failure_at,
    CASE 
        WHEN m.consecutive_failures >= 5 THEN 'critical'
        WHEN m.consecutive_failures >= 3 THEN 'warning'
        WHEN m.success_rate < 50 THEN 'degraded'
        ELSE 'healthy'
    END as health_status
FROM schedules s
LEFT JOIN schedule_metrics m ON s.id = m.schedule_id;

-- Trigger to update next_execution_at when schedule is saved
CREATE OR REPLACE FUNCTION update_next_execution()
RETURNS TRIGGER AS $$
BEGIN
    -- This would normally calculate based on cron expression
    -- Placeholder for actual cron calculation logic
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_next_execution
BEFORE INSERT OR UPDATE ON schedules
FOR EACH ROW
EXECUTE FUNCTION update_next_execution();

-- Trigger to update metrics after execution
CREATE OR REPLACE FUNCTION update_schedule_metrics()
RETURNS TRIGGER AS $$
DECLARE
    v_success_count INTEGER;
    v_failure_count INTEGER;
    v_total_count INTEGER;
    v_avg_duration INTEGER;
BEGIN
    -- Calculate metrics
    SELECT 
        COUNT(*) FILTER (WHERE status = 'success'),
        COUNT(*) FILTER (WHERE status = 'failed'),
        COUNT(*),
        AVG(duration_ms) FILTER (WHERE status = 'success')
    INTO v_success_count, v_failure_count, v_total_count, v_avg_duration
    FROM executions
    WHERE schedule_id = NEW.schedule_id;
    
    -- Update or insert metrics
    INSERT INTO schedule_metrics (
        schedule_id,
        total_executions,
        success_count,
        failure_count,
        avg_duration_ms,
        success_rate,
        updated_at
    ) VALUES (
        NEW.schedule_id,
        v_total_count,
        v_success_count,
        v_failure_count,
        v_avg_duration,
        CASE WHEN v_total_count > 0 THEN (v_success_count::DECIMAL / v_total_count * 100) ELSE 0 END,
        CURRENT_TIMESTAMP
    )
    ON CONFLICT (schedule_id) DO UPDATE SET
        total_executions = v_total_count,
        success_count = v_success_count,
        failure_count = v_failure_count,
        avg_duration_ms = v_avg_duration,
        success_rate = CASE WHEN v_total_count > 0 THEN (v_success_count::DECIMAL / v_total_count * 100) ELSE 0 END,
        updated_at = CURRENT_TIMESTAMP;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trigger_update_metrics
AFTER INSERT OR UPDATE ON executions
FOR EACH ROW
WHEN (NEW.status IN ('success', 'failed', 'timeout'))
EXECUTE FUNCTION update_schedule_metrics();

-- Function to clean up old executions
CREATE OR REPLACE FUNCTION cleanup_old_executions()
RETURNS INTEGER AS $$
DECLARE
    deleted_count INTEGER;
BEGIN
    DELETE FROM executions
    WHERE created_at < CURRENT_TIMESTAMP - INTERVAL '30 days'
      AND schedule_id NOT IN (
          SELECT id FROM schedules WHERE tags @> ARRAY['preserve-history']
      );
    
    GET DIAGNOSTICS deleted_count = ROW_COUNT;
    RETURN deleted_count;
END;
$$ LANGUAGE plpgsql;

-- Function to get next N execution times for a schedule
CREATE OR REPLACE FUNCTION get_next_runs(
    p_schedule_id UUID,
    p_count INTEGER DEFAULT 5
)
RETURNS TABLE(run_time TIMESTAMP WITH TIME ZONE) AS $$
BEGIN
    -- This would use cron expression parser
    -- Placeholder returning sample times
    RETURN QUERY
    SELECT generate_series(
        CURRENT_TIMESTAMP,
        CURRENT_TIMESTAMP + INTERVAL '7 days',
        INTERVAL '1 day'
    ) LIMIT p_count;
END;
$$ LANGUAGE plpgsql;

-- Comments for documentation
COMMENT ON TABLE schedules IS 'Core schedule definitions for workflow orchestration';
COMMENT ON TABLE executions IS 'Complete execution history with performance metrics';
COMMENT ON TABLE execution_locks IS 'Distributed locking mechanism to prevent execution overlaps';
COMMENT ON TABLE schedule_metrics IS 'Aggregated performance metrics for each schedule';
COMMENT ON TABLE notification_channels IS 'Configured channels for alert delivery';
COMMENT ON TABLE cron_presets IS 'Reusable cron expressions for common scheduling patterns';
COMMENT ON TABLE audit_log IS 'Complete audit trail of all schedule modifications';