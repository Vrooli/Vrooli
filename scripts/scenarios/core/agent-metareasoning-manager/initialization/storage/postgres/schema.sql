-- Agent Metareasoning Manager Database Schema
-- Lightweight metadata registry for workflow discovery and tracking

-- Workflow registry: Metadata and references to workflows in n8n/Windmill
CREATE TABLE IF NOT EXISTS workflow_registry (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform VARCHAR(20) NOT NULL CHECK (platform IN ('n8n', 'windmill')),
    platform_id VARCHAR(255) NOT NULL,  -- The actual workflow ID in n8n/Windmill
    name VARCHAR(100) NOT NULL,
    description TEXT,
    category VARCHAR(50),  -- pros-cons, swot, risk-assessment, etc.
    tags TEXT[],
    
    -- For semantic search
    embedding_id VARCHAR(100),  -- Reference to Qdrant vector
    capabilities JSONB,  -- What this workflow can do
    
    -- Discovery metadata  
    input_schema JSONB,  -- Expected inputs
    output_schema JSONB,  -- Expected outputs
    
    -- Usage tracking (lightweight)
    usage_count INTEGER DEFAULT 0,
    last_used TIMESTAMP WITH TIME ZONE,
    avg_execution_time_ms INTEGER,
    
    -- Timestamps
    discovered_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(platform, platform_id)
);

-- Create indexes for workflow registry
CREATE INDEX idx_registry_platform ON workflow_registry(platform);
CREATE INDEX idx_registry_category ON workflow_registry(category);
CREATE INDEX idx_registry_tags ON workflow_registry USING GIN(tags);
CREATE INDEX idx_registry_usage ON workflow_registry(usage_count DESC);

-- Execution log: Minimal tracking for coordination and metrics
CREATE TABLE IF NOT EXISTS execution_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID REFERENCES workflow_registry(id) ON DELETE CASCADE,
    platform VARCHAR(20) NOT NULL,
    platform_execution_id VARCHAR(255),  -- Platform's execution ID for reference
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'running', 'success', 'failed', 'timeout')),
    execution_time_ms INTEGER,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Agent preferences: Track which workflows agents prefer for different tasks
CREATE TABLE IF NOT EXISTS agent_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id VARCHAR(100) NOT NULL,
    task_type VARCHAR(50) NOT NULL,
    workflow_id UUID REFERENCES workflow_registry(id) ON DELETE CASCADE,
    preference_score DECIMAL(3,2) DEFAULT 0.5,
    last_used TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(agent_id, task_type, workflow_id)
);

-- Removed: workflow_metrics table - platforms track their own metrics
-- We only track lightweight usage in workflow_registry

-- Semantic search patterns: For finding similar workflows
CREATE TABLE IF NOT EXISTS search_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_text TEXT NOT NULL,
    embedding_id VARCHAR(100),  -- Reference to Qdrant vector
    matched_workflows UUID[],  -- Array of workflow_registry IDs
    search_count INTEGER DEFAULT 1,
    last_searched TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(pattern_text)
);

-- Removed: workflow_chains and model_performance tables
-- These are better tracked by the platforms themselves

-- Create indexes for execution log
CREATE INDEX idx_execution_log_workflow ON execution_log(workflow_id);
CREATE INDEX idx_execution_log_status ON execution_log(status);
CREATE INDEX idx_execution_log_started ON execution_log(started_at DESC);

-- Create indexes for agent preferences
CREATE INDEX idx_agent_preferences_agent ON agent_preferences(agent_id);
CREATE INDEX idx_agent_preferences_task ON agent_preferences(task_type);

-- Create indexes for search patterns
CREATE INDEX idx_search_patterns_embedding ON search_patterns(embedding_id);
CREATE INDEX idx_search_patterns_count ON search_patterns(search_count DESC);

-- Update triggers for updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_workflow_registry_updated_at BEFORE UPDATE ON workflow_registry
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to update workflow registry usage stats after execution
CREATE OR REPLACE FUNCTION update_workflow_usage_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status = 'success' AND NEW.workflow_id IS NOT NULL THEN
        UPDATE workflow_registry SET
            usage_count = usage_count + 1,
            last_used = NEW.completed_at,
            avg_execution_time_ms = CASE
                WHEN usage_count = 0 THEN NEW.execution_time_ms
                ELSE ((avg_execution_time_ms * usage_count + NEW.execution_time_ms) / (usage_count + 1))::INTEGER
            END,
            updated_at = CURRENT_TIMESTAMP
        WHERE id = NEW.workflow_id;
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_usage_on_execution_complete
    AFTER UPDATE OF status ON execution_log
    FOR EACH ROW
    WHEN (NEW.status = 'success' AND OLD.status != 'success')
    EXECUTE FUNCTION update_workflow_usage_stats();