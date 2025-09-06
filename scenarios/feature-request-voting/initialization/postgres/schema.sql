-- Feature Request Voting Schema
-- Multi-tenant feature request and voting system

-- Create UUID extension if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Scenarios table (multi-tenant support)
CREATE TABLE IF NOT EXISTS scenarios (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    settings JSONB DEFAULT '{}',
    auth_config JSONB DEFAULT '{"mode": "public"}', -- public, authenticated, custom
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Users table (simplified, integrates with scenario-authenticator)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    external_id VARCHAR(255), -- ID from scenario-authenticator
    email VARCHAR(255),
    username VARCHAR(255),
    display_name VARCHAR(255),
    avatar_url TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(external_id)
);

-- Feature request statuses enum
CREATE TYPE feature_status AS ENUM (
    'proposed',
    'under_review', 
    'in_development',
    'shipped',
    'wont_fix'
);

-- Priority levels enum
CREATE TYPE priority_level AS ENUM (
    'low',
    'medium', 
    'high',
    'critical'
);

-- Feature requests table
CREATE TABLE IF NOT EXISTS feature_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id UUID NOT NULL REFERENCES scenarios(id) ON DELETE CASCADE,
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,
    status feature_status DEFAULT 'proposed',
    priority priority_level DEFAULT 'medium',
    creator_id UUID REFERENCES users(id) ON DELETE SET NULL,
    vote_count INTEGER DEFAULT 0,
    comment_count INTEGER DEFAULT 0,
    position INTEGER DEFAULT 0, -- For ordering within status columns
    tags TEXT[] DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    shipped_at TIMESTAMP,
    archived_at TIMESTAMP,
    INDEX idx_scenario_status (scenario_id, status),
    INDEX idx_vote_count (vote_count DESC),
    INDEX idx_created_at (created_at DESC)
);

-- Votes table
CREATE TABLE IF NOT EXISTS votes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    feature_request_id UUID NOT NULL REFERENCES feature_requests(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    session_id VARCHAR(255), -- For anonymous voting
    value INTEGER NOT NULL CHECK (value IN (-1, 1)),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(feature_request_id, user_id),
    UNIQUE(feature_request_id, session_id),
    INDEX idx_feature_request (feature_request_id)
);

-- Comments table
CREATE TABLE IF NOT EXISTS comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    feature_request_id UUID NOT NULL REFERENCES feature_requests(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    parent_id UUID REFERENCES comments(id) ON DELETE CASCADE, -- For threaded comments
    content TEXT NOT NULL,
    is_edited BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP,
    INDEX idx_feature_request (feature_request_id),
    INDEX idx_parent (parent_id)
);

-- Status change history
CREATE TABLE IF NOT EXISTS status_changes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    feature_request_id UUID NOT NULL REFERENCES feature_requests(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    from_status feature_status,
    to_status feature_status NOT NULL,
    reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_feature_request (feature_request_id)
);

-- Scenario permissions (who can vote/propose)
CREATE TABLE IF NOT EXISTS scenario_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id UUID NOT NULL REFERENCES scenarios(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL, -- owner, moderator, contributor
    can_propose BOOLEAN DEFAULT TRUE,
    can_vote BOOLEAN DEFAULT TRUE,
    can_moderate BOOLEAN DEFAULT FALSE,
    can_edit_settings BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(scenario_id, user_id),
    INDEX idx_scenario_user (scenario_id, user_id)
);

-- Analytics events
CREATE TABLE IF NOT EXISTS analytics_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id UUID REFERENCES scenarios(id) ON DELETE CASCADE,
    feature_request_id UUID REFERENCES feature_requests(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id) ON DELETE SET NULL,
    event_type VARCHAR(50) NOT NULL, -- view, vote, comment, status_change
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    INDEX idx_scenario_event (scenario_id, event_type),
    INDEX idx_created_at (created_at DESC)
);

-- Functions for updating timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_scenarios_updated_at BEFORE UPDATE ON scenarios
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_feature_requests_updated_at BEFORE UPDATE ON feature_requests
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_comments_updated_at BEFORE UPDATE ON comments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to update vote counts
CREATE OR REPLACE FUNCTION update_vote_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE feature_requests 
        SET vote_count = vote_count + NEW.value
        WHERE id = NEW.feature_request_id;
    ELSIF TG_OP = 'UPDATE' THEN
        UPDATE feature_requests 
        SET vote_count = vote_count - OLD.value + NEW.value
        WHERE id = NEW.feature_request_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE feature_requests 
        SET vote_count = vote_count - OLD.value
        WHERE id = OLD.feature_request_id;
    END IF;
    RETURN NULL;
END;
$$ language 'plpgsql';

-- Trigger for vote count updates
CREATE TRIGGER update_feature_request_vote_count
AFTER INSERT OR UPDATE OR DELETE ON votes
    FOR EACH ROW EXECUTE FUNCTION update_vote_count();

-- Function to update comment counts
CREATE OR REPLACE FUNCTION update_comment_count()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE feature_requests 
        SET comment_count = comment_count + 1
        WHERE id = NEW.feature_request_id;
    ELSIF TG_OP = 'DELETE' THEN
        UPDATE feature_requests 
        SET comment_count = comment_count - 1
        WHERE id = OLD.feature_request_id;
    END IF;
    RETURN NULL;
END;
$$ language 'plpgsql';

-- Trigger for comment count updates
CREATE TRIGGER update_feature_request_comment_count
AFTER INSERT OR DELETE ON comments
    FOR EACH ROW EXECUTE FUNCTION update_comment_count();

-- Indexes for performance
CREATE INDEX idx_feature_requests_scenario_created ON feature_requests(scenario_id, created_at DESC);
CREATE INDEX idx_feature_requests_scenario_votes ON feature_requests(scenario_id, vote_count DESC);
CREATE INDEX idx_votes_user ON votes(user_id);
CREATE INDEX idx_comments_user ON comments(user_id);
CREATE INDEX idx_analytics_date_range ON analytics_events(scenario_id, created_at);

-- Views for common queries
CREATE OR REPLACE VIEW feature_requests_with_user_votes AS
SELECT 
    fr.*,
    v.user_id as voter_user_id,
    v.value as user_vote
FROM feature_requests fr
LEFT JOIN votes v ON fr.id = v.feature_request_id;

CREATE OR REPLACE VIEW feature_requests_summary AS
SELECT 
    s.name as scenario_name,
    fr.status,
    COUNT(*) as count,
    SUM(fr.vote_count) as total_votes
FROM feature_requests fr
JOIN scenarios s ON fr.scenario_id = s.id
GROUP BY s.name, fr.status;