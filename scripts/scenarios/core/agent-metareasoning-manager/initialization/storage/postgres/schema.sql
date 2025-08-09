-- Agent Metareasoning Manager Database Schema
-- Purpose: Store prompts, workflows, and reasoning patterns for enhanced agent decision-making

-- Enable necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Reasoning patterns enum
CREATE TYPE reasoning_pattern AS ENUM (
    'pros_cons_analysis',
    'swot_analysis',
    'risk_assessment',
    'devil_advocate',
    'introspection',
    'multi_perspective',
    'assumption_checking',
    'bias_detection',
    'yes_man_avoidance',
    'critical_thinking',
    'creative_exploration',
    'systematic_review'
);

-- Prompts table
CREATE TABLE prompts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    pattern reasoning_pattern,
    content TEXT NOT NULL,
    variables JSONB DEFAULT '[]', -- Variables that can be injected
    metadata JSONB DEFAULT '{}',
    version INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    tags TEXT[],
    usage_count INTEGER DEFAULT 0,
    average_rating DECIMAL(3,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID
);

-- Workflows table (n8n and Windmill workflows)
CREATE TABLE workflows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    platform VARCHAR(50) NOT NULL, -- 'n8n' or 'windmill'
    pattern reasoning_pattern,
    workflow_data JSONB NOT NULL, -- Full workflow JSON
    input_schema JSONB DEFAULT '{}',
    output_schema JSONB DEFAULT '{}',
    dependencies TEXT[], -- Other workflows this depends on
    version INTEGER DEFAULT 1,
    is_active BOOLEAN DEFAULT true,
    tags TEXT[],
    execution_count INTEGER DEFAULT 0,
    average_duration_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID
);

-- Templates table (reusable reasoning templates)
CREATE TABLE templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    pattern reasoning_pattern NOT NULL,
    structure JSONB NOT NULL, -- Template structure with placeholders
    example_usage TEXT,
    best_practices TEXT,
    limitations TEXT,
    tags TEXT[],
    is_public BOOLEAN DEFAULT false,
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID
);

-- Reasoning chains table (sequences of reasoning steps)
CREATE TABLE reasoning_chains (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    steps JSONB NOT NULL, -- Array of steps with prompt/workflow references
    input_requirements JSONB DEFAULT '{}',
    output_format JSONB DEFAULT '{}',
    use_cases TEXT[],
    tags TEXT[],
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID
);

-- Execution history table
CREATE TABLE execution_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_type VARCHAR(50) NOT NULL, -- 'prompt', 'workflow', 'template', 'chain'
    resource_id UUID NOT NULL,
    input_data JSONB,
    output_data JSONB,
    execution_time_ms INTEGER,
    status VARCHAR(50), -- 'success', 'failed', 'timeout'
    error_message TEXT,
    model_used VARCHAR(100),
    metadata JSONB DEFAULT '{}',
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    executed_by UUID
);

-- Prompt versions table
CREATE TABLE prompt_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    prompt_id UUID REFERENCES prompts(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    content TEXT NOT NULL,
    variables JSONB DEFAULT '[]',
    change_summary TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    UNIQUE(prompt_id, version_number)
);

-- Workflow versions table
CREATE TABLE workflow_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID REFERENCES workflows(id) ON DELETE CASCADE,
    version_number INTEGER NOT NULL,
    workflow_data JSONB NOT NULL,
    change_summary TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    UNIQUE(workflow_id, version_number)
);

-- Collections table (organize prompts/workflows)
CREATE TABLE collections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    type VARCHAR(50), -- 'prompts', 'workflows', 'mixed'
    items JSONB DEFAULT '[]', -- Array of {type, id} references
    tags TEXT[],
    is_public BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID
);

-- Ratings table
CREATE TABLE ratings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_type VARCHAR(50) NOT NULL,
    resource_id UUID NOT NULL,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    review TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID,
    UNIQUE(resource_type, resource_id, created_by)
);

-- Indexes for performance
CREATE INDEX idx_prompts_pattern ON prompts(pattern);
CREATE INDEX idx_prompts_category ON prompts(category);
CREATE INDEX idx_prompts_tags ON prompts USING GIN(tags);
CREATE INDEX idx_prompts_active ON prompts(is_active);
CREATE INDEX idx_workflows_platform ON workflows(platform);
CREATE INDEX idx_workflows_pattern ON workflows(pattern);
CREATE INDEX idx_workflows_tags ON workflows USING GIN(tags);
CREATE INDEX idx_templates_pattern ON templates(pattern);
CREATE INDEX idx_templates_public ON templates(is_public);
CREATE INDEX idx_execution_history_resource ON execution_history(resource_type, resource_id);
CREATE INDEX idx_execution_history_status ON execution_history(status);
CREATE INDEX idx_execution_history_date ON execution_history(executed_at);

-- Triggers for updated_at
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_prompts_updated_at BEFORE UPDATE ON prompts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflows_updated_at BEFORE UPDATE ON workflows
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_templates_updated_at BEFORE UPDATE ON templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_collections_updated_at BEFORE UPDATE ON collections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- API-specific tables for CLI/API functionality

