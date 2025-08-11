-- Agent Metareasoning Manager Database Schema
-- Stores execution history, workflow metadata, and performance metrics

-- Execution history for tracking all reasoning workflow executions
CREATE TABLE IF NOT EXISTS execution_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(100) NOT NULL,
    workflow_type VARCHAR(50) NOT NULL,
    input_data JSONB NOT NULL,
    output_data JSONB,
    execution_time_ms INTEGER,
    status VARCHAR(20) NOT NULL CHECK (status IN ('pending', 'running', 'success', 'failed', 'timeout')),
    model_used VARCHAR(50),
    error_message TEXT,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Prompt usage tracking for effectiveness analysis
CREATE TABLE IF NOT EXISTS prompt_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    prompt_id VARCHAR(100) NOT NULL,
    prompt_collection VARCHAR(50) NOT NULL,
    execution_id UUID REFERENCES execution_history(id) ON DELETE CASCADE,
    effectiveness_score DECIMAL(3,2) CHECK (effectiveness_score >= 0 AND effectiveness_score <= 1),
    user_feedback JSONB,
    model_performance JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Workflow performance metrics for optimization
CREATE TABLE IF NOT EXISTS workflow_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_name VARCHAR(100) NOT NULL UNIQUE,
    workflow_platform VARCHAR(20) NOT NULL CHECK (workflow_platform IN ('n8n', 'windmill')),
    total_executions INTEGER DEFAULT 0,
    successful_executions INTEGER DEFAULT 0,
    failed_executions INTEGER DEFAULT 0,
    avg_execution_time_ms INTEGER,
    min_execution_time_ms INTEGER,
    max_execution_time_ms INTEGER,
    success_rate DECIMAL(5,2) GENERATED ALWAYS AS (
        CASE 
            WHEN total_executions > 0 THEN (successful_executions::DECIMAL / total_executions * 100)
            ELSE 0
        END
    ) STORED,
    last_execution TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Reasoning patterns for semantic search integration
CREATE TABLE IF NOT EXISTS reasoning_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_name VARCHAR(100) NOT NULL,
    pattern_type VARCHAR(50) NOT NULL,
    description TEXT,
    usage_count INTEGER DEFAULT 0,
    effectiveness_avg DECIMAL(3,2),
    embedding_id VARCHAR(100), -- Reference to Qdrant vector
    tags TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Workflow chains for complex reasoning sequences
CREATE TABLE IF NOT EXISTS workflow_chains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    chain_name VARCHAR(100) NOT NULL,
    chain_description TEXT,
    workflow_sequence JSONB NOT NULL, -- Array of workflow IDs in order
    total_executions INTEGER DEFAULT 0,
    avg_total_time_ms INTEGER,
    success_rate DECIMAL(5,2),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Model performance tracking
CREATE TABLE IF NOT EXISTS model_performance (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_name VARCHAR(50) NOT NULL,
    workflow_type VARCHAR(50) NOT NULL,
    total_uses INTEGER DEFAULT 0,
    avg_response_time_ms INTEGER,
    avg_token_count INTEGER,
    quality_score DECIMAL(3,2),
    cost_estimate DECIMAL(10,4),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(model_name, workflow_type)
);

-- Create indexes for performance
CREATE INDEX idx_execution_history_workflow ON execution_history(workflow_type);
CREATE INDEX idx_execution_history_status ON execution_history(status);
CREATE INDEX idx_execution_history_created ON execution_history(created_at DESC);
CREATE INDEX idx_execution_history_model ON execution_history(model_used);

CREATE INDEX idx_prompt_usage_prompt ON prompt_usage(prompt_id);
CREATE INDEX idx_prompt_usage_collection ON prompt_usage(prompt_collection);
CREATE INDEX idx_prompt_usage_effectiveness ON prompt_usage(effectiveness_score);

CREATE INDEX idx_workflow_metrics_name ON workflow_metrics(workflow_name);
CREATE INDEX idx_workflow_metrics_platform ON workflow_metrics(workflow_platform);
CREATE INDEX idx_workflow_metrics_success_rate ON workflow_metrics(success_rate);

CREATE INDEX idx_reasoning_patterns_type ON reasoning_patterns(pattern_type);
CREATE INDEX idx_reasoning_patterns_tags ON reasoning_patterns USING GIN(tags);
CREATE INDEX idx_reasoning_patterns_effectiveness ON reasoning_patterns(effectiveness_avg);

-- Update triggers for updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_execution_history_updated_at BEFORE UPDATE ON execution_history
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflow_metrics_updated_at BEFORE UPDATE ON workflow_metrics
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_reasoning_patterns_updated_at BEFORE UPDATE ON reasoning_patterns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflow_chains_updated_at BEFORE UPDATE ON workflow_chains
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_model_performance_updated_at BEFORE UPDATE ON model_performance
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to update workflow metrics after execution
CREATE OR REPLACE FUNCTION update_workflow_metrics_on_execution()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.status IN ('success', 'failed') THEN
        INSERT INTO workflow_metrics (
            workflow_name,
            workflow_platform,
            total_executions,
            successful_executions,
            failed_executions,
            avg_execution_time_ms,
            min_execution_time_ms,
            max_execution_time_ms,
            last_execution
        )
        VALUES (
            NEW.workflow_type,
            CASE 
                WHEN NEW.resource_type = 'n8n' THEN 'n8n'
                WHEN NEW.resource_type = 'windmill' THEN 'windmill'
                ELSE 'n8n'
            END,
            1,
            CASE WHEN NEW.status = 'success' THEN 1 ELSE 0 END,
            CASE WHEN NEW.status = 'failed' THEN 1 ELSE 0 END,
            NEW.execution_time_ms,
            NEW.execution_time_ms,
            NEW.execution_time_ms,
            NEW.created_at
        )
        ON CONFLICT (workflow_name) DO UPDATE SET
            total_executions = workflow_metrics.total_executions + 1,
            successful_executions = workflow_metrics.successful_executions + 
                CASE WHEN NEW.status = 'success' THEN 1 ELSE 0 END,
            failed_executions = workflow_metrics.failed_executions + 
                CASE WHEN NEW.status = 'failed' THEN 1 ELSE 0 END,
            avg_execution_time_ms = (
                (workflow_metrics.avg_execution_time_ms * workflow_metrics.total_executions + NEW.execution_time_ms) / 
                (workflow_metrics.total_executions + 1)
            ),
            min_execution_time_ms = LEAST(workflow_metrics.min_execution_time_ms, NEW.execution_time_ms),
            max_execution_time_ms = GREATEST(workflow_metrics.max_execution_time_ms, NEW.execution_time_ms),
            last_execution = NEW.created_at,
            updated_at = CURRENT_TIMESTAMP;
    END IF;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_metrics_on_execution_insert
    AFTER INSERT ON execution_history
    FOR EACH ROW
    EXECUTE FUNCTION update_workflow_metrics_on_execution();

CREATE TRIGGER update_metrics_on_execution_update
    AFTER UPDATE OF status ON execution_history
    FOR EACH ROW
    WHEN (OLD.status IS DISTINCT FROM NEW.status)
    EXECUTE FUNCTION update_workflow_metrics_on_execution();