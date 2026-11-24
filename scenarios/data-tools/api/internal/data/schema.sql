-- SCENARIO_NAME_PLACEHOLDER Database Schema
-- Core database structure for API and application data

-- Resources table: Main entity management
CREATE TABLE IF NOT EXISTS resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(100),
    status VARCHAR(50) DEFAULT 'active',
    config JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    
    -- Relationships
    parent_id UUID REFERENCES resources(id) ON DELETE SET NULL,
    owner_id VARCHAR(255),
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Constraints
    CONSTRAINT unique_name_per_type UNIQUE(name, type)
);

-- Workflows table: Workflow definitions
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    platform VARCHAR(50) NOT NULL CHECK (platform IN ('n8n', 'node-red')),
    
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
    result JSONB DEFAULT '{}',
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
CREATE INDEX IF NOT EXISTS idx_resources_type ON resources(type) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_resources_status ON resources(status) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_resources_owner ON resources(owner_id) WHERE deleted_at IS NULL;
CREATE INDEX IF NOT EXISTS idx_resources_created ON resources(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_workflows_platform ON workflows(platform) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_workflows_tags ON workflows USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_workflows_active ON workflows(is_active);

CREATE INDEX IF NOT EXISTS idx_executions_workflow ON executions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);
CREATE INDEX IF NOT EXISTS idx_executions_started ON executions(started_at DESC);
CREATE INDEX IF NOT EXISTS idx_executions_correlation ON executions(correlation_id);

CREATE INDEX IF NOT EXISTS idx_events_type ON events(event_type);
CREATE INDEX IF NOT EXISTS idx_events_resource ON events(resource_id);
CREATE INDEX IF NOT EXISTS idx_events_execution ON events(execution_id);
CREATE INDEX IF NOT EXISTS idx_events_created ON events(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_events_tags ON events USING GIN(tags);

CREATE INDEX IF NOT EXISTS idx_configuration_category ON configuration(category) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_configuration_key ON configuration(key) WHERE is_active = true;

CREATE INDEX IF NOT EXISTS idx_metrics_name ON metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_metrics_timestamp ON metrics(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_metrics_resource ON metrics(resource_type, resource_id);

CREATE INDEX IF NOT EXISTS idx_sessions_token ON sessions(session_token) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id) WHERE is_active = true;
CREATE INDEX IF NOT EXISTS idx_sessions_activity ON sessions(last_activity DESC) WHERE is_active = true;

-- Update triggers for timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_resources_updated_at ON resources;
CREATE TRIGGER update_resources_updated_at BEFORE UPDATE ON resources
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_workflows_updated_at ON workflows;
CREATE TRIGGER update_workflows_updated_at BEFORE UPDATE ON workflows
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_configuration_updated_at ON configuration;
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

DROP TRIGGER IF EXISTS update_workflow_stats_on_execution ON executions;
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
-- Datasets table: Main data catalog
CREATE TABLE IF NOT EXISTS datasets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    schema_definition JSONB NOT NULL DEFAULT '{}',
    format VARCHAR(50) CHECK (format IN ('csv', 'json', 'xml', 'excel', 'parquet', 'avro', 'custom')),
    size_bytes BIGINT DEFAULT 0,
    row_count BIGINT DEFAULT 0,
    column_count INTEGER DEFAULT 0,
    quality_score DECIMAL(5,4) DEFAULT 0.0,
    minio_path VARCHAR(500),
    
    -- Metadata
    tags TEXT[],
    metadata JSONB DEFAULT '{}',
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_accessed TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Data Transformations table
CREATE TABLE IF NOT EXISTS data_transformations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dataset_id UUID REFERENCES datasets(id) ON DELETE CASCADE,
    transformation_type VARCHAR(50) CHECK (transformation_type IN ('filter', 'map', 'join', 'aggregate', 'pivot', 'sort', 'custom')),
    parameters JSONB NOT NULL DEFAULT '{}',
    sql_query TEXT,
    input_schema JSONB DEFAULT '{}',
    output_schema JSONB DEFAULT '{}',
    
    -- Performance metrics
    execution_time_ms INTEGER,
    rows_processed BIGINT,
    success BOOLEAN DEFAULT true,
    error_message TEXT,
    
    -- Audit
    created_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    -- Execution stats
    execution_stats JSONB DEFAULT '{}'
);

