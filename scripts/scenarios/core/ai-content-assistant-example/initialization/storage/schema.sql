-- AI Content Assistant Database Schema
-- Campaign-based content management with document context and AI generation

-- Create application schema
CREATE SCHEMA IF NOT EXISTS ai_content_assistant;

-- Core application metadata
CREATE TABLE IF NOT EXISTS ai_content_assistant.app_metadata (
    id SERIAL PRIMARY KEY,
    app_name VARCHAR(255) NOT NULL DEFAULT 'AI Content Assistant',
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    deployed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    configuration JSONB,
    
    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Campaign management
CREATE TABLE IF NOT EXISTS ai_content_assistant.campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Campaign settings
    brand_guidelines JSONB, -- tone, style, keywords
    content_preferences JSONB, -- preferred formats, lengths
    target_audience TEXT,
    
    -- Status and metadata
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Indexes
    CONSTRAINT campaigns_name_unique UNIQUE(name)
);

-- Document storage and metadata
CREATE TABLE IF NOT EXISTS ai_content_assistant.campaign_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID REFERENCES ai_content_assistant.campaigns(id) ON DELETE CASCADE,
    
    -- File information
    filename VARCHAR(255) NOT NULL,
    original_filename VARCHAR(255) NOT NULL,
    file_path TEXT NOT NULL, -- MinIO path
    file_size_bytes INTEGER,
    content_type VARCHAR(100),
    
    -- Processed content
    extracted_text TEXT,
    text_chunks JSONB, -- Array of text chunks for embedding
    embedding_id VARCHAR(255), -- Qdrant point ID
    
    -- Status and metadata
    processing_status VARCHAR(50) DEFAULT 'uploaded', -- uploaded, processing, completed, failed
    processing_error TEXT,
    upload_date TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    processed_date TIMESTAMP WITH TIME ZONE
);

-- Enhanced user sessions with campaign context
CREATE TABLE IF NOT EXISTS ai_content_assistant.user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id VARCHAR(255) UNIQUE NOT NULL,
    user_identifier VARCHAR(255),
    
    -- Campaign context
    current_campaign_id UUID REFERENCES ai_content_assistant.campaigns(id),
    
    -- Session metadata
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_activity TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true,
    
    -- Context and preferences
    context_data JSONB,
    preferences JSONB,
    
    -- Performance tracking
    request_count INTEGER DEFAULT 0,
    total_processing_time_ms INTEGER DEFAULT 0
);

-- Content generation history
CREATE TABLE IF NOT EXISTS ai_content_assistant.generated_content (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID REFERENCES ai_content_assistant.campaigns(id) ON DELETE CASCADE,
    
    -- Content details
    content_type VARCHAR(50) NOT NULL, -- 'blog_post', 'social_media', 'email', 'image_prompt'
    title VARCHAR(500),
    prompt TEXT NOT NULL,
    generated_content TEXT,
    
    -- Context used
    used_documents UUID[], -- Array of document IDs used for context
    context_summary TEXT,
    
    -- Generation metadata
    model_used VARCHAR(100),
    generation_parameters JSONB,
    
    -- Status and timing
    status VARCHAR(50) DEFAULT 'completed', -- generating, completed, failed
    generation_time_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- User context
    session_id UUID REFERENCES ai_content_assistant.user_sessions(id),
    user_identifier VARCHAR(255)
);

-- Application events and audit log
CREATE TABLE IF NOT EXISTS ai_content_assistant.activity_log (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Event details
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB,
    
    -- Campaign and user context
    campaign_id UUID REFERENCES ai_content_assistant.campaigns(id),
    session_id UUID REFERENCES ai_content_assistant.user_sessions(id),
    user_identifier VARCHAR(255),
    
    -- Performance metrics
    processing_time_ms INTEGER,
    resource_usage JSONB,
    
    -- Status tracking
    status VARCHAR(50) DEFAULT 'completed', -- pending, completed, failed
    error_message TEXT
);

-- Resource usage tracking
CREATE TABLE IF NOT EXISTS ai_content_assistant.resource_metrics (
    id SERIAL PRIMARY KEY,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Resource identification
    resource_type VARCHAR(50) NOT NULL, -- ollama, n8n, whisper, etc.
    resource_name VARCHAR(100) NOT NULL,
    operation VARCHAR(100) NOT NULL,
    
    -- Performance metrics
    duration_ms INTEGER NOT NULL,
    success BOOLEAN NOT NULL DEFAULT true,
    
    -- Usage details
    input_size_bytes INTEGER,
    output_size_bytes INTEGER,
    tokens_used INTEGER, -- For AI resources
    
    -- Cost tracking
    estimated_cost_usd DECIMAL(10,6),
    
    -- Context
    session_id UUID REFERENCES ai_content_assistant.user_sessions(id),
    request_metadata JSONB
);

