-- Task Planner Database Schema
-- AI-powered task management with progressive refinement

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "vector";

-- Enum types
CREATE TYPE task_status AS ENUM ('unstructured', 'backlog', 'staged', 'in_progress', 'completed', 'cancelled', 'failed');
CREATE TYPE task_priority AS ENUM ('critical', 'high', 'medium', 'low');
CREATE TYPE agent_action AS ENUM ('parse', 'research', 'plan', 'implement', 'review');
CREATE TYPE app_type AS ENUM ('vrooli', 'scenario', 'generated', 'external');

-- Applications being managed
CREATE TABLE apps (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    type app_type DEFAULT 'external',
    repository_url TEXT,
    documentation_url TEXT,
    
    -- Configuration
    api_token VARCHAR(255) UNIQUE NOT NULL,
    webhook_url TEXT,
    auto_implement BOOLEAN DEFAULT FALSE,
    
    -- Statistics
    total_tasks INTEGER DEFAULT 0,
    completed_tasks INTEGER DEFAULT 0,
    avg_completion_hours DECIMAL(10,2),
    
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Unstructured input sessions
CREATE TABLE unstructured_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    app_id UUID REFERENCES apps(id) ON DELETE CASCADE,
    
    -- Raw input
    raw_text TEXT NOT NULL,
    input_type VARCHAR(50) DEFAULT 'markdown', -- markdown, plaintext, structured_list
    
    -- Processing status
    processed BOOLEAN DEFAULT FALSE,
    tasks_extracted INTEGER DEFAULT 0,
    
    -- User who submitted
    submitted_by VARCHAR(255),
    submitted_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMP
);

-- Tasks in various stages
CREATE TABLE tasks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    app_id UUID REFERENCES apps(id) ON DELETE CASCADE,
    session_id UUID REFERENCES unstructured_sessions(id) ON DELETE SET NULL,
    parent_task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    
    -- Core task info
    title VARCHAR(500) NOT NULL,
    description TEXT,
    status task_status DEFAULT 'unstructured',
    priority task_priority DEFAULT 'medium',
    
    -- Research & planning data
    investigation_report TEXT,
    implementation_plan TEXT,
    technical_requirements TEXT,
    estimated_hours DECIMAL(10,2),
    confidence_score DECIMAL(3,2), -- 0.00 to 1.00
    
    -- Implementation data
    implementation_code TEXT,
    implementation_result TEXT,
    test_results TEXT,
    
    -- Vector embeddings for similarity
    title_embedding VECTOR(1536),
    content_embedding VECTOR(1536),
    
    -- Dependencies and relationships
    depends_on UUID[], -- Array of task IDs
    blocks UUID[], -- Array of task IDs this blocks
    
    -- Tags and categorization
    tags TEXT[] DEFAULT '{}',
    labels JSONB DEFAULT '{}'::jsonb,
    
    -- Assignment and tracking
    assigned_to VARCHAR(255),
    assigned_at TIMESTAMP,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Agent processing history
CREATE TABLE agent_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    action agent_action NOT NULL,
    
    -- Execution details
    agent_name VARCHAR(255),
    model_used VARCHAR(100),
    prompt_template TEXT,
    input_data JSONB,
    output_data JSONB,
    
    -- Performance metrics
    tokens_used INTEGER,
    execution_time_ms INTEGER,
    cost_usd DECIMAL(10,4),
    
    -- Results
    successful BOOLEAN DEFAULT FALSE,
    error_message TEXT,
    confidence DECIMAL(3,2),
    
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- Task transitions (audit trail)
CREATE TABLE task_transitions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    
    from_status task_status,
    to_status task_status NOT NULL,
    
    -- Who/what triggered the transition
    triggered_by VARCHAR(255), -- user, agent, system
    trigger_type VARCHAR(50), -- manual, automated, scheduled
    
    -- Reasoning
    reason TEXT,
    notes TEXT,
    
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Research artifacts
CREATE TABLE research_artifacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    task_id UUID REFERENCES tasks(id) ON DELETE CASCADE,
    
    -- Artifact details
    type VARCHAR(50), -- documentation, code_example, api_reference, tutorial
    source_url TEXT,
    title VARCHAR(500),
    content TEXT,
    
    -- Relevance and quality
    relevance_score DECIMAL(3,2),
    quality_score DECIMAL(3,2),
    
    -- Vector embedding for similarity search
    content_embedding VECTOR(1536),
    
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Task templates (for recurring patterns)
CREATE TABLE task_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    app_id UUID REFERENCES apps(id) ON DELETE CASCADE,
    
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Template content
    title_template VARCHAR(500),
    description_template TEXT,
    investigation_template TEXT,
    implementation_template TEXT,
    
    -- Auto-configuration
    default_priority task_priority DEFAULT 'medium',
    default_tags TEXT[] DEFAULT '{}',
    default_labels JSONB DEFAULT '{}'::jsonb,
    
    -- Usage tracking
    times_used INTEGER DEFAULT 0,
    last_used_at TIMESTAMP,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_tasks_app ON tasks(app_id);
CREATE INDEX idx_tasks_status ON tasks(status);
CREATE INDEX idx_tasks_priority ON tasks(priority);
CREATE INDEX idx_tasks_created ON tasks(created_at DESC);
CREATE INDEX idx_tasks_parent ON tasks(parent_task_id);
CREATE INDEX idx_tasks_title_embedding ON tasks USING ivfflat (title_embedding vector_cosine_ops);
CREATE INDEX idx_tasks_content_embedding ON tasks USING ivfflat (content_embedding vector_cosine_ops);
CREATE INDEX idx_tasks_title_trgm ON tasks USING gin (title gin_trgm_ops);
CREATE INDEX idx_tasks_tags ON tasks USING gin (tags);

CREATE INDEX idx_sessions_app ON unstructured_sessions(app_id);
CREATE INDEX idx_sessions_processed ON unstructured_sessions(processed);

CREATE INDEX idx_agent_runs_task ON agent_runs(task_id);
CREATE INDEX idx_agent_runs_action ON agent_runs(action);

CREATE INDEX idx_transitions_task ON task_transitions(task_id);
CREATE INDEX idx_transitions_created ON task_transitions(created_at DESC);

CREATE INDEX idx_research_task ON research_artifacts(task_id);
CREATE INDEX idx_research_embedding ON research_artifacts USING ivfflat (content_embedding vector_cosine_ops);

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_apps_updated_at BEFORE UPDATE ON apps
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_tasks_updated_at BEFORE UPDATE ON tasks
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_templates_updated_at BEFORE UPDATE ON task_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();