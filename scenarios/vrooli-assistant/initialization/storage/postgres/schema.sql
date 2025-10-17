-- Vrooli Assistant Database Schema
-- Stores captured issues, agent sessions, and pattern analysis

-- Create database if not exists (run as superuser)
-- CREATE DATABASE vrooli_assistant;

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Issues table - stores all captured issues
CREATE TABLE IF NOT EXISTS issues (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    timestamp TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    screenshot_path TEXT,
    scenario_name VARCHAR(255),
    url TEXT,
    description TEXT NOT NULL,
    context_data JSONB,
    agent_session_id UUID,
    status VARCHAR(50) NOT NULL DEFAULT 'captured',
    resolution_notes TEXT,
    tags TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT status_check CHECK (status IN ('captured', 'assigned', 'fixing', 'resolved', 'failed', 'deferred'))
);

-- Agent sessions table - tracks agent work on issues
CREATE TABLE IF NOT EXISTS agent_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    issue_id UUID REFERENCES issues(id) ON DELETE CASCADE,
    agent_type VARCHAR(50) NOT NULL,
    start_time TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    end_time TIMESTAMP,
    status VARCHAR(50) NOT NULL DEFAULT 'running',
    output TEXT,
    context_provided JSONB,
    resources_used JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT status_check CHECK (status IN ('running', 'completed', 'failed', 'cancelled'))
);

-- Patterns table - stores recognized issue patterns
CREATE TABLE IF NOT EXISTS patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pattern_type VARCHAR(100) NOT NULL,
    pattern_signature TEXT NOT NULL,
    frequency INTEGER DEFAULT 1,
    affected_scenarios TEXT[],
    suggested_fix TEXT,
    embedding VECTOR(1536), -- For similarity search with qdrant
    first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Issue relationships - links similar or related issues
CREATE TABLE IF NOT EXISTS issue_relationships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    issue_id UUID REFERENCES issues(id) ON DELETE CASCADE,
    related_issue_id UUID REFERENCES issues(id) ON DELETE CASCADE,
    relationship_type VARCHAR(50) NOT NULL,
    confidence FLOAT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT relationship_check CHECK (relationship_type IN ('duplicate', 'related', 'caused_by', 'blocks'))
);

-- Agent performance metrics
CREATE TABLE IF NOT EXISTS agent_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    agent_type VARCHAR(50) NOT NULL,
    issue_type VARCHAR(100),
    success_count INTEGER DEFAULT 0,
    failure_count INTEGER DEFAULT 0,
    avg_resolution_time INTERVAL,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User preferences and settings
CREATE TABLE IF NOT EXISTS user_settings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    setting_key VARCHAR(100) UNIQUE NOT NULL,
    setting_value JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_issues_status ON issues(status);
CREATE INDEX IF NOT EXISTS idx_issues_scenario ON issues(scenario_name);
CREATE INDEX IF NOT EXISTS idx_issues_timestamp ON issues(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_issues_tags ON issues USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_issues_context ON issues USING GIN(context_data);

CREATE INDEX IF NOT EXISTS idx_agent_sessions_issue ON agent_sessions(issue_id);
CREATE INDEX IF NOT EXISTS idx_agent_sessions_status ON agent_sessions(status);
CREATE INDEX IF NOT EXISTS idx_agent_sessions_type ON agent_sessions(agent_type);

CREATE INDEX IF NOT EXISTS idx_patterns_type ON patterns(pattern_type);
CREATE INDEX IF NOT EXISTS idx_patterns_scenarios ON patterns USING GIN(affected_scenarios);

-- Create triggers for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_issues_updated_at BEFORE UPDATE ON issues
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_user_settings_updated_at BEFORE UPDATE ON user_settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create views for common queries
CREATE OR REPLACE VIEW recent_issues AS
SELECT 
    i.*,
    s.agent_type,
    s.status as agent_status,
    s.start_time as agent_start_time,
    s.end_time as agent_end_time
FROM issues i
LEFT JOIN agent_sessions s ON i.agent_session_id = s.id
ORDER BY i.timestamp DESC
LIMIT 100;

CREATE OR REPLACE VIEW issue_statistics AS
SELECT 
    scenario_name,
    COUNT(*) as total_issues,
    COUNT(CASE WHEN status = 'resolved' THEN 1 END) as resolved_issues,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_issues,
    COUNT(CASE WHEN status IN ('captured', 'assigned', 'fixing') THEN 1 END) as pending_issues,
    AVG(EXTRACT(EPOCH FROM (updated_at - created_at))) as avg_resolution_seconds
FROM issues
GROUP BY scenario_name;

CREATE OR REPLACE VIEW agent_performance AS
SELECT 
    agent_type,
    COUNT(*) as total_sessions,
    COUNT(CASE WHEN status = 'completed' THEN 1 END) as successful_sessions,
    COUNT(CASE WHEN status = 'failed' THEN 1 END) as failed_sessions,
    AVG(EXTRACT(EPOCH FROM (end_time - start_time))) as avg_session_seconds
FROM agent_sessions
WHERE end_time IS NOT NULL
GROUP BY agent_type;

-- Insert default settings
INSERT INTO user_settings (setting_key, setting_value) VALUES
    ('hotkey', '{"value": "Cmd+Shift+Space", "platform": "all"}'::jsonb),
    ('default_agent_type', '{"value": "claude-code"}'::jsonb),
    ('auto_spawn_agent', '{"value": true}'::jsonb),
    ('screenshot_quality', '{"value": "high"}'::jsonb),
    ('max_history_items', '{"value": 100}'::jsonb)
ON CONFLICT (setting_key) DO NOTHING;