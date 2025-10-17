-- Issue Tracker Database Schema
-- Centralized hub for tracking issues across all Vrooli-generated applications

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "vector";

-- Enum types
CREATE TYPE issue_status AS ENUM ('open', 'active', 'completed', 'failed', 'archived', 'wont_fix', 'duplicate');
CREATE TYPE issue_priority AS ENUM ('critical', 'high', 'medium', 'low');
CREATE TYPE issue_type AS ENUM ('bug', 'feature', 'improvement', 'documentation', 'performance', 'security');
CREATE TYPE agent_status AS ENUM ('idle', 'working', 'failed', 'completed');
CREATE TYPE app_status AS ENUM ('active', 'inactive', 'deprecated', 'testing');
CREATE TYPE agent_capability AS ENUM ('investigate', 'fix', 'test', 'document', 'refactor');

-- Applications that report issues to this tracker
CREATE TABLE apps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    scenario_name VARCHAR(255),  -- Which scenario generated this app
    type VARCHAR(50) CHECK (type IN ('vrooli-core', 'scenario', 'generated-app', 'external')),
    api_token VARCHAR(255) UNIQUE NOT NULL,  -- For authentication when reporting issues
    webhook_url TEXT,  -- Optional webhook for status updates
    status app_status DEFAULT 'active',
    
    -- Metadata about the app
    version VARCHAR(50),
    repository_url TEXT,
    deployment_path TEXT,
    environment VARCHAR(50),  -- dev, staging, prod
    
    -- Statistics
    total_issues INTEGER DEFAULT 0,
    open_issues INTEGER DEFAULT 0,
    avg_resolution_time_hours DECIMAL(10,2),
    
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Agent templates for different investigation/fix strategies
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Prompt engineering
    system_prompt TEXT NOT NULL,
    user_prompt_template TEXT NOT NULL,  -- Template with placeholders
    
    -- Capabilities and constraints
    capabilities agent_capability[] DEFAULT '{}',
    max_tokens INTEGER DEFAULT 4096,
    temperature DECIMAL(2,1) DEFAULT 0.2,
    model VARCHAR(100) DEFAULT 'gpt-5-nano',
    
    -- Performance tracking
    success_rate DECIMAL(3,2) DEFAULT 0.00,
    avg_completion_time_seconds INTEGER,
    total_runs INTEGER DEFAULT 0,
    successful_runs INTEGER DEFAULT 0,
    total_tokens_used BIGINT DEFAULT 0,
    total_cost_usd DECIMAL(10,4) DEFAULT 0.00,
    
    -- Configuration
    config JSONB DEFAULT '{}'::jsonb,
    tags TEXT[] DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Issues from all tracked applications
CREATE TABLE issues (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    app_id UUID REFERENCES apps(id) ON DELETE CASCADE,
    external_id VARCHAR(255),  -- ID from the source app if any
    
    -- Issue details
    title VARCHAR(500) NOT NULL,
    description TEXT NOT NULL,
    status issue_status DEFAULT 'open',
    priority issue_priority DEFAULT 'medium',
    type issue_type DEFAULT 'bug',
    
    -- Reporter info
    reporter_name VARCHAR(255),
    reporter_email VARCHAR(255),
    reporter_id VARCHAR(255),  -- User ID from source app
    
    -- Assignment
    assigned_agent_id UUID REFERENCES agents(id),
    assigned_at TIMESTAMP,
    
    -- Semantic search and similarity
    embedding VECTOR(1536),
    embedding_generated_at TIMESTAMP,
    
    -- Investigation data
    investigation_report TEXT,
    root_cause TEXT,
    suggested_fix TEXT,
    confidence_score DECIMAL(3,2),  -- 0.00 to 1.00
    
    -- Fix attempt data
    fix_attempted BOOLEAN DEFAULT FALSE,
    fix_successful BOOLEAN DEFAULT FALSE,
    fix_commit_hash VARCHAR(40),
    fix_pr_url TEXT,
    fix_verification_status VARCHAR(50),
    
    -- Error context
    error_message TEXT,
    error_logs TEXT,
    stack_trace TEXT,
    affected_files TEXT[],
    affected_components TEXT[],
    environment_info JSONB DEFAULT '{}'::jsonb,
    
    -- Categorization
    tags TEXT[] DEFAULT '{}',
    labels JSONB DEFAULT '{}'::jsonb,
    
    -- Pattern detection
    pattern_group_id UUID,  -- Groups similar issues
    is_recurring BOOLEAN DEFAULT FALSE,
    occurrence_count INTEGER DEFAULT 1,
    
    metadata JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP,
    first_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Agent runs (tracking each investigation/fix attempt)
CREATE TABLE agent_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    issue_id UUID REFERENCES issues(id) ON DELETE CASCADE,
    agent_id UUID REFERENCES agents(id),
    status agent_status DEFAULT 'idle',
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    duration_seconds INTEGER,
    
    -- Run details
    input_context TEXT,
    output_report TEXT,
    actions_taken JSONB DEFAULT '[]'::jsonb,
    files_modified TEXT[],
    tests_run INTEGER DEFAULT 0,
    tests_passed INTEGER DEFAULT 0,
    
    -- Resource usage
    tokens_used INTEGER,
    api_calls_made INTEGER,
    cost_estimate DECIMAL(10,4),
    
    -- Success metrics
    successful BOOLEAN DEFAULT FALSE,
    error_message TEXT,
    
    metadata JSONB DEFAULT '{}'::jsonb
);

