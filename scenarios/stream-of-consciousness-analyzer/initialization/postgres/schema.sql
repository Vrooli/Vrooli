-- Stream of Consciousness Analyzer Database Schema

-- Create database if not exists
SELECT 'CREATE DATABASE soc_analyzer'
WHERE NOT EXISTS (SELECT FROM pg_database WHERE datname = 'soc_analyzer')\gexec

-- Connect to the database
\c soc_analyzer;

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Campaigns table (for organizing thought contexts)
CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    context_prompt TEXT, -- AI guidance for this campaign
    color VARCHAR(7) DEFAULT '#7C3AED', -- For UI theming
    icon VARCHAR(50) DEFAULT 'brain', -- Icon identifier
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Stream entries (raw input - enhanced for autonomous processing)
CREATE TABLE IF NOT EXISTS stream_entries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255) DEFAULT 'default',
    campaign VARCHAR(255) DEFAULT 'general',
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    raw_text TEXT NOT NULL,
    content TEXT NOT NULL, -- backwards compatibility
    type VARCHAR(50) DEFAULT 'text', -- text, voice, image
    source VARCHAR(100), -- manual, voice-recorder, api, etc.
    processed_data JSONB DEFAULT '{}',
    embedding_vector JSONB,
    insight_embedding JSONB,
    theme_embedding JSONB,
    insight_score FLOAT DEFAULT 0,
    metadata JSONB DEFAULT '{}',
    processed BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Organized notes (processed output)
CREATE TABLE IF NOT EXISTS organized_notes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    stream_entry_id UUID REFERENCES stream_entries(id) ON DELETE SET NULL,
    title VARCHAR(500),
    content TEXT NOT NULL,
    summary TEXT,
    category VARCHAR(100),
    tags TEXT[],
    priority INTEGER DEFAULT 3, -- 1-5 scale
    embedding_id VARCHAR(255), -- Reference to vector in Qdrant
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Extracted insights (patterns and key information)
CREATE TABLE IF NOT EXISTS insights (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    note_ids UUID[], -- Related organized notes
    insight_type VARCHAR(100), -- action_item, decision, idea, pattern, etc.
    content TEXT NOT NULL,
    confidence FLOAT DEFAULT 0.8,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Context documents (for campaign context)
CREATE TABLE IF NOT EXISTS context_documents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    content TEXT,
    document_type VARCHAR(50), -- reference, template, guide, etc.
    embedding_id VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Processing history (for debugging and improvement)
CREATE TABLE IF NOT EXISTS processing_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    stream_entry_id UUID REFERENCES stream_entries(id) ON DELETE CASCADE,
    workflow_name VARCHAR(100),
    status VARCHAR(50), -- started, completed, failed
    input_data JSONB,
    output_data JSONB,
    error_message TEXT,
    processing_time_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_stream_entries_campaign ON stream_entries(campaign_id);
CREATE INDEX idx_stream_entries_processed ON stream_entries(processed);
CREATE INDEX idx_organized_notes_campaign ON organized_notes(campaign_id);
CREATE INDEX idx_organized_notes_tags ON organized_notes USING GIN(tags);
CREATE INDEX idx_insights_campaign ON insights(campaign_id);
CREATE INDEX idx_context_documents_campaign ON context_documents(campaign_id);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at triggers
CREATE TRIGGER update_campaigns_updated_at BEFORE UPDATE ON campaigns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_organized_notes_updated_at BEFORE UPDATE ON organized_notes
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Insert default campaign
INSERT INTO campaigns (name, description, context_prompt, color, icon)
VALUES 
    ('General Thoughts', 'Default campaign for uncategorized thoughts', 
     'You are helping organize general thoughts and ideas. Focus on clarity and actionability.',
     '#7C3AED', 'brain'),
    ('Daily Reflections', 'Personal reflections and journal entries',
     'You are organizing personal reflections. Be empathetic and help identify patterns in thoughts and emotions.',
     '#EC4899', 'journal'),
    ('Work Ideas', 'Professional thoughts and project ideas',
     'You are organizing work-related thoughts. Focus on project relevance, priorities, and action items.',
     '#3B82F6', 'briefcase')
ON CONFLICT DO NOTHING;