-- Audio Tools Database Schema
-- Database structure for audio processing and analysis platform

-- Audio Assets table: Stores metadata about audio files
CREATE TABLE IF NOT EXISTS audio_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    file_path TEXT NOT NULL,
    format VARCHAR(20),
    duration_seconds DECIMAL(10,3),
    sample_rate INTEGER,
    bit_depth INTEGER,
    channels INTEGER,
    bitrate INTEGER,
    file_size_bytes BIGINT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_processed TIMESTAMP WITH TIME ZONE,
    quality_score DECIMAL(3,2),
    noise_level DECIMAL(5,2),
    speech_detected BOOLEAN DEFAULT FALSE,
    language VARCHAR(10),
    tags TEXT[]
);

-- Workflows table: Workflow definitions
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    platform VARCHAR(50) NOT NULL CHECK (platform IN ('n8n', 'windmill', 'node-red')),
    
    -- Workflow configuration
    definition JSONB NOT NULL,
    input_schema JSONB,
    output_schema JSONB,
    config JSONB DEFAULT '{}',
    
    -- Metadata
    version INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    tags TEXT[],
    
    -- Statistics
    execution_count INTEGER DEFAULT 0,
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    avg_duration_ms INTEGER,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Executions table: Workflow execution history
CREATE TABLE IF NOT EXISTS executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID REFERENCES workflows(id) ON DELETE CASCADE,
    
    -- Execution details
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    input_data JSONB,
    output_data JSONB,
    error_message TEXT,
    
    -- Performance metrics
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    
    -- Resource usage
    resource_usage JSONB DEFAULT '{}',
    
    -- Context
    triggered_by VARCHAR(255),
    correlation_id UUID,
    metadata JSONB DEFAULT '{}'
);

