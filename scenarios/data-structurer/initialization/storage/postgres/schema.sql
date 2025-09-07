-- Data Structurer Schema Definition
-- Creates tables for managing schemas and processed data

-- Create extension for UUID generation if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Schemas table: Stores user-defined data schemas
CREATE TABLE IF NOT EXISTS schemas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    schema_definition JSONB NOT NULL,
    example_data JSONB,
    version INTEGER NOT NULL DEFAULT 1,
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255),
    -- Validation constraints
    CONSTRAINT valid_schema_name CHECK (name ~ '^[a-z0-9_-]+$'),
    CONSTRAINT valid_version CHECK (version > 0)
);

-- Create index for faster schema lookups
CREATE INDEX IF NOT EXISTS idx_schemas_name ON schemas(name);
CREATE INDEX IF NOT EXISTS idx_schemas_active ON schemas(is_active);
CREATE INDEX IF NOT EXISTS idx_schemas_created_at ON schemas(created_at DESC);

-- Processed Data table: Stores results of data processing
CREATE TABLE IF NOT EXISTS processed_data (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    schema_id UUID NOT NULL REFERENCES schemas(id) ON DELETE CASCADE,
    source_file_name VARCHAR(500),
    source_file_path TEXT,
    source_file_type VARCHAR(50),
    source_file_size BIGINT,
    raw_content TEXT,
    structured_data JSONB NOT NULL,
    confidence_score FLOAT CHECK (confidence_score >= 0.0 AND confidence_score <= 1.0),
    processing_status VARCHAR(20) NOT NULL DEFAULT 'pending',
    error_message TEXT,
    processing_time_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP WITH TIME ZONE,
    metadata JSONB DEFAULT '{}',
    -- Validation constraints  
    CONSTRAINT valid_processing_status CHECK (processing_status IN ('pending', 'processing', 'completed', 'failed', 'retrying')),
    CONSTRAINT valid_confidence_score CHECK (confidence_score IS NULL OR (confidence_score >= 0.0 AND confidence_score <= 1.0))
);

-- Create indexes for efficient querying
CREATE INDEX IF NOT EXISTS idx_processed_data_schema_id ON processed_data(schema_id);
CREATE INDEX IF NOT EXISTS idx_processed_data_status ON processed_data(processing_status);
CREATE INDEX IF NOT EXISTS idx_processed_data_created_at ON processed_data(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_processed_data_confidence ON processed_data(confidence_score DESC) WHERE confidence_score IS NOT NULL;

-- GIN index for JSONB searching
CREATE INDEX IF NOT EXISTS idx_schemas_definition_gin ON schemas USING GIN (schema_definition);
CREATE INDEX IF NOT EXISTS idx_processed_data_structured_gin ON processed_data USING GIN (structured_data);
CREATE INDEX IF NOT EXISTS idx_processed_data_metadata_gin ON processed_data USING GIN (metadata);

-- Processing Jobs table: Tracks async processing jobs
CREATE TABLE IF NOT EXISTS processing_jobs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    schema_id UUID NOT NULL REFERENCES schemas(id) ON DELETE CASCADE,
    input_type VARCHAR(20) NOT NULL,
    input_data TEXT, -- File path or raw text
    batch_mode BOOLEAN NOT NULL DEFAULT false,
    total_items INTEGER,
    processed_items INTEGER DEFAULT 0,
    failed_items INTEGER DEFAULT 0,
    status VARCHAR(20) NOT NULL DEFAULT 'queued',
    priority INTEGER NOT NULL DEFAULT 5,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    error_details JSONB,
    result_summary JSONB,
    -- Validation constraints
    CONSTRAINT valid_job_status CHECK (status IN ('queued', 'running', 'completed', 'failed', 'cancelled')),
    CONSTRAINT valid_input_type CHECK (input_type IN ('file', 'text', 'url', 'batch')),
    CONSTRAINT valid_priority CHECK (priority >= 1 AND priority <= 10),
    CONSTRAINT valid_items_count CHECK (processed_items >= 0 AND failed_items >= 0)
);

