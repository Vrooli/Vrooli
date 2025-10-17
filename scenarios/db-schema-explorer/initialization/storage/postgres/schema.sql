-- Database Schema Explorer - PostgreSQL Schema
-- This schema stores metadata about explored databases, query history, and visualization layouts

-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS db_explorer;

-- Set search path
SET search_path TO db_explorer, public;

-- Schema snapshots table
CREATE TABLE IF NOT EXISTS schema_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    database_name VARCHAR(255) NOT NULL,
    schema_name VARCHAR(255) DEFAULT 'public',
    snapshot_timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    tables_count INTEGER DEFAULT 0,
    columns_count INTEGER DEFAULT 0,
    relationships_count INTEGER DEFAULT 0,
    indexes_count INTEGER DEFAULT 0,
    schema_data JSONB NOT NULL,
    version VARCHAR(50),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index for faster lookups
CREATE INDEX idx_schema_snapshots_database ON schema_snapshots(database_name, schema_name);
CREATE INDEX idx_schema_snapshots_timestamp ON schema_snapshots(snapshot_timestamp DESC);

-- Query history table (metadata only, embeddings in Qdrant)
CREATE TABLE IF NOT EXISTS query_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    natural_language TEXT NOT NULL,
    generated_sql TEXT NOT NULL,
    database_name VARCHAR(255) NOT NULL,
    schema_name VARCHAR(255) DEFAULT 'public',
    execution_time_ms INTEGER,
    result_count INTEGER,
    query_type VARCHAR(50),
    tables_used TEXT[],
    user_feedback VARCHAR(20), -- 'helpful', 'not_helpful', null
    confidence_score INTEGER,
    qdrant_vector_id UUID, -- Reference to vector in Qdrant
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    executed_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for query history
CREATE INDEX idx_query_history_database ON query_history(database_name);
CREATE INDEX idx_query_history_created ON query_history(created_at DESC);
CREATE INDEX idx_query_history_feedback ON query_history(user_feedback);
CREATE INDEX idx_query_history_type ON query_history(query_type);

-- Visualization layouts table
CREATE TABLE IF NOT EXISTS visualization_layouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    database_name VARCHAR(255) NOT NULL,
    schema_name VARCHAR(255) DEFAULT 'public',
    layout_type VARCHAR(50) DEFAULT 'standard', -- 'compact', 'detailed', 'relationship'
    layout_data JSONB NOT NULL, -- Stores positions, zoom, selected tables, etc.
    is_default BOOLEAN DEFAULT FALSE,
    is_shared BOOLEAN DEFAULT FALSE,
    created_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for layouts
CREATE INDEX idx_visualization_layouts_database ON visualization_layouts(database_name, schema_name);
CREATE INDEX idx_visualization_layouts_shared ON visualization_layouts(is_shared);

-- Database connections table (stores metadata about explored databases)
CREATE TABLE IF NOT EXISTS database_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    connection_name VARCHAR(255) NOT NULL UNIQUE,
    database_type VARCHAR(50) DEFAULT 'postgres',
    host VARCHAR(255),
    port INTEGER,
    database_name VARCHAR(255),
    schema_name VARCHAR(255) DEFAULT 'public',
    is_active BOOLEAN DEFAULT TRUE,
    last_accessed TIMESTAMP WITH TIME ZONE,
    metadata JSONB, -- Additional connection metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index for active connections
CREATE INDEX idx_database_connections_active ON database_connections(is_active);

