-- Text Tools Database Schema
-- Version: 1.0.0
-- Description: Core schema for text processing and analysis platform

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm"; -- For fuzzy text search
CREATE EXTENSION IF NOT EXISTS "pgvector"; -- For semantic search vectors

-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS text_tools;
SET search_path TO text_tools, public;

-- Text Documents table
CREATE TABLE IF NOT EXISTS text_documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    content_hash VARCHAR(64) NOT NULL, -- SHA256 hash
    format VARCHAR(50) NOT NULL CHECK (format IN ('plain', 'markdown', 'html', 'pdf', 'docx', 'rtf', 'latex', 'xml', 'json')),
    size_bytes BIGINT NOT NULL,
    language VARCHAR(10),
    encoding VARCHAR(50) DEFAULT 'UTF-8',
    minio_path VARCHAR(500), -- Path in MinIO for large files
    content TEXT, -- For small files (<1MB), store directly
    metadata JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    modified_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    accessed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE, -- Soft delete
    CONSTRAINT unique_content_hash UNIQUE(content_hash)
);

-- Create indexes for text_documents
CREATE INDEX idx_text_documents_name ON text_documents USING gin(to_tsvector('english', name));
CREATE INDEX idx_text_documents_format ON text_documents(format);
CREATE INDEX idx_text_documents_language ON text_documents(language);
CREATE INDEX idx_text_documents_tags ON text_documents USING gin(tags);
CREATE INDEX idx_text_documents_metadata ON text_documents USING gin(metadata);
CREATE INDEX idx_text_documents_created_at ON text_documents(created_at DESC);
CREATE INDEX idx_text_documents_content_hash ON text_documents(content_hash);

-- Text Operations table
CREATE TABLE IF NOT EXISTS text_operations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID REFERENCES text_documents(id) ON DELETE CASCADE,
    operation_type VARCHAR(50) NOT NULL CHECK (operation_type IN (
        'diff', 'search', 'transform', 'extract', 'analyze',
        'summarize', 'translate', 'sentiment', 'entities', 'keywords'
    )),
    parameters JSONB NOT NULL DEFAULT '{}',
    input_data JSONB, -- For operations without document_id
    result_summary JSONB NOT NULL DEFAULT '{}',
    result_path VARCHAR(500), -- MinIO path for large results
    result_cache_key VARCHAR(255), -- Redis cache key
    duration_ms INTEGER,
    error_message TEXT,
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'processing', 'completed', 'failed', 'cancelled')),
    created_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Create indexes for text_operations
CREATE INDEX idx_text_operations_document_id ON text_operations(document_id);
CREATE INDEX idx_text_operations_type ON text_operations(operation_type);
CREATE INDEX idx_text_operations_status ON text_operations(status);
CREATE INDEX idx_text_operations_created_at ON text_operations(created_at DESC);
CREATE INDEX idx_text_operations_cache_key ON text_operations(result_cache_key);

-- Text Templates table
CREATE TABLE IF NOT EXISTS text_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    category VARCHAR(100),
    content TEXT NOT NULL,
    variables JSONB DEFAULT '[]', -- Array of variable definitions
    description TEXT,
    tags TEXT[] DEFAULT '{}',
    usage_count INTEGER DEFAULT 0,
    is_public BOOLEAN DEFAULT true,
    created_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    modified_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for text_templates
CREATE INDEX idx_text_templates_name ON text_templates(name);
CREATE INDEX idx_text_templates_category ON text_templates(category);
CREATE INDEX idx_text_templates_tags ON text_templates USING gin(tags);
CREATE INDEX idx_text_templates_usage_count ON text_templates(usage_count DESC);

-- Text Comparisons table (for diff operations)
CREATE TABLE IF NOT EXISTS text_comparisons (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document1_id UUID REFERENCES text_documents(id) ON DELETE CASCADE,
    document2_id UUID REFERENCES text_documents(id) ON DELETE CASCADE,
    operation_id UUID REFERENCES text_operations(id) ON DELETE CASCADE,
    similarity_score DECIMAL(5,4), -- 0.0000 to 1.0000
    diff_type VARCHAR(50) CHECK (diff_type IN ('line', 'word', 'character', 'semantic', 'structural')),
    changes_count INTEGER,
    additions INTEGER,
    deletions INTEGER,
    modifications INTEGER,
    diff_result JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for text_comparisons
CREATE INDEX idx_text_comparisons_doc1 ON text_comparisons(document1_id);
CREATE INDEX idx_text_comparisons_doc2 ON text_comparisons(document2_id);
CREATE INDEX idx_text_comparisons_operation ON text_comparisons(operation_id);
CREATE INDEX idx_text_comparisons_similarity ON text_comparisons(similarity_score DESC);

-- Text Embeddings table (for semantic search)
CREATE TABLE IF NOT EXISTS text_embeddings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    document_id UUID REFERENCES text_documents(id) ON DELETE CASCADE,
    chunk_index INTEGER NOT NULL, -- For documents split into chunks
    chunk_text TEXT,
    embedding vector(1536), -- OpenAI/Ollama embedding dimension
    model_name VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_document_chunk UNIQUE(document_id, chunk_index)
);