-- Events table: Application event log
CREATE TABLE IF NOT EXISTS events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type VARCHAR(100) NOT NULL,
    event_name VARCHAR(255) NOT NULL,
    
    -- Event data
    payload JSONB,
    source VARCHAR(100),
    target VARCHAR(100),
    
    -- Context
    resource_id UUID REFERENCES resources(id) ON DELETE CASCADE,
    execution_id UUID REFERENCES executions(id) ON DELETE CASCADE,
    user_id VARCHAR(255),
    session_id VARCHAR(255),
    
    -- Metadata
    severity VARCHAR(20) DEFAULT 'info',
    tags TEXT[],
    
    -- Timestamp
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Configuration table: Application settings
CREATE TABLE IF NOT EXISTS configuration (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(255) UNIQUE NOT NULL,
    value JSONB NOT NULL,
    
    -- Configuration metadata
    category VARCHAR(100),
    description TEXT,
    is_secret BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    
    -- Validation
    schema JSONB,
    default_value JSONB,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Metrics table: Performance and usage metrics
CREATE TABLE IF NOT EXISTS metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_name VARCHAR(255) NOT NULL,
    metric_type VARCHAR(50) NOT NULL,
    
    -- Metric data
    value NUMERIC NOT NULL,
    unit VARCHAR(50),
    
    -- Context
    resource_type VARCHAR(100),
    resource_id VARCHAR(255),
    
    -- Dimensions for grouping
    dimensions JSONB DEFAULT '{}',
    
    -- Timestamp
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Sessions table: User session management
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_token VARCHAR(255) UNIQUE NOT NULL,
    user_id VARCHAR(255),
    
    -- Session data
    data JSONB DEFAULT '{}',
    ip_address INET,
    user_agent TEXT,
    
    -- Status
    is_active BOOLEAN DEFAULT true,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_activity TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for performance
CREATE INDEX idx_resources_type ON resources(type) WHERE deleted_at IS NULL;
CREATE INDEX idx_resources_status ON resources(status) WHERE deleted_at IS NULL;
CREATE INDEX idx_resources_owner ON resources(owner_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_resources_created ON resources(created_at DESC);

CREATE INDEX idx_workflows_platform ON workflows(platform) WHERE is_active = true;
CREATE INDEX idx_workflows_tags ON workflows USING GIN(tags);
CREATE INDEX idx_workflows_active ON workflows(is_active);

CREATE INDEX idx_executions_workflow ON executions(workflow_id);
CREATE INDEX idx_executions_status ON executions(status);
CREATE INDEX idx_executions_started ON executions(started_at DESC);
CREATE INDEX idx_executions_correlation ON executions(correlation_id);

CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_resource ON events(resource_id);
CREATE INDEX idx_events_execution ON events(execution_id);
CREATE INDEX idx_events_created ON events(created_at DESC);
CREATE INDEX idx_events_tags ON events USING GIN(tags);

CREATE INDEX idx_configuration_category ON configuration(category) WHERE is_active = true;
CREATE INDEX idx_configuration_key ON configuration(key) WHERE is_active = true;

CREATE INDEX idx_metrics_name ON metrics(metric_name);
CREATE INDEX idx_metrics_timestamp ON metrics(timestamp DESC);
CREATE INDEX idx_metrics_resource ON metrics(resource_type, resource_id);

CREATE INDEX idx_sessions_token ON sessions(session_token) WHERE is_active = true;
CREATE INDEX idx_sessions_user ON sessions(user_id) WHERE is_active = true;
CREATE INDEX idx_sessions_activity ON sessions(last_activity DESC) WHERE is_active = true;

-- Update triggers for timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_resources_updated_at BEFORE UPDATE ON resources
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflows_updated_at BEFORE UPDATE ON workflows
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_configuration_updated_at BEFORE UPDATE ON configuration
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to update workflow statistics after execution
CREATE OR REPLACE FUNCTION update_workflow_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status IN ('success', 'failed') AND NEW.workflow_id IS NOT NULL THEN
        UPDATE workflows SET
            execution_count = execution_count + 1,
            success_count = success_count + CASE WHEN NEW.status = 'success' THEN 1 ELSE 0 END,
            failure_count = failure_count + CASE WHEN NEW.status = 'failed' THEN 1 ELSE 0 END,
            avg_duration_ms = CASE
                WHEN execution_count = 0 THEN NEW.duration_ms
                ELSE ((avg_duration_ms * execution_count + COALESCE(NEW.duration_ms, 0)) / (execution_count + 1))::INTEGER
            END,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = NEW.workflow_id;
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_workflow_stats_on_execution
    AFTER INSERT OR UPDATE OF status ON executions
    FOR EACH ROW
    WHEN (NEW.status IN ('success', 'failed'))
    EXECUTE FUNCTION update_workflow_stats();

-- Useful views for monitoring
CREATE OR REPLACE VIEW workflow_performance AS
SELECT 
    w.name,
    w.platform,
    w.execution_count,
    w.success_count,
    w.failure_count,
    CASE WHEN w.execution_count > 0 
         THEN (w.success_count::FLOAT / w.execution_count * 100)::NUMERIC(5,2)
         ELSE 0 
    END as success_rate,
    w.avg_duration_ms,
    w.is_active
FROM workflows w
ORDER BY w.execution_count DESC;

CREATE OR REPLACE VIEW recent_executions AS
SELECT 
    e.id,
    w.name as workflow_name,
    e.status,
    e.started_at,
    e.completed_at,
    e.duration_ms,
    e.error_message
FROM executions e
JOIN workflows w ON e.workflow_id = w.id
ORDER BY e.started_at DESC
LIMIT 100;

CREATE OR REPLACE VIEW resource_summary AS
SELECT 
    type,
    status,
    COUNT(*) as count,
    MAX(created_at) as latest_created,
    MAX(updated_at) as latest_updated
FROM resources
WHERE deleted_at IS NULL
GROUP BY type, status
ORDER BY type, status;