-- Data Quality Reports table
CREATE TABLE IF NOT EXISTS data_quality_reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    dataset_id UUID REFERENCES datasets(id) ON DELETE CASCADE,
    
    -- Quality scores
    completeness_score DECIMAL(5,4) DEFAULT 0.0,
    accuracy_score DECIMAL(5,4) DEFAULT 0.0,
    consistency_score DECIMAL(5,4) DEFAULT 0.0,
    validity_score DECIMAL(5,4) DEFAULT 0.0,
    
    -- Issues found
    anomalies_detected JSONB DEFAULT '[]',
    duplicate_count BIGINT DEFAULT 0,
    null_percentage DECIMAL(5,4) DEFAULT 0.0,
    
    -- Recommendations
    recommendations JSONB DEFAULT '[]',
    
    -- Timestamp
    generated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Streaming Sources table
CREATE TABLE IF NOT EXISTS streaming_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    source_type VARCHAR(50) CHECK (source_type IN ('kafka', 'webhook', 'file_watch', 'database_cdc')),
    connection_config JSONB NOT NULL DEFAULT '{}',
    schema_definition JSONB DEFAULT '{}',
    processing_rules JSONB DEFAULT '[]',
    
    -- Status
    is_active BOOLEAN DEFAULT false,
    last_message_at TIMESTAMP WITH TIME ZONE,
    message_count BIGINT DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    throughput_per_sec DECIMAL(10,2) DEFAULT 0.0,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Query Cache table for performance
CREATE TABLE IF NOT EXISTS query_cache (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    query_hash VARCHAR(64) UNIQUE NOT NULL,
    query_text TEXT NOT NULL,
    result_data JSONB,
    result_count INTEGER,
    execution_time_ms INTEGER,
    
    -- Cache management
    hit_count INTEGER DEFAULT 0,
    last_accessed TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Data Lineage table for tracking data flow
CREATE TABLE IF NOT EXISTS data_lineage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_dataset_id UUID REFERENCES datasets(id) ON DELETE CASCADE,
    target_dataset_id UUID REFERENCES datasets(id) ON DELETE CASCADE,
    transformation_id UUID REFERENCES data_transformations(id) ON DELETE CASCADE,
    
    -- Lineage details
    operation_type VARCHAR(50),
    operation_details JSONB DEFAULT '{}',
    
    -- Impact analysis
    rows_affected BIGINT,
    columns_affected TEXT[],
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_datasets_name ON datasets(name);
CREATE INDEX IF NOT EXISTS idx_datasets_format ON datasets(format);
CREATE INDEX IF NOT EXISTS idx_datasets_tags ON datasets USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_datasets_quality ON datasets(quality_score DESC);
CREATE INDEX IF NOT EXISTS idx_datasets_accessed ON datasets(last_accessed DESC);

CREATE INDEX IF NOT EXISTS idx_transformations_dataset ON data_transformations(dataset_id);
CREATE INDEX IF NOT EXISTS idx_transformations_type ON data_transformations(transformation_type);
CREATE INDEX IF NOT EXISTS idx_transformations_created ON data_transformations(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_quality_dataset ON data_quality_reports(dataset_id);
CREATE INDEX IF NOT EXISTS idx_quality_generated ON data_quality_reports(generated_at DESC);

CREATE INDEX IF NOT EXISTS idx_streaming_type ON streaming_sources(source_type);
CREATE INDEX IF NOT EXISTS idx_streaming_active ON streaming_sources(is_active);

CREATE INDEX IF NOT EXISTS idx_cache_hash ON query_cache(query_hash);
CREATE INDEX IF NOT EXISTS idx_cache_accessed ON query_cache(last_accessed DESC);
CREATE INDEX IF NOT EXISTS idx_cache_expires ON query_cache(expires_at) WHERE expires_at IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_lineage_source ON data_lineage(source_dataset_id);
CREATE INDEX IF NOT EXISTS idx_lineage_target ON data_lineage(target_dataset_id);
CREATE INDEX IF NOT EXISTS idx_lineage_transformation ON data_lineage(transformation_id);

CREATE INDEX IF NOT EXISTS idx_resources_name ON resources(name);
CREATE INDEX IF NOT EXISTS idx_resources_created ON resources(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);

-- Update triggers for timestamps
DROP TRIGGER IF EXISTS update_datasets_updated_at ON datasets;
CREATE TRIGGER update_datasets_updated_at BEFORE UPDATE ON datasets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_streaming_sources_updated_at ON streaming_sources;
CREATE TRIGGER update_streaming_sources_updated_at BEFORE UPDATE ON streaming_sources
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_resources_updated_at ON resources;
CREATE TRIGGER update_resources_updated_at BEFORE UPDATE ON resources
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to update dataset access time
CREATE OR REPLACE FUNCTION update_dataset_accessed()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE datasets SET last_accessed = CURRENT_TIMESTAMP WHERE id = NEW.dataset_id;
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_dataset_accessed_on_transformation ON data_transformations;
CREATE TRIGGER update_dataset_accessed_on_transformation
    AFTER INSERT ON data_transformations
    FOR EACH ROW
    EXECUTE FUNCTION update_dataset_accessed();

-- Function to update query cache hit count
CREATE OR REPLACE FUNCTION update_cache_hit()
RETURNS TRIGGER AS $$
BEGIN
    NEW.hit_count = OLD.hit_count + 1;
    NEW.last_accessed = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Useful views for monitoring
CREATE OR REPLACE VIEW dataset_overview AS
SELECT 
    d.name,
    d.format,
    d.row_count,
    d.column_count,
    d.quality_score,
    d.size_bytes,
    COUNT(DISTINCT dt.id) as transformation_count,
    MAX(dt.created_at) as last_transformed,
    d.last_accessed
FROM datasets d
LEFT JOIN data_transformations dt ON d.id = dt.dataset_id
GROUP BY d.id, d.name, d.format, d.row_count, d.column_count, d.quality_score, d.size_bytes, d.last_accessed
ORDER BY d.last_accessed DESC;

CREATE OR REPLACE VIEW transformation_performance AS
SELECT 
    dt.transformation_type,
    COUNT(*) as execution_count,
    AVG(dt.execution_time_ms) as avg_execution_ms,
    MAX(dt.execution_time_ms) as max_execution_ms,
    SUM(dt.rows_processed) as total_rows_processed,
    COUNT(CASE WHEN dt.success THEN 1 END) as success_count,
    COUNT(CASE WHEN NOT dt.success THEN 1 END) as failure_count
FROM data_transformations dt
GROUP BY dt.transformation_type
ORDER BY execution_count DESC;

CREATE OR REPLACE VIEW streaming_status AS
SELECT 
    ss.name,
    ss.source_type,
    ss.is_active,
    ss.message_count,
    ss.error_count,
    ss.throughput_per_sec,
    ss.last_message_at,
    CASE 
        WHEN ss.is_active AND ss.last_message_at > (CURRENT_TIMESTAMP - INTERVAL '5 minutes') THEN 'healthy'
        WHEN ss.is_active AND ss.last_message_at > (CURRENT_TIMESTAMP - INTERVAL '30 minutes') THEN 'warning'
        WHEN ss.is_active THEN 'error'
        ELSE 'inactive'
    END as health_status
FROM streaming_sources ss
ORDER BY ss.is_active DESC, ss.last_message_at DESC;
