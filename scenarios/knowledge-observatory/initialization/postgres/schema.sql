-- Knowledge Observatory Schema
-- Stores metadata, quality metrics, and search history for knowledge monitoring

CREATE SCHEMA IF NOT EXISTS knowledge_observatory;

-- Quality metrics for collections
CREATE TABLE IF NOT EXISTS knowledge_observatory.quality_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    collection_name VARCHAR(255) NOT NULL,
    coherence_score DECIMAL(3,2) CHECK (coherence_score >= 0 AND coherence_score <= 1),
    freshness_score DECIMAL(3,2) CHECK (freshness_score >= 0 AND freshness_score <= 1),
    redundancy_score DECIMAL(3,2) CHECK (redundancy_score >= 0 AND redundancy_score <= 1),
    coverage_score DECIMAL(3,2) CHECK (coverage_score >= 0 AND coverage_score <= 1),
    total_entries INTEGER DEFAULT 0,
    avg_quality DECIMAL(3,2) GENERATED ALWAYS AS (
        (coherence_score + freshness_score + redundancy_score + coverage_score) / 4
    ) STORED,
    measured_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Search history for analytics
CREATE TABLE IF NOT EXISTS knowledge_observatory.search_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    query TEXT NOT NULL,
    collection VARCHAR(255),
    result_count INTEGER DEFAULT 0,
    avg_score DECIMAL(3,2),
    response_time_ms INTEGER,
    user_session VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Ingest history for auditing and replay
CREATE TABLE IF NOT EXISTS knowledge_observatory.ingest_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    record_id VARCHAR(255) NOT NULL,
    namespace VARCHAR(255) NOT NULL,
    collection_name VARCHAR(255) NOT NULL,
    content_hash VARCHAR(64),
    visibility VARCHAR(20) NOT NULL CHECK (visibility IN ('private', 'shared', 'global')),
    source TEXT,
    source_type VARCHAR(100),
    status VARCHAR(20) NOT NULL CHECK (status IN ('success', 'failure')),
    error_message TEXT,
    took_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Async ingest jobs
CREATE TABLE IF NOT EXISTS knowledge_observatory.ingest_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    request_json JSONB NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'running', 'success', 'failure')),
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP WITH TIME ZONE,
    finished_at TIMESTAMP WITH TIME ZONE,
    total_chunks INTEGER DEFAULT 0,
    completed_chunks INTEGER DEFAULT 0
);

-- Knowledge entries metadata (cached from Qdrant)
CREATE TABLE IF NOT EXISTS knowledge_observatory.knowledge_metadata (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    vector_id VARCHAR(255) UNIQUE NOT NULL,
    collection_name VARCHAR(255) NOT NULL,
    content_hash VARCHAR(64),
    source_scenario VARCHAR(255),
    source_type VARCHAR(100),
    quality_score DECIMAL(3,2),
    access_count INTEGER DEFAULT 0,
    last_accessed TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Alerts and notifications
CREATE TABLE IF NOT EXISTS knowledge_observatory.alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    level VARCHAR(20) NOT NULL CHECK (level IN ('info', 'warning', 'critical')),
    collection_name VARCHAR(255),
    metric_name VARCHAR(100),
    threshold_value DECIMAL(3,2),
    actual_value DECIMAL(3,2),
    message TEXT,
    acknowledged BOOLEAN DEFAULT FALSE,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Knowledge relationships (for graph visualization)
CREATE TABLE IF NOT EXISTS knowledge_observatory.knowledge_relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id VARCHAR(255) NOT NULL,
    target_id VARCHAR(255) NOT NULL,
    relationship_type VARCHAR(100) NOT NULL,
    weight DECIMAL(3,2) DEFAULT 0.5,
    discovered_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(source_id, target_id, relationship_type)
);

-- Collection statistics
CREATE TABLE IF NOT EXISTS knowledge_observatory.collection_stats (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    collection_name VARCHAR(255) UNIQUE NOT NULL,
    total_entries INTEGER DEFAULT 0,
    total_searches INTEGER DEFAULT 0,
    avg_search_score DECIMAL(3,2),
    most_searched_terms JSONB,
    growth_rate DECIMAL(5,2),
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- User preferences and saved queries
CREATE TABLE IF NOT EXISTS knowledge_observatory.user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) UNIQUE,
    default_collection VARCHAR(255),
    saved_queries JSONB,
    dashboard_layout JSONB,
    alert_preferences JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_quality_metrics_collection ON knowledge_observatory.quality_metrics(collection_name);
CREATE INDEX idx_quality_metrics_measured_at ON knowledge_observatory.quality_metrics(measured_at DESC);
CREATE INDEX idx_search_history_query ON knowledge_observatory.search_history USING gin(to_tsvector('english', query));
CREATE INDEX idx_search_history_created ON knowledge_observatory.search_history(created_at DESC);
CREATE INDEX idx_ingest_history_created ON knowledge_observatory.ingest_history(created_at DESC);
CREATE INDEX idx_ingest_history_namespace ON knowledge_observatory.ingest_history(namespace);
CREATE INDEX idx_ingest_history_record_id ON knowledge_observatory.ingest_history(record_id);
CREATE INDEX idx_ingest_jobs_created ON knowledge_observatory.ingest_jobs(created_at DESC);
CREATE INDEX idx_ingest_jobs_status ON knowledge_observatory.ingest_jobs(status);
CREATE INDEX idx_knowledge_metadata_collection ON knowledge_observatory.knowledge_metadata(collection_name);
CREATE INDEX idx_knowledge_metadata_source ON knowledge_observatory.knowledge_metadata(source_scenario);
CREATE INDEX idx_alerts_level ON knowledge_observatory.alerts(level) WHERE NOT acknowledged;
CREATE INDEX idx_relationships_source ON knowledge_observatory.knowledge_relationships(source_id);
CREATE INDEX idx_relationships_target ON knowledge_observatory.knowledge_relationships(target_id);

-- Create update trigger for updated_at columns
CREATE OR REPLACE FUNCTION knowledge_observatory.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_quality_metrics_updated_at BEFORE UPDATE
    ON knowledge_observatory.quality_metrics FOR EACH ROW
    EXECUTE FUNCTION knowledge_observatory.update_updated_at_column();

CREATE TRIGGER update_knowledge_metadata_updated_at BEFORE UPDATE
    ON knowledge_observatory.knowledge_metadata FOR EACH ROW
    EXECUTE FUNCTION knowledge_observatory.update_updated_at_column();

CREATE TRIGGER update_user_preferences_updated_at BEFORE UPDATE
    ON knowledge_observatory.user_preferences FOR EACH ROW
    EXECUTE FUNCTION knowledge_observatory.update_updated_at_column();

-- Create view for dashboard metrics
CREATE OR REPLACE VIEW knowledge_observatory.dashboard_metrics AS
SELECT 
    c.collection_name,
    c.total_entries,
    q.coherence_score,
    q.freshness_score,
    q.redundancy_score,
    q.coverage_score,
    q.avg_quality,
    c.total_searches,
    c.avg_search_score,
    q.measured_at
FROM knowledge_observatory.collection_stats c
LEFT JOIN LATERAL (
    SELECT * FROM knowledge_observatory.quality_metrics
    WHERE collection_name = c.collection_name
    ORDER BY measured_at DESC
    LIMIT 1
) q ON true;

-- Grant permissions
GRANT USAGE ON SCHEMA knowledge_observatory TO PUBLIC;
GRANT SELECT, INSERT, UPDATE ON ALL TABLES IN SCHEMA knowledge_observatory TO PUBLIC;
GRANT USAGE ON ALL SEQUENCES IN SCHEMA knowledge_observatory TO PUBLIC;