-- Create indexes for text_embeddings
CREATE INDEX idx_text_embeddings_document ON text_embeddings(document_id);
CREATE INDEX idx_text_embeddings_vector ON text_embeddings USING ivfflat (embedding vector_cosine_ops);

-- Text Processing Pipelines table
CREATE TABLE IF NOT EXISTS text_pipelines (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    steps JSONB NOT NULL, -- Array of processing steps
    input_format VARCHAR(50),
    output_format VARCHAR(50),
    is_public BOOLEAN DEFAULT false,
    usage_count INTEGER DEFAULT 0,
    average_duration_ms INTEGER,
    created_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    modified_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for text_pipelines
CREATE INDEX idx_text_pipelines_name ON text_pipelines(name);
CREATE INDEX idx_text_pipelines_usage ON text_pipelines(usage_count DESC);

-- Text Extraction Rules table
CREATE TABLE IF NOT EXISTS text_extraction_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    source_format VARCHAR(50) NOT NULL,
    rule_type VARCHAR(50) CHECK (rule_type IN ('regex', 'xpath', 'css_selector', 'json_path', 'custom')),
    pattern TEXT NOT NULL,
    description TEXT,
    test_input TEXT,
    test_output TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    modified_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for text_extraction_rules
CREATE INDEX idx_text_extraction_rules_format ON text_extraction_rules(source_format);
CREATE INDEX idx_text_extraction_rules_type ON text_extraction_rules(rule_type);
CREATE INDEX idx_text_extraction_rules_active ON text_extraction_rules(is_active);

-- API Usage Statistics table
CREATE TABLE IF NOT EXISTS api_usage_stats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    scenario_name VARCHAR(255),
    user_id VARCHAR(255),
    request_size_bytes INTEGER,
    response_size_bytes INTEGER,
    duration_ms INTEGER,
    status_code INTEGER,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for api_usage_stats
CREATE INDEX idx_api_usage_stats_endpoint ON api_usage_stats(endpoint);
CREATE INDEX idx_api_usage_stats_scenario ON api_usage_stats(scenario_name);
CREATE INDEX idx_api_usage_stats_created_at ON api_usage_stats(created_at DESC);
CREATE INDEX idx_api_usage_stats_status ON api_usage_stats(status_code);

-- Function to update modified_at timestamp
CREATE OR REPLACE FUNCTION update_modified_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.modified_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for modified_at
CREATE TRIGGER update_text_documents_modified
    BEFORE UPDATE ON text_documents
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_at();

CREATE TRIGGER update_text_templates_modified
    BEFORE UPDATE ON text_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_at();

CREATE TRIGGER update_text_pipelines_modified
    BEFORE UPDATE ON text_pipelines
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_at();

CREATE TRIGGER update_text_extraction_rules_modified
    BEFORE UPDATE ON text_extraction_rules
    FOR EACH ROW
    EXECUTE FUNCTION update_modified_at();

-- Function to calculate text similarity
CREATE OR REPLACE FUNCTION calculate_text_similarity(text1 TEXT, text2 TEXT)
RETURNS DECIMAL(5,4) AS $$
BEGIN
    RETURN similarity(text1, text2);
END;
$$ LANGUAGE plpgsql;

-- Function to extract text statistics
CREATE OR REPLACE FUNCTION extract_text_stats(input_text TEXT)
RETURNS JSONB AS $$
DECLARE
    result JSONB;
BEGIN
    result = jsonb_build_object(
        'character_count', length(input_text),
        'word_count', array_length(string_to_array(input_text, ' '), 1),
        'line_count', array_length(string_to_array(input_text, E'\n'), 1),
        'paragraph_count', array_length(string_to_array(input_text, E'\n\n'), 1)
    );
    RETURN result;
END;
$$ LANGUAGE plpgsql;

-- Create materialized view for operation statistics
CREATE MATERIALIZED VIEW IF NOT EXISTS operation_statistics AS
SELECT 
    operation_type,
    COUNT(*) as total_operations,
    AVG(duration_ms) as avg_duration_ms,
    MIN(duration_ms) as min_duration_ms,
    MAX(duration_ms) as max_duration_ms,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) as successful_ops,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_ops,
    DATE_TRUNC('day', created_at) as day
FROM text_operations
WHERE created_at > CURRENT_DATE - INTERVAL '30 days'
GROUP BY operation_type, DATE_TRUNC('day', created_at);

-- Create index on materialized view
CREATE INDEX idx_operation_statistics_day ON operation_statistics(day DESC);

-- Grant permissions (adjust as needed for your setup)
GRANT ALL ON SCHEMA text_tools TO postgres;
GRANT ALL ON ALL TABLES IN SCHEMA text_tools TO postgres;
GRANT ALL ON ALL SEQUENCES IN SCHEMA text_tools TO postgres;
GRANT ALL ON ALL FUNCTIONS IN SCHEMA text_tools TO postgres;