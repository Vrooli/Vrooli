-- Personal Digital Twin Database Schema

CREATE DATABASE IF NOT EXISTS digital_twin;
\c digital_twin;

-- Personas table
CREATE TABLE personas (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    base_model VARCHAR(100) DEFAULT 'llama3.2',
    fine_tuned_model_path TEXT,
    training_status VARCHAR(50) DEFAULT 'not_started',
    personality_traits JSONB DEFAULT '{}',
    knowledge_domains JSONB DEFAULT '[]',
    conversation_style JSONB DEFAULT '{}',
    document_count INTEGER DEFAULT 0,
    total_tokens INTEGER DEFAULT 0,
    last_trained TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Data sources configuration
CREATE TABLE data_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    persona_id UUID REFERENCES personas(id) ON DELETE CASCADE,
    source_type VARCHAR(50) NOT NULL, -- 'google_drive', 'github', 'email', 'local_files'
    source_config JSONB NOT NULL,
    auth_credentials JSONB, -- encrypted
    sync_frequency VARCHAR(50) DEFAULT 'manual', -- 'manual', 'hourly', 'daily', 'weekly'
    last_sync TIMESTAMP,
    total_documents INTEGER DEFAULT 0,
    total_size_bytes BIGINT DEFAULT 0,
    status VARCHAR(50) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Processed documents tracking
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    persona_id UUID REFERENCES personas(id) ON DELETE CASCADE,
    source_id UUID REFERENCES data_sources(id) ON DELETE CASCADE,
    original_path TEXT NOT NULL,
    processed_path TEXT,
    document_type VARCHAR(50),
    content_hash VARCHAR(64),
    metadata JSONB DEFAULT '{}',
    chunk_count INTEGER DEFAULT 0,
    token_count INTEGER DEFAULT 0,
    vector_ids UUID[] DEFAULT '{}',
    processing_status VARCHAR(50) DEFAULT 'pending',
    processed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Conversation history
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    persona_id UUID REFERENCES personas(id) ON DELETE CASCADE,
    session_id VARCHAR(255),
    user_id VARCHAR(255),
    messages JSONB DEFAULT '[]',
    context_window JSONB DEFAULT '[]',
    total_tokens_used INTEGER DEFAULT 0,
    feedback_score FLOAT,
    feedback_notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP
);

-- Training jobs
CREATE TABLE training_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    persona_id UUID REFERENCES personas(id) ON DELETE CASCADE,
    job_type VARCHAR(50) NOT NULL, -- 'fine_tune', 'embedding_update', 'reindex'
    model_name VARCHAR(100),
    technique VARCHAR(50), -- 'lora', 'qlora', 'full'
    training_config JSONB,
    dataset_path TEXT,
    checkpoint_path TEXT,
    metrics JSONB DEFAULT '{}',
    status VARCHAR(50) DEFAULT 'queued',
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Memory snapshots for rollback
CREATE TABLE memory_snapshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    persona_id UUID REFERENCES personas(id) ON DELETE CASCADE,
    snapshot_name VARCHAR(255),
    description TEXT,
    vector_collection_snapshot VARCHAR(255),
    model_checkpoint_path TEXT,
    document_count INTEGER,
    total_tokens INTEGER,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- API access tokens for external integrations
CREATE TABLE api_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    persona_id UUID REFERENCES personas(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(255),
    permissions JSONB DEFAULT '["read"]',
    last_used TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_personas_name ON personas(name);
CREATE INDEX idx_data_sources_persona ON data_sources(persona_id);
CREATE INDEX idx_documents_persona ON documents(persona_id);
CREATE INDEX idx_documents_source ON documents(source_id);
CREATE INDEX idx_documents_status ON documents(processing_status);
CREATE INDEX idx_conversations_persona ON conversations(persona_id);
CREATE INDEX idx_conversations_session ON conversations(session_id);
CREATE INDEX idx_training_jobs_persona ON training_jobs(persona_id);
CREATE INDEX idx_training_jobs_status ON training_jobs(status);

-- Update trigger
CREATE OR REPLACE FUNCTION update_last_updated()
RETURNS TRIGGER AS $$
BEGIN
    NEW.last_updated = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER personas_last_updated BEFORE UPDATE ON personas
    FOR EACH ROW EXECUTE FUNCTION update_last_updated();