-- Schema analysis results table
CREATE TABLE IF NOT EXISTS schema_analysis (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    snapshot_id UUID REFERENCES schema_snapshots(id) ON DELETE CASCADE,
    analysis_type VARCHAR(100) NOT NULL, -- 'consistency', 'performance', 'normalization'
    findings JSONB NOT NULL,
    severity VARCHAR(20), -- 'info', 'warning', 'critical'
    recommendations JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index for analysis lookups
CREATE INDEX idx_schema_analysis_snapshot ON schema_analysis(snapshot_id);
CREATE INDEX idx_schema_analysis_type ON schema_analysis(analysis_type);
CREATE INDEX idx_schema_analysis_severity ON schema_analysis(severity);

-- Query patterns table (for learning common patterns)
CREATE TABLE IF NOT EXISTS query_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_name VARCHAR(255) NOT NULL,
    pattern_type VARCHAR(100), -- 'aggregation', 'join', 'filter', etc.
    pattern_template TEXT NOT NULL,
    usage_count INTEGER DEFAULT 0,
    success_rate DECIMAL(5,2),
    average_execution_time_ms INTEGER,
    tables_involved TEXT[],
    example_queries JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index for pattern lookups
CREATE INDEX idx_query_patterns_type ON query_patterns(pattern_type);
CREATE INDEX idx_query_patterns_usage ON query_patterns(usage_count DESC);

-- Performance metrics table
CREATE TABLE IF NOT EXISTS performance_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    query_id UUID REFERENCES query_history(id) ON DELETE CASCADE,
    execution_plan JSONB,
    actual_time_ms DECIMAL(10,2),
    planning_time_ms DECIMAL(10,2),
    rows_examined INTEGER,
    rows_returned INTEGER,
    index_usage JSONB,
    optimization_suggestions JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Index for performance lookups
CREATE INDEX idx_performance_metrics_query ON performance_metrics(query_id);
CREATE INDEX idx_performance_metrics_time ON performance_metrics(actual_time_ms);

-- Create update trigger for updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply update trigger to tables with updated_at
CREATE TRIGGER update_schema_snapshots_updated_at BEFORE UPDATE ON schema_snapshots
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_visualization_layouts_updated_at BEFORE UPDATE ON visualization_layouts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_database_connections_updated_at BEFORE UPDATE ON database_connections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_query_patterns_updated_at BEFORE UPDATE ON query_patterns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create view for query statistics
CREATE OR REPLACE VIEW query_statistics AS
SELECT 
    database_name,
    query_type,
    COUNT(*) as total_queries,
    AVG(execution_time_ms) as avg_execution_time,
    MAX(execution_time_ms) as max_execution_time,
    MIN(execution_time_ms) as min_execution_time,
    AVG(result_count) as avg_result_count,
    SUM(CASE WHEN user_feedback = 'helpful' THEN 1 ELSE 0 END) as helpful_count,
    SUM(CASE WHEN user_feedback = 'not_helpful' THEN 1 ELSE 0 END) as not_helpful_count,
    DATE_TRUNC('day', created_at) as query_date
FROM query_history
GROUP BY database_name, query_type, DATE_TRUNC('day', created_at);

-- Create view for active database summary
CREATE OR REPLACE VIEW database_summary AS
SELECT 
    dc.connection_name,
    dc.database_name,
    dc.last_accessed,
    ss.tables_count,
    ss.columns_count,
    ss.relationships_count,
    ss.snapshot_timestamp as last_snapshot,
    COUNT(DISTINCT qh.id) as total_queries,
    COUNT(DISTINCT vl.id) as saved_layouts
FROM database_connections dc
LEFT JOIN schema_snapshots ss ON ss.database_name = dc.database_name
    AND ss.snapshot_timestamp = (
        SELECT MAX(snapshot_timestamp) 
        FROM schema_snapshots 
        WHERE database_name = dc.database_name
    )
LEFT JOIN query_history qh ON qh.database_name = dc.database_name
LEFT JOIN visualization_layouts vl ON vl.database_name = dc.database_name
WHERE dc.is_active = TRUE
GROUP BY dc.connection_name, dc.database_name, dc.last_accessed, 
         ss.tables_count, ss.columns_count, ss.relationships_count, ss.snapshot_timestamp;

-- Grant permissions (adjust as needed)
GRANT ALL PRIVILEGES ON SCHEMA db_explorer TO PUBLIC;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA db_explorer TO PUBLIC;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA db_explorer TO PUBLIC;