-- API tokens table (for CLI authentication)
CREATE TABLE api_tokens (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    token_hash VARCHAR(255) UNIQUE NOT NULL,
    permissions JSONB DEFAULT '{}', -- {read: true, write: false, admin: false}
    scopes TEXT[] DEFAULT '{}', -- ['prompts', 'workflows', 'analyze']
    expires_at TIMESTAMP WITH TIME ZONE,
    last_used_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    created_by UUID
);

-- Execution status enum for API operations
CREATE TYPE execution_status AS ENUM (
    'pending',
    'running', 
    'completed',
    'failed',
    'timeout',
    'cancelled'
);

-- Analysis type enum for API endpoints
CREATE TYPE analysis_type AS ENUM (
    'decision',
    'pros_cons',
    'swot',
    'risk_assessment',
    'reasoning_chain',
    'template_application'
);

-- Analysis executions table (track API analysis requests)
CREATE TABLE analysis_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    type analysis_type NOT NULL,
    input_data JSONB NOT NULL,
    output_data JSONB,
    status execution_status DEFAULT 'pending',
    execution_time_ms INTEGER,
    workflow_id UUID, -- Reference to workflow used
    prompt_id UUID,   -- Reference to prompt used
    template_id UUID, -- Reference to template used
    error_message TEXT,
    n8n_execution_id VARCHAR(255), -- External n8n execution ID
    windmill_job_id VARCHAR(255),  -- External Windmill job ID
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    executed_by UUID,
    api_token_id UUID REFERENCES api_tokens(id)
);

-- API usage statistics table
CREATE TABLE api_usage_stats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    endpoint VARCHAR(255) NOT NULL,
    method VARCHAR(10) NOT NULL,
    response_code INTEGER NOT NULL,
    execution_time_ms INTEGER NOT NULL,
    request_size_bytes INTEGER,
    response_size_bytes INTEGER,
    user_agent VARCHAR(500),
    ip_address INET,
    api_token_id UUID REFERENCES api_tokens(id),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Workflow execution queue table
CREATE TABLE workflow_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    analysis_execution_id UUID REFERENCES analysis_executions(id) ON DELETE CASCADE,
    workflow_name VARCHAR(255) NOT NULL,
    platform VARCHAR(50) NOT NULL, -- 'n8n' or 'windmill'
    input_payload JSONB NOT NULL,
    priority INTEGER DEFAULT 5, -- 1-10 priority scale
    status execution_status DEFAULT 'pending',
    scheduled_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    external_id VARCHAR(255), -- n8n/windmill execution ID
    result_data JSONB,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3
);

-- User sessions table (optional - for future web UI)
CREATE TABLE user_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    session_token VARCHAR(255) UNIQUE NOT NULL,
    user_identifier VARCHAR(255) NOT NULL, -- Could be email, username, etc.
    metadata JSONB DEFAULT '{}',
    expires_at TIMESTAMP WITH TIME ZONE NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_accessed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- API configuration table
CREATE TABLE api_config (
    key VARCHAR(255) PRIMARY KEY,
    value JSONB NOT NULL,
    description TEXT,
    is_sensitive BOOLEAN DEFAULT false,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_by UUID
);

-- Additional indexes for API performance
CREATE INDEX idx_api_tokens_hash ON api_tokens(token_hash);
CREATE INDEX idx_api_tokens_active ON api_tokens(is_active);
CREATE INDEX idx_analysis_executions_status ON analysis_executions(status);
CREATE INDEX idx_analysis_executions_type ON analysis_executions(type);
CREATE INDEX idx_analysis_executions_created ON analysis_executions(created_at);
CREATE INDEX idx_api_usage_endpoint ON api_usage_stats(endpoint);
CREATE INDEX idx_api_usage_created ON api_usage_stats(created_at);
CREATE INDEX idx_workflow_queue_status ON workflow_queue(status);
CREATE INDEX idx_workflow_queue_priority ON workflow_queue(priority, scheduled_at);
CREATE INDEX idx_user_sessions_token ON user_sessions(session_token);
CREATE INDEX idx_user_sessions_expires ON user_sessions(expires_at);

-- Triggers for API tables
CREATE TRIGGER update_api_config_updated_at BEFORE UPDATE ON api_config
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Foreign key constraints (optional references)
ALTER TABLE analysis_executions 
    ADD CONSTRAINT fk_analysis_workflow 
    FOREIGN KEY (workflow_id) REFERENCES workflows(id) ON DELETE SET NULL;

ALTER TABLE analysis_executions 
    ADD CONSTRAINT fk_analysis_prompt 
    FOREIGN KEY (prompt_id) REFERENCES prompts(id) ON DELETE SET NULL;

ALTER TABLE analysis_executions 
    ADD CONSTRAINT fk_analysis_template 
    FOREIGN KEY (template_id) REFERENCES templates(id) ON DELETE SET NULL;