-- Configuration and feature flags
CREATE TABLE IF NOT EXISTS ai_content_assistant.app_config (
    id SERIAL PRIMARY KEY,
    config_key VARCHAR(255) UNIQUE NOT NULL,
    config_value JSONB NOT NULL,
    config_type VARCHAR(50) NOT NULL DEFAULT 'setting', -- setting, feature_flag, resource_config
    
    -- Metadata
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
-- Campaign indexes
CREATE INDEX idx_campaigns_active ON ai_content_assistant.campaigns(is_active, created_at DESC);
CREATE INDEX idx_campaigns_name ON ai_content_assistant.campaigns(name);

-- Document indexes
CREATE INDEX idx_campaign_documents_campaign ON ai_content_assistant.campaign_documents(campaign_id, upload_date DESC);
CREATE INDEX idx_campaign_documents_status ON ai_content_assistant.campaign_documents(processing_status);
CREATE INDEX idx_campaign_documents_embedding ON ai_content_assistant.campaign_documents(embedding_id) WHERE embedding_id IS NOT NULL;

-- User session indexes
CREATE INDEX idx_user_sessions_session_id ON ai_content_assistant.user_sessions(session_id);
CREATE INDEX idx_user_sessions_campaign ON ai_content_assistant.user_sessions(current_campaign_id);
CREATE INDEX idx_user_sessions_active ON ai_content_assistant.user_sessions(is_active, last_activity);

-- Generated content indexes
CREATE INDEX idx_generated_content_campaign ON ai_content_assistant.generated_content(campaign_id, created_at DESC);
CREATE INDEX idx_generated_content_type ON ai_content_assistant.generated_content(content_type);
CREATE INDEX idx_generated_content_session ON ai_content_assistant.generated_content(session_id);

-- Activity log indexes
CREATE INDEX idx_activity_log_timestamp ON ai_content_assistant.activity_log(timestamp DESC);
CREATE INDEX idx_activity_log_campaign ON ai_content_assistant.activity_log(campaign_id);
CREATE INDEX idx_activity_log_session ON ai_content_assistant.activity_log(session_id);
CREATE INDEX idx_activity_log_event_type ON ai_content_assistant.activity_log(event_type);

-- Resource metrics indexes
CREATE INDEX idx_resource_metrics_timestamp ON ai_content_assistant.resource_metrics(timestamp DESC);
CREATE INDEX idx_resource_metrics_resource ON ai_content_assistant.resource_metrics(resource_type, resource_name);

-- App config index
CREATE INDEX idx_app_config_key ON ai_content_assistant.app_config(config_key);

-- Useful views for monitoring and analytics
CREATE OR REPLACE VIEW ai_content_assistant.campaign_summary AS
SELECT 
    c.id,
    c.name,
    c.description,
    COUNT(DISTINCT cd.id) as document_count,
    COUNT(DISTINCT gc.id) as generated_content_count,
    MAX(gc.created_at) as last_generation_date,
    c.created_at,
    c.is_active
FROM ai_content_assistant.campaigns c
LEFT JOIN ai_content_assistant.campaign_documents cd ON c.id = cd.campaign_id
LEFT JOIN ai_content_assistant.generated_content gc ON c.id = gc.campaign_id
GROUP BY c.id, c.name, c.description, c.created_at, c.is_active
ORDER BY c.created_at DESC;

CREATE OR REPLACE VIEW ai_content_assistant.content_generation_stats AS
SELECT 
    content_type,
    COUNT(*) as total_generated,
    AVG(generation_time_ms) as avg_generation_time_ms,
    COUNT(DISTINCT campaign_id) as campaigns_using,
    DATE_TRUNC('day', created_at) as generation_date
FROM ai_content_assistant.generated_content
WHERE created_at > NOW() - INTERVAL '30 days'
GROUP BY content_type, DATE_TRUNC('day', created_at)
ORDER BY generation_date DESC, total_generated DESC;

CREATE OR REPLACE VIEW ai_content_assistant.document_processing_status AS
SELECT 
    processing_status,
    COUNT(*) as document_count,
    COUNT(DISTINCT campaign_id) as campaigns_affected,
    AVG(CASE WHEN processed_date IS NOT NULL AND upload_date IS NOT NULL 
        THEN EXTRACT(EPOCH FROM (processed_date - upload_date)) 
        ELSE NULL END) as avg_processing_time_seconds
FROM ai_content_assistant.campaign_documents
GROUP BY processing_status
ORDER BY document_count DESC;

CREATE OR REPLACE VIEW ai_content_assistant.session_summary AS
SELECT 
    DATE_TRUNC('hour', started_at) as hour,
    COUNT(*) as total_sessions,
    COUNT(DISTINCT user_identifier) as unique_users,
    COUNT(DISTINCT current_campaign_id) as active_campaigns,
    AVG(EXTRACT(EPOCH FROM (last_activity - started_at))) as avg_session_duration_seconds,
    SUM(request_count) as total_requests
FROM ai_content_assistant.user_sessions
WHERE started_at > NOW() - INTERVAL '24 hours'
GROUP BY DATE_TRUNC('hour', started_at)
ORDER BY hour DESC;

CREATE OR REPLACE VIEW ai_content_assistant.resource_performance AS
SELECT 
    resource_type,
    resource_name,
    COUNT(*) as total_calls,
    AVG(duration_ms) as avg_duration_ms,
    PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration_ms) as p95_duration_ms,
    SUM(CASE WHEN success THEN 1 ELSE 0 END)::FLOAT / COUNT(*) * 100 as success_rate_percent,
    SUM(COALESCE(estimated_cost_usd, 0)) as total_estimated_cost_usd
FROM ai_content_assistant.resource_metrics
WHERE timestamp > NOW() - INTERVAL '24 hours'
GROUP BY resource_type, resource_name
ORDER BY total_calls DESC;

-- Grant permissions (adjust based on your user setup)
GRANT USAGE ON SCHEMA ai_content_assistant TO PUBLIC;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA ai_content_assistant TO PUBLIC;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA ai_content_assistant TO PUBLIC;
GRANT SELECT ON ALL TABLES IN SCHEMA ai_content_assistant TO PUBLIC;

-- Enable row level security for multi-tenant scenarios (optional)
-- ALTER TABLE ai_content_assistant.campaigns ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE ai_content_assistant.campaign_documents ENABLE ROW LEVEL SECURITY;
-- ALTER TABLE ai_content_assistant.generated_content ENABLE ROW LEVEL SECURITY;