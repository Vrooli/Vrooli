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