-- Drop the incorrect notes table and recreate with API schema
DROP TABLE IF EXISTS notes CASCADE;

-- Create correct notes table for API Library
CREATE TABLE notes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('gotcha', 'tip', 'warning', 'example', 'success', 'failure')),
    
    -- Optional reference to specific endpoint
    endpoint_id UUID REFERENCES endpoints(id) ON DELETE CASCADE,
    
    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255) DEFAULT 'system',
    scenario_source VARCHAR(255), -- Which scenario discovered this
    
    -- Voting system for usefulness
    helpful_count INTEGER DEFAULT 0,
    not_helpful_count INTEGER DEFAULT 0
);

-- Create index
CREATE INDEX idx_notes_api_id ON notes(api_id);
CREATE INDEX idx_notes_type ON notes(type);

-- Create research_requests table
CREATE TABLE IF NOT EXISTS research_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    capability TEXT NOT NULL,
    requirements JSONB,
    status VARCHAR(50) DEFAULT 'queued' CHECK (status IN ('queued', 'in_progress', 'completed', 'failed')),
    
    -- Results
    apis_discovered INTEGER DEFAULT 0,
    completion_time TIMESTAMP,
    error_message TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create api_usage_logs table
CREATE TABLE IF NOT EXISTS api_usage_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID REFERENCES apis(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL CHECK (action IN ('search', 'view', 'configure', 'use', 'note_added')),
    scenario_name VARCHAR(255),
    search_query TEXT,
    result_count INTEGER,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_usage_logs_api_id ON api_usage_logs(api_id);
CREATE INDEX idx_usage_logs_action ON api_usage_logs(action);
CREATE INDEX idx_usage_logs_created ON api_usage_logs(created_at);