-- Create indexes for job queue management
CREATE INDEX IF NOT EXISTS idx_processing_jobs_status ON processing_jobs(status);
CREATE INDEX IF NOT EXISTS idx_processing_jobs_priority ON processing_jobs(priority DESC, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_processing_jobs_schema_id ON processing_jobs(schema_id);

-- Schema Templates table: Pre-built schema templates for common use cases
CREATE TABLE IF NOT EXISTS schema_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    category VARCHAR(100) NOT NULL,
    description TEXT NOT NULL,
    schema_definition JSONB NOT NULL,
    example_data JSONB NOT NULL,
    usage_count INTEGER DEFAULT 0,
    is_public BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    tags TEXT[] DEFAULT '{}',
    -- Validation constraints
    CONSTRAINT valid_template_name CHECK (name ~ '^[a-z0-9_-]+$')
);

-- Create indexes for template discovery
CREATE INDEX IF NOT EXISTS idx_schema_templates_category ON schema_templates(category);
CREATE INDEX IF NOT EXISTS idx_schema_templates_public ON schema_templates(is_public);
CREATE INDEX IF NOT EXISTS idx_schema_templates_usage ON schema_templates(usage_count DESC);
CREATE INDEX IF NOT EXISTS idx_schema_templates_tags ON schema_templates USING GIN (tags);

-- Function to update the updated_at timestamp automatically
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers to auto-update timestamps
DROP TRIGGER IF EXISTS update_schemas_updated_at ON schemas;
CREATE TRIGGER update_schemas_updated_at 
    BEFORE UPDATE ON schemas 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_schema_templates_updated_at ON schema_templates;
CREATE TRIGGER update_schema_templates_updated_at 
    BEFORE UPDATE ON schema_templates 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Views for common queries

-- Active schemas view
CREATE OR REPLACE VIEW active_schemas AS
SELECT 
    id,
    name,
    description,
    schema_definition,
    example_data,
    version,
    created_at,
    updated_at,
    (SELECT COUNT(*) FROM processed_data pd WHERE pd.schema_id = s.id) as usage_count,
    (SELECT AVG(confidence_score) FROM processed_data pd WHERE pd.schema_id = s.id AND pd.confidence_score IS NOT NULL) as avg_confidence
FROM schemas s 
WHERE is_active = true;

-- Processing summary view
CREATE OR REPLACE VIEW processing_summary AS
SELECT 
    s.id as schema_id,
    s.name as schema_name,
    COUNT(pd.id) as total_processed,
    COUNT(CASE WHEN pd.processing_status = 'completed' THEN 1 END) as completed_count,
    COUNT(CASE WHEN pd.processing_status = 'failed' THEN 1 END) as failed_count,
    COUNT(CASE WHEN pd.processing_status = 'processing' THEN 1 END) as processing_count,
    AVG(pd.confidence_score) as avg_confidence,
    AVG(pd.processing_time_ms) as avg_processing_time_ms,
    MAX(pd.created_at) as last_processed_at
FROM schemas s
LEFT JOIN processed_data pd ON s.id = pd.schema_id
GROUP BY s.id, s.name;

-- Grant appropriate permissions (adjust as needed for your security model)
-- These grants assume a 'vrooli_app' role exists
-- GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO vrooli_app;
-- GRANT SELECT ON ALL VIEWS IN SCHEMA public TO vrooli_app;
-- GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO vrooli_app;

-- Comments for documentation
COMMENT ON TABLE schemas IS 'User-defined data schemas for structuring unstructured data';
COMMENT ON TABLE processed_data IS 'Results of processing unstructured data through schemas';
COMMENT ON TABLE processing_jobs IS 'Async job queue for batch and long-running processing tasks';
COMMENT ON TABLE schema_templates IS 'Pre-built schema templates for common data types';

COMMENT ON COLUMN schemas.schema_definition IS 'JSON schema definition in JSON Schema format';
COMMENT ON COLUMN processed_data.confidence_score IS 'AI confidence in extraction accuracy (0.0-1.0)';
COMMENT ON COLUMN processed_data.processing_status IS 'Current status: pending, processing, completed, failed, retrying';
COMMENT ON COLUMN processing_jobs.priority IS 'Job priority: 1 (highest) to 10 (lowest)';