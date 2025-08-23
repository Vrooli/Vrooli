-- Idea Generator Database Schema
-- Purpose: Store campaigns, ideas, chat history, documents, and agent interactions

-- Enable necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "vector";

-- Users table (for session and ownership management)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username VARCHAR(100) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE,
    preferences JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_active TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Campaigns table
CREATE TABLE campaigns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(7) DEFAULT '#3B82F6', -- Hex color for UI theming
    context JSONB DEFAULT '{}',
    settings JSONB DEFAULT '{}',
    embedding vector(1536), -- Campaign context embeddings
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    owner_id UUID REFERENCES users(id),
    is_active BOOLEAN DEFAULT true
);

-- Documents table (uploaded files and their metadata)
CREATE TABLE documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    filename VARCHAR(255) NOT NULL,
    original_name VARCHAR(255) NOT NULL,
    file_type VARCHAR(50) NOT NULL,
    file_size BIGINT NOT NULL,
    storage_path TEXT NOT NULL, -- MinIO path
    extracted_text TEXT, -- Text extracted by Unstructured-IO
    embedding vector(1536), -- Document embeddings
    processing_status VARCHAR(50) DEFAULT 'pending', -- pending, processing, completed, failed
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP WITH TIME ZONE
);

-- Ideas table
CREATE TABLE ideas (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    category VARCHAR(100),
    tags TEXT[],
    metadata JSONB DEFAULT '{}',
    generation_prompt TEXT,
    generation_model VARCHAR(100),
    embedding vector(1536),
    score DECIMAL(5,2),
    status VARCHAR(50) DEFAULT 'draft', -- draft, refined, archived, finalized
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID REFERENCES users(id)
);

-- Chat sessions table (for real-time idea refinement)
CREATE TABLE chat_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    idea_id UUID REFERENCES ideas(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id),
    session_type VARCHAR(50) DEFAULT 'refinement', -- refinement, brainstorm, critique
    status VARCHAR(50) DEFAULT 'active', -- active, archived, completed
    context JSONB DEFAULT '{}', -- Session context and state
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP WITH TIME ZONE
);

-- Chat messages table
CREATE TABLE chat_messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_id UUID REFERENCES chat_sessions(id) ON DELETE CASCADE,
    sender_type VARCHAR(20) NOT NULL, -- 'user', 'agent'
    sender_name VARCHAR(100), -- User name or agent type (revise, investigate, etc.)
    message_text TEXT NOT NULL,
    message_type VARCHAR(50) DEFAULT 'text', -- text, idea_update, file_reference
    metadata JSONB DEFAULT '{}', -- Agent reasoning, references, etc.
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Agent interactions table (historical record)
CREATE TABLE agent_interactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    idea_id UUID REFERENCES ideas(id) ON DELETE CASCADE,
    session_id UUID REFERENCES chat_sessions(id),
    interaction_type VARCHAR(50) NOT NULL, -- 'revise', 'investigate', 'critique', 'expand', etc.
    agent_prompt TEXT,
    agent_response TEXT,
    model_used VARCHAR(100),
    context_documents UUID[], -- Array of document IDs used for context
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Search queries table (for analytics and learning)
CREATE TABLE search_queries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID REFERENCES users(id),
    campaign_id UUID REFERENCES campaigns(id),
    query_text TEXT NOT NULL,
    query_embedding vector(1536),
    results_count INTEGER,
    clicked_results UUID[], -- Array of idea/document IDs clicked
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Idea versions table (track idea evolution)
CREATE TABLE idea_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    idea_id UUID REFERENCES ideas(id) ON DELETE CASCADE,
    session_id UUID REFERENCES chat_sessions(id),
    version_number INTEGER NOT NULL,
    title VARCHAR(500),
    content TEXT,
    embedding vector(1536),
    changed_by VARCHAR(50), -- 'user' or 'agent:type'
    change_summary TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(idea_id, version_number)
);

-- Performance indexes
CREATE INDEX idx_users_username ON users(username);
CREATE INDEX idx_users_last_active ON users(last_active);

CREATE INDEX idx_campaigns_owner ON campaigns(owner_id);
CREATE INDEX idx_campaigns_active ON campaigns(is_active);
CREATE INDEX idx_campaigns_embedding ON campaigns USING ivfflat (embedding vector_cosine_ops);

CREATE INDEX idx_documents_campaign ON documents(campaign_id);
CREATE INDEX idx_documents_status ON documents(processing_status);
CREATE INDEX idx_documents_embedding ON documents USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX idx_documents_text_search ON documents USING GIN(to_tsvector('english', extracted_text));

CREATE INDEX idx_ideas_campaign ON ideas(campaign_id);
CREATE INDEX idx_ideas_status ON ideas(status);
CREATE INDEX idx_ideas_created_by ON ideas(created_by);
CREATE INDEX idx_ideas_tags ON ideas USING GIN(tags);
CREATE INDEX idx_ideas_metadata ON ideas USING GIN(metadata);
CREATE INDEX idx_ideas_content_search ON ideas USING GIN(to_tsvector('english', content));
CREATE INDEX idx_ideas_embedding ON ideas USING ivfflat (embedding vector_cosine_ops);

CREATE INDEX idx_chat_sessions_idea ON chat_sessions(idea_id);
CREATE INDEX idx_chat_sessions_user ON chat_sessions(user_id);
CREATE INDEX idx_chat_sessions_status ON chat_sessions(status);

CREATE INDEX idx_chat_messages_session ON chat_messages(session_id);
CREATE INDEX idx_chat_messages_created_at ON chat_messages(created_at);
CREATE INDEX idx_chat_messages_sender ON chat_messages(sender_type);

CREATE INDEX idx_agent_interactions_idea ON agent_interactions(idea_id);
CREATE INDEX idx_agent_interactions_session ON agent_interactions(session_id);
CREATE INDEX idx_agent_interactions_type ON agent_interactions(interaction_type);

CREATE INDEX idx_search_queries_user ON search_queries(user_id);
CREATE INDEX idx_search_queries_campaign ON search_queries(campaign_id);
CREATE INDEX idx_search_queries_embedding ON search_queries USING ivfflat (query_embedding vector_cosine_ops);

CREATE INDEX idx_idea_versions_idea ON idea_versions(idea_id);
CREATE INDEX idx_idea_versions_session ON idea_versions(session_id);

-- Triggers for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_campaigns_updated_at BEFORE UPDATE ON campaigns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_ideas_updated_at BEFORE UPDATE ON ideas
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_chat_sessions_updated_at BEFORE UPDATE ON chat_sessions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function for semantic search across ideas and documents
CREATE OR REPLACE FUNCTION search_ideas_and_documents(
    query_embedding vector(1536),
    campaign_filter UUID DEFAULT NULL,
    limit_results INTEGER DEFAULT 20
) RETURNS TABLE (
    result_type TEXT,
    result_id UUID,
    title TEXT,
    content TEXT,
    score FLOAT
) AS $$
BEGIN
    RETURN QUERY
    (
        SELECT 'idea'::TEXT as result_type, 
               i.id as result_id,
               i.title,
               i.content,
               (1 - (i.embedding <=> query_embedding)) as score
        FROM ideas i
        WHERE (campaign_filter IS NULL OR i.campaign_id = campaign_filter)
          AND i.embedding IS NOT NULL
        ORDER BY i.embedding <=> query_embedding
        LIMIT limit_results / 2
    )
    UNION ALL
    (
        SELECT 'document'::TEXT as result_type,
               d.id as result_id,
               d.original_name as title,
               d.extracted_text as content,
               (1 - (d.embedding <=> query_embedding)) as score
        FROM documents d
        WHERE (campaign_filter IS NULL OR d.campaign_id = campaign_filter)
          AND d.embedding IS NOT NULL
          AND d.processing_status = 'completed'
        ORDER BY d.embedding <=> query_embedding
        LIMIT limit_results / 2
    )
    ORDER BY score DESC;
END;
$$ LANGUAGE plpgsql;