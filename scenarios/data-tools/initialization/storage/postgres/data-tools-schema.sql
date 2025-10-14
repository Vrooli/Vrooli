-- Data Tools Database Schema
-- Core database structure for data processing and transformation

-- Resources table: Generic resources for extensibility
CREATE TABLE IF NOT EXISTS resources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    config JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Executions table: Track workflow executions
CREATE TABLE IF NOT EXISTS executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    result JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

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
CREATE INDEX idx_datasets_name ON datasets(name);
CREATE INDEX idx_datasets_format ON datasets(format);
CREATE INDEX idx_datasets_tags ON datasets USING GIN(tags);
CREATE INDEX idx_datasets_quality ON datasets(quality_score DESC);
CREATE INDEX idx_datasets_accessed ON datasets(last_accessed DESC);

CREATE INDEX idx_transformations_dataset ON data_transformations(dataset_id);
CREATE INDEX idx_transformations_type ON data_transformations(transformation_type);
CREATE INDEX idx_transformations_created ON data_transformations(created_at DESC);

CREATE INDEX idx_quality_dataset ON data_quality_reports(dataset_id);
CREATE INDEX idx_quality_generated ON data_quality_reports(generated_at DESC);

CREATE INDEX idx_streaming_type ON streaming_sources(source_type);
CREATE INDEX idx_streaming_active ON streaming_sources(is_active);

CREATE INDEX idx_cache_hash ON query_cache(query_hash);
CREATE INDEX idx_cache_accessed ON query_cache(last_accessed DESC);
CREATE INDEX idx_cache_expires ON query_cache(expires_at) WHERE expires_at IS NOT NULL;

CREATE INDEX idx_lineage_source ON data_lineage(source_dataset_id);
CREATE INDEX idx_lineage_target ON data_lineage(target_dataset_id);
CREATE INDEX idx_lineage_transformation ON data_lineage(transformation_id);

CREATE INDEX idx_resources_name ON resources(name);
CREATE INDEX idx_resources_created ON resources(created_at DESC);

CREATE INDEX idx_executions_status ON executions(status);
CREATE INDEX idx_executions_created ON executions(created_at DESC);

-- Update triggers for timestamps
CREATE TRIGGER update_datasets_updated_at BEFORE UPDATE ON datasets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_streaming_sources_updated_at BEFORE UPDATE ON streaming_sources
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

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