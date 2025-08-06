-- Podcast Transcription Assistant Database Schema
-- Specialized schema for audio transcription, AI analysis, and semantic search

-- Create application schema
CREATE SCHEMA IF NOT EXISTS podcast_transcription_assistant;

-- Core transcriptions table
CREATE TABLE IF NOT EXISTS podcast_transcription_assistant.transcriptions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- File information
    filename VARCHAR(500) NOT NULL,
    original_file_path TEXT NOT NULL,
    file_size_bytes BIGINT NOT NULL,
    content_type VARCHAR(100) NOT NULL,
    duration_seconds INTEGER,
    
    -- Transcription content
    transcription_text TEXT NOT NULL,
    word_timestamps JSONB, -- Array of {word, start, end} objects
    confidence_score DECIMAL(5,4), -- Whisper confidence
    language_detected VARCHAR(10),
    
    -- Processing metadata
    whisper_model_used VARCHAR(50) NOT NULL DEFAULT 'base',
    processing_time_ms INTEGER NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'completed', -- pending, completed, failed, processing
    error_message TEXT,
    
    -- Embedding status for search
    embedding_status VARCHAR(20) NOT NULL DEFAULT 'pending', -- pending, completed, failed
    embedding_model VARCHAR(50) DEFAULT 'nomic-embed-text',
    
    -- Storage references
    minio_audio_path TEXT NOT NULL, -- Path in MinIO audio-files bucket
    minio_transcript_path TEXT, -- Path in MinIO transcriptions bucket
    
    -- User context
    session_id VARCHAR(255),
    user_identifier VARCHAR(255),
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- AI analysis results table
CREATE TABLE IF NOT EXISTS podcast_transcription_assistant.ai_analyses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transcription_id UUID NOT NULL REFERENCES podcast_transcription_assistant.transcriptions(id) ON DELETE CASCADE,
    
    -- Analysis details
    analysis_type VARCHAR(50) NOT NULL, -- 'summary', 'key_insights', 'custom'
    prompt_used TEXT NOT NULL,
    result_text TEXT NOT NULL,
    
    -- AI processing metadata
    ai_model_used VARCHAR(50) NOT NULL DEFAULT 'llama3.1:8b',
    processing_time_ms INTEGER NOT NULL,
    tokens_used INTEGER,
    confidence_score DECIMAL(5,4),
    
    -- User context
    session_id VARCHAR(255),
    user_identifier VARCHAR(255),
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User sessions for tracking preferences and activity
CREATE TABLE IF NOT EXISTS podcast_transcription_assistant.user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id VARCHAR(255) UNIQUE NOT NULL,
    user_identifier VARCHAR(255),
    
    -- User preferences
    preferences JSONB DEFAULT '{}',
    search_history JSONB DEFAULT '[]', -- Array of recent searches
    favorite_transcriptions UUID[], -- Array of transcription IDs
    
    -- Session metadata
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_activity TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true,
    
    -- Usage statistics
    transcriptions_count INTEGER DEFAULT 0,
    analyses_count INTEGER DEFAULT 0,
    searches_count INTEGER DEFAULT 0,
    total_audio_duration_seconds INTEGER DEFAULT 0
);

-- Search queries and results for analytics
CREATE TABLE IF NOT EXISTS podcast_transcription_assistant.search_queries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Query details
    query_text TEXT NOT NULL,
    query_embedding_model VARCHAR(50) NOT NULL DEFAULT 'nomic-embed-text',
    
    -- Results metadata
    results_count INTEGER NOT NULL DEFAULT 0,
    max_similarity_score DECIMAL(5,4),
    processing_time_ms INTEGER NOT NULL,
    
    -- User context
    session_id VARCHAR(255),
    user_identifier VARCHAR(255),
    
    -- Timestamp
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Resource usage tracking for monitoring and optimization
CREATE TABLE IF NOT EXISTS podcast_transcription_assistant.resource_metrics (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Resource identification
    resource_type VARCHAR(50) NOT NULL, -- whisper, ollama, qdrant, minio, etc.
    operation VARCHAR(100) NOT NULL, -- transcribe, analyze, search, upload, etc.
    
    -- Performance metrics
    duration_ms INTEGER NOT NULL,
    success BOOLEAN NOT NULL DEFAULT true,
    
    -- Usage details
    input_size_bytes INTEGER,
    output_size_bytes INTEGER,
    tokens_used INTEGER,
    
    -- Context
    transcription_id UUID REFERENCES podcast_transcription_assistant.transcriptions(id),
    session_id VARCHAR(255),
    request_metadata JSONB
);

-- Application configuration
CREATE TABLE IF NOT EXISTS podcast_transcription_assistant.app_config (
    id SERIAL PRIMARY KEY,
    config_key VARCHAR(255) UNIQUE NOT NULL,
    config_value JSONB NOT NULL,
    config_type VARCHAR(50) NOT NULL DEFAULT 'setting',
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Performance indexes
CREATE INDEX idx_transcriptions_created_at ON podcast_transcription_assistant.transcriptions(created_at DESC);
CREATE INDEX idx_transcriptions_session ON podcast_transcription_assistant.transcriptions(session_id);
CREATE INDEX idx_transcriptions_user ON podcast_transcription_assistant.transcriptions(user_identifier);
CREATE INDEX idx_transcriptions_status ON podcast_transcription_assistant.transcriptions(status);
CREATE INDEX idx_transcriptions_embedding_status ON podcast_transcription_assistant.transcriptions(embedding_status);
CREATE INDEX idx_transcriptions_filename ON podcast_transcription_assistant.transcriptions(filename);
CREATE INDEX idx_transcriptions_duration ON podcast_transcription_assistant.transcriptions(duration_seconds);

CREATE INDEX idx_ai_analyses_transcription_id ON podcast_transcription_assistant.ai_analyses(transcription_id);
CREATE INDEX idx_ai_analyses_type ON podcast_transcription_assistant.ai_analyses(analysis_type);
CREATE INDEX idx_ai_analyses_created_at ON podcast_transcription_assistant.ai_analyses(created_at DESC);
CREATE INDEX idx_ai_analyses_session ON podcast_transcription_assistant.ai_analyses(session_id);

CREATE INDEX idx_user_sessions_session_id ON podcast_transcription_assistant.user_sessions(session_id);
CREATE INDEX idx_user_sessions_active ON podcast_transcription_assistant.user_sessions(is_active, last_activity);
CREATE INDEX idx_user_sessions_user ON podcast_transcription_assistant.user_sessions(user_identifier);

CREATE INDEX idx_search_queries_created_at ON podcast_transcription_assistant.search_queries(created_at DESC);
CREATE INDEX idx_search_queries_session ON podcast_transcription_assistant.search_queries(session_id);
CREATE INDEX idx_search_queries_text ON podcast_transcription_assistant.search_queries USING gin(to_tsvector('english', query_text));

CREATE INDEX idx_resource_metrics_timestamp ON podcast_transcription_assistant.resource_metrics(timestamp DESC);
CREATE INDEX idx_resource_metrics_resource ON podcast_transcription_assistant.resource_metrics(resource_type, operation);
CREATE INDEX idx_resource_metrics_transcription ON podcast_transcription_assistant.resource_metrics(transcription_id);

CREATE INDEX idx_app_config_key ON podcast_transcription_assistant.app_config(config_key);

-- Analytics views
CREATE OR REPLACE VIEW podcast_transcription_assistant.transcription_stats AS
SELECT 
    DATE_TRUNC('day', created_at) as date,
    COUNT(*) as transcriptions_count,
    COUNT(DISTINCT session_id) as unique_sessions,
    SUM(duration_seconds) as total_audio_seconds,
    AVG(duration_seconds) as avg_audio_seconds,
    AVG(processing_time_ms) as avg_processing_ms,
    SUM(file_size_bytes) as total_file_size_bytes,
    COUNT(CASE WHEN status = 'completed' THEN 1 END)::FLOAT / COUNT(*) * 100 as success_rate_percent
FROM podcast_transcription_assistant.transcriptions
WHERE created_at > NOW() - INTERVAL '30 days'
GROUP BY DATE_TRUNC('day', created_at)
ORDER BY date DESC;

CREATE OR REPLACE VIEW podcast_transcription_assistant.user_activity_summary AS
SELECT 
    user_identifier,
    COUNT(DISTINCT t.id) as transcriptions_count,
    COUNT(DISTINCT a.id) as analyses_count,
    SUM(t.duration_seconds) as total_audio_seconds,
    AVG(t.processing_time_ms) as avg_transcription_time_ms,
    MAX(t.created_at) as last_transcription_at,
    COUNT(DISTINCT DATE_TRUNC('day', t.created_at)) as active_days
FROM podcast_transcription_assistant.user_sessions us
LEFT JOIN podcast_transcription_assistant.transcriptions t ON us.session_id = t.session_id
LEFT JOIN podcast_transcription_assistant.ai_analyses a ON us.session_id = a.session_id
WHERE us.user_identifier IS NOT NULL
GROUP BY user_identifier
HAVING COUNT(DISTINCT t.id) > 0
ORDER BY transcriptions_count DESC;

CREATE OR REPLACE VIEW podcast_transcription_assistant.resource_performance AS
SELECT 
    resource_type,
    operation,
    COUNT(*) as total_operations,
    AVG(duration_ms) as avg_duration_ms,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95_duration_ms,
    SUM(CASE WHEN success THEN 1 ELSE 0 END)::FLOAT / COUNT(*) * 100 as success_rate_percent,
    SUM(COALESCE(tokens_used, 0)) as total_tokens_used
FROM podcast_transcription_assistant.resource_metrics
WHERE timestamp > NOW() - INTERVAL '24 hours'
GROUP BY resource_type, operation
ORDER BY total_operations DESC;

CREATE OR REPLACE VIEW podcast_transcription_assistant.search_analytics AS
SELECT 
    DATE_TRUNC('hour', created_at) as hour,
    COUNT(*) as search_count,
    AVG(results_count) as avg_results_count,
    AVG(max_similarity_score) as avg_max_similarity,
    AVG(processing_time_ms) as avg_processing_time_ms,
    COUNT(DISTINCT session_id) as unique_sessions
FROM podcast_transcription_assistant.search_queries
WHERE created_at > NOW() - INTERVAL '7 days'
GROUP BY DATE_TRUNC('hour', created_at)
ORDER BY hour DESC;

-- Insert default configuration
INSERT INTO podcast_transcription_assistant.app_config (config_key, config_value, config_type, description) VALUES
('app.name', '"Podcast Transcription Assistant"', 'setting', 'Application display name'),
('app.version', '"1.0.0"', 'setting', 'Application version'),
('whisper.default_model', '"base"', 'setting', 'Default Whisper model for transcription'),
('whisper.max_file_size_mb', '500', 'setting', 'Maximum audio file size in MB'),
('ollama.default_model', '"llama3.1:8b"', 'setting', 'Default Ollama model for AI analysis'),
('ollama.temperature', '0.7', 'setting', 'Default temperature for AI analysis'),
('search.max_results', '50', 'setting', 'Maximum search results to return'),
('search.similarity_threshold', '0.7', 'setting', 'Minimum similarity score for search results'),
('ui.items_per_page', '20', 'setting', 'Number of transcriptions per page'),
('storage.retention_days', '365', 'setting', 'Days to retain transcriptions and files'),
('features.semantic_search', 'true', 'feature_flag', 'Enable semantic search functionality'),
('features.ai_analysis', 'true', 'feature_flag', 'Enable AI analysis tools'),
('features.export', 'true', 'feature_flag', 'Enable export functionality')
ON CONFLICT (config_key) DO NOTHING;

-- Grant permissions
GRANT USAGE ON SCHEMA podcast_transcription_assistant TO PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA podcast_transcription_assistant TO PUBLIC;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA podcast_transcription_assistant TO PUBLIC;
GRANT SELECT ON ALL TABLES IN SCHEMA podcast_transcription_assistant TO PUBLIC;

-- Update schema name in any foreign key references
ALTER TABLE IF EXISTS podcast_transcription_assistant.ai_analyses 
    DROP CONSTRAINT IF EXISTS ai_analyses_transcription_id_fkey,
    ADD CONSTRAINT ai_analyses_transcription_id_fkey 
    FOREIGN KEY (transcription_id) REFERENCES podcast_transcription_assistant.transcriptions(id) ON DELETE CASCADE;

ALTER TABLE IF EXISTS podcast_transcription_assistant.resource_metrics 
    DROP CONSTRAINT IF EXISTS resource_metrics_transcription_id_fkey,
    ADD CONSTRAINT resource_metrics_transcription_id_fkey 
    FOREIGN KEY (transcription_id) REFERENCES podcast_transcription_assistant.transcriptions(id) ON DELETE SET NULL;