-- Pattern groups for clustering similar issues across apps
CREATE TABLE pattern_groups (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255),
    description TEXT,
    pattern_type VARCHAR(50),  -- error_pattern, behavior_pattern, performance_pattern
    
    -- Pattern characteristics
    common_keywords TEXT[],
    common_stack_trace_patterns TEXT[],
    affected_app_ids UUID[] DEFAULT '{}',
    
    -- Statistics
    total_issues INTEGER DEFAULT 0,
    active_issues INTEGER DEFAULT 0,
    avg_resolution_time_hours DECIMAL(10,2),
    
    -- Solution tracking
    known_solutions TEXT[],
    successful_fix_count INTEGER DEFAULT 0,
    
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Issue comments/updates
CREATE TABLE issue_comments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    issue_id UUID REFERENCES issues(id) ON DELETE CASCADE,
    author_type VARCHAR(20) CHECK (author_type IN ('user', 'agent', 'system', 'app')),
    author_id VARCHAR(255),
    author_name VARCHAR(255),
    content TEXT NOT NULL,
    
    -- For tracking state changes
    is_state_change BOOLEAN DEFAULT FALSE,
    old_state issue_status,
    new_state issue_status,
    
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Related issues (for tracking duplicates or related problems)
CREATE TABLE related_issues (
    issue_id UUID REFERENCES issues(id) ON DELETE CASCADE,
    related_issue_id UUID REFERENCES issues(id) ON DELETE CASCADE,
    relationship_type VARCHAR(50) CHECK (relationship_type IN ('duplicate', 'blocks', 'blocked_by', 'related', 'causes', 'caused_by')),
    similarity_score DECIMAL(3,2),
    cross_app BOOLEAN DEFAULT FALSE,  -- Issues from different apps
    detected_by VARCHAR(50),  -- manual, semantic_search, pattern_detection
    PRIMARY KEY (issue_id, related_issue_id)
);

-- API request tracking for rate limiting and analytics
CREATE TABLE api_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    app_id UUID REFERENCES apps(id) ON DELETE CASCADE,
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    status_code INTEGER,
    response_time_ms INTEGER,
    request_body JSONB,
    response_body JSONB,
    ip_address INET,
    user_agent TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_issues_app ON issues(app_id);
CREATE INDEX idx_issues_status ON issues(status);
CREATE INDEX idx_issues_priority ON issues(priority);
CREATE INDEX idx_issues_type ON issues(type);
CREATE INDEX idx_issues_created ON issues(created_at DESC);
CREATE INDEX idx_issues_pattern_group ON issues(pattern_group_id);
CREATE INDEX idx_issues_external_id ON issues(app_id, external_id);
CREATE INDEX idx_issues_embedding ON issues USING ivfflat (embedding vector_cosine_ops);
CREATE INDEX idx_issues_title_trgm ON issues USING gin (title gin_trgm_ops);
CREATE INDEX idx_issues_description_trgm ON issues USING gin (description gin_trgm_ops);
CREATE INDEX idx_issues_tags ON issues USING gin (tags);

CREATE INDEX idx_agent_runs_issue ON agent_runs(issue_id);
CREATE INDEX idx_agent_runs_agent ON agent_runs(agent_id);
CREATE INDEX idx_agent_runs_status ON agent_runs(status);
CREATE INDEX idx_agent_runs_created ON agent_runs(started_at DESC);

CREATE INDEX idx_apps_api_token ON apps(api_token);
CREATE INDEX idx_apps_status ON apps(status);
CREATE INDEX idx_apps_scenario ON apps(scenario_name);

CREATE INDEX idx_comments_issue ON issue_comments(issue_id);
CREATE INDEX idx_comments_created ON issue_comments(created_at DESC);

CREATE INDEX idx_patterns_affected_apps ON pattern_groups USING gin (affected_app_ids);

CREATE INDEX idx_api_requests_app ON api_requests(app_id);
CREATE INDEX idx_api_requests_created ON api_requests(created_at DESC);

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_issues_updated_at BEFORE UPDATE ON issues
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_agents_updated_at BEFORE UPDATE ON agents
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_apps_updated_at BEFORE UPDATE ON apps
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pattern_groups_updated_at BEFORE UPDATE ON pattern_groups
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to update app statistics when issues change
CREATE OR REPLACE FUNCTION update_app_issue_stats()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE apps
    SET total_issues = (SELECT COUNT(*) FROM issues WHERE app_id = COALESCE(NEW.app_id, OLD.app_id)),
        open_issues = (SELECT COUNT(*) FROM issues WHERE app_id = COALESCE(NEW.app_id, OLD.app_id) AND status IN ('open', 'active'))
    WHERE id = COALESCE(NEW.app_id, OLD.app_id);
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_app_stats_on_issue_change
AFTER INSERT OR UPDATE OR DELETE ON issues
FOR EACH ROW EXECUTE FUNCTION update_app_issue_stats();
