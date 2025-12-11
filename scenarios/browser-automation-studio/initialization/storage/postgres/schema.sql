-- Vrooli Ascension Database Schema
-- PostgreSQL schema for workflow and execution storage

-- Create database if not exists
-- Note: This must be run as superuser or database owner
-- CREATE DATABASE browser_automation_studio;

-- Use the database
-- \c browser_automation_studio;

-- Workflow folders table
CREATE TABLE IF NOT EXISTS workflow_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    path VARCHAR(500) NOT NULL UNIQUE,
    parent_id UUID REFERENCES workflow_folders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create index for folder path lookups
CREATE INDEX idx_workflow_folders_path ON workflow_folders(path);
CREATE INDEX idx_workflow_folders_parent ON workflow_folders(parent_id);

-- Workflows table
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    folder_path VARCHAR(500) DEFAULT '/',
    flow_definition JSONB NOT NULL DEFAULT '{"nodes": [], "edges": []}',
    description TEXT,
    tags TEXT[] DEFAULT '{}',
    version INTEGER DEFAULT 1,
    is_template BOOLEAN DEFAULT FALSE,
    created_by VARCHAR(255),
    last_change_source VARCHAR(255) DEFAULT 'manual',
    last_change_description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_workflow_name_in_folder UNIQUE (name, folder_path)
);

-- Create indexes for workflows
CREATE INDEX idx_workflows_folder_path ON workflows(folder_path);
CREATE INDEX idx_workflows_tags ON workflows USING GIN(tags);
CREATE INDEX idx_workflows_created_at ON workflows(created_at);
CREATE INDEX idx_workflows_is_template ON workflows(is_template);

-- Workflow versions table (for version history)
CREATE TABLE IF NOT EXISTS workflow_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    flow_definition JSONB NOT NULL,
    change_description TEXT,
    created_by VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_workflow_version UNIQUE (workflow_id, version)
);

-- Create index for version lookups
CREATE INDEX idx_workflow_versions_workflow ON workflow_versions(workflow_id);

-- Executions table
CREATE TABLE IF NOT EXISTS executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    workflow_version INTEGER,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    trigger_type VARCHAR(50) DEFAULT 'manual',
    trigger_metadata JSONB,
    parameters JSONB,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    last_heartbeat TIMESTAMP WITH TIME ZONE,
    error TEXT,
    result JSONB,
    progress INTEGER DEFAULT 0,
    current_step VARCHAR(255),
    CONSTRAINT chk_status CHECK (status IN ('pending', 'running', 'completed', 'failed', 'cancelled'))
);

-- Create indexes for executions
CREATE INDEX idx_executions_workflow ON executions(workflow_id);
CREATE INDEX idx_executions_status ON executions(status);
CREATE INDEX idx_executions_started_at ON executions(started_at);
CREATE INDEX idx_executions_trigger_type ON executions(trigger_type);
CREATE INDEX idx_executions_last_heartbeat ON executions(last_heartbeat);

-- Index for stale execution recovery (progress continuity)
-- This enables efficient detection of executions left in running/pending state
-- after service restarts or crashes
CREATE INDEX idx_executions_stale_recovery ON executions(status, started_at, last_heartbeat)
    WHERE status IN ('running', 'pending');

-- Execution logs table
CREATE TABLE IF NOT EXISTS execution_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    level VARCHAR(20) NOT NULL DEFAULT 'info',
    step_name VARCHAR(255),
    message TEXT NOT NULL,
    metadata JSONB,
    CONSTRAINT chk_log_level CHECK (level IN ('debug', 'info', 'warning', 'error', 'success'))
);

-- Create indexes for logs
CREATE INDEX idx_execution_logs_execution ON execution_logs(execution_id);
CREATE INDEX idx_execution_logs_timestamp ON execution_logs(timestamp);
CREATE INDEX idx_execution_logs_level ON execution_logs(level);

-- Screenshots table
CREATE TABLE IF NOT EXISTS screenshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_name VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    storage_url TEXT NOT NULL,
    thumbnail_url TEXT,
    width INTEGER,
    height INTEGER,
    size_bytes BIGINT,
    metadata JSONB
);

-- Execution steps table
CREATE TABLE IF NOT EXISTS execution_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_index INTEGER NOT NULL,
    node_id VARCHAR(255) NOT NULL,
    step_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'running',
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    error TEXT,
    input JSONB DEFAULT '{}',
    output JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_execution_step UNIQUE (execution_id, step_index),
    CONSTRAINT chk_execution_step_status CHECK (status IN ('pending','running','completed','failed'))
);

-- Execution artifacts table
CREATE TABLE IF NOT EXISTS execution_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_id UUID REFERENCES execution_steps(id) ON DELETE CASCADE,
    step_index INTEGER,
    artifact_type VARCHAR(100) NOT NULL,
    label VARCHAR(255),
    storage_url TEXT,
    thumbnail_url TEXT,
    content_type VARCHAR(100),
    size_bytes BIGINT,
    payload JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for screenshots
CREATE INDEX idx_screenshots_execution ON screenshots(execution_id);
CREATE INDEX idx_screenshots_timestamp ON screenshots(timestamp);

-- Create indexes for execution steps & artifacts
CREATE INDEX idx_execution_steps_execution ON execution_steps(execution_id);
CREATE INDEX idx_execution_artifacts_execution ON execution_artifacts(execution_id);
CREATE INDEX idx_execution_artifacts_step ON execution_artifacts(step_id);

-- Extracted data table
CREATE TABLE IF NOT EXISTS extracted_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_name VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    data_key VARCHAR(255) NOT NULL,
    data_value JSONB NOT NULL,
    data_type VARCHAR(50),
    metadata JSONB
);

-- Create indexes for extracted data
CREATE INDEX idx_extracted_data_execution ON extracted_data(execution_id);
CREATE INDEX idx_extracted_data_key ON extracted_data(data_key);
CREATE INDEX idx_extracted_data_timestamp ON extracted_data(timestamp);

-- Workflow schedules table
CREATE TABLE IF NOT EXISTS workflow_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    cron_expression VARCHAR(255),
    timezone VARCHAR(100) DEFAULT 'UTC',
    is_active BOOLEAN DEFAULT TRUE,
    parameters JSONB,
    next_run_at TIMESTAMP WITH TIME ZONE,
    last_run_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for schedules
CREATE INDEX idx_workflow_schedules_workflow ON workflow_schedules(workflow_id);
CREATE INDEX idx_workflow_schedules_active ON workflow_schedules(is_active);
CREATE INDEX idx_workflow_schedules_next_run ON workflow_schedules(next_run_at);

-- Workflow templates table
CREATE TABLE IF NOT EXISTS workflow_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    category VARCHAR(100) NOT NULL,
    description TEXT,
    flow_definition JSONB NOT NULL,
    icon VARCHAR(50),
    example_parameters JSONB,
    tags TEXT[] DEFAULT '{}',
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for templates
CREATE INDEX idx_workflow_templates_category ON workflow_templates(category);
CREATE INDEX idx_workflow_templates_tags ON workflow_templates USING GIN(tags);
CREATE INDEX idx_workflow_templates_usage ON workflow_templates(usage_count DESC);

-- AI generation history table
CREATE TABLE IF NOT EXISTS ai_generations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID REFERENCES workflows(id) ON DELETE SET NULL,
    prompt TEXT NOT NULL,
    generated_flow JSONB NOT NULL,
    model VARCHAR(100),
    generation_time_ms INTEGER,
    success BOOLEAN DEFAULT TRUE,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for AI generations
CREATE INDEX idx_ai_generations_workflow ON ai_generations(workflow_id);
CREATE INDEX idx_ai_generations_created ON ai_generations(created_at);

-- Update trigger for updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
CREATE TRIGGER update_workflow_folders_updated_at BEFORE UPDATE ON workflow_folders
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflows_updated_at BEFORE UPDATE ON workflows
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflow_schedules_updated_at BEFORE UPDATE ON workflow_schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_workflow_templates_updated_at BEFORE UPDATE ON workflow_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_execution_steps_updated_at BEFORE UPDATE ON execution_steps
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_execution_artifacts_updated_at BEFORE UPDATE ON execution_artifacts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Insert default folders
INSERT INTO workflow_folders (path, name, description) VALUES
    ('/ui-validation', 'UI Validation', 'Workflows for testing user interfaces'),
    ('/data-collection', 'Data Collection', 'Web scraping and data extraction workflows'),
    ('/automation', 'Automation', 'Business process automation workflows'),
    ('/monitoring', 'Monitoring', 'Website and application monitoring workflows')
ON CONFLICT (path) DO NOTHING;

-- Entitlement usage tracking table
-- Tracks workflow execution counts per user per billing month for tier limit enforcement
CREATE TABLE IF NOT EXISTS entitlement_usage (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_identity VARCHAR(255) NOT NULL,
    billing_month VARCHAR(7) NOT NULL,  -- Format: YYYY-MM
    execution_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_user_month UNIQUE (user_identity, billing_month)
);

-- Create indexes for entitlement usage lookups
CREATE INDEX idx_entitlement_usage_user ON entitlement_usage(user_identity);
CREATE INDEX idx_entitlement_usage_month ON entitlement_usage(billing_month);

-- User identity storage table
-- Stores the user's email for entitlement verification
-- This is persisted locally so the user doesn't have to re-enter it
CREATE TABLE IF NOT EXISTS user_settings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key VARCHAR(255) NOT NULL UNIQUE,
    value TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- UX Metrics Schema Extension
-- Raw interaction traces (write-heavy, append-only)
CREATE TABLE IF NOT EXISTS ux_interaction_traces (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_index INTEGER NOT NULL,
    action_type VARCHAR(50) NOT NULL,
    element_id VARCHAR(255),
    selector TEXT,
    position_x DOUBLE PRECISION,
    position_y DOUBLE PRECISION,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL,
    duration_ms BIGINT,
    success BOOLEAN NOT NULL DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ux_traces_execution ON ux_interaction_traces(execution_id);
CREATE INDEX IF NOT EXISTS idx_ux_traces_step ON ux_interaction_traces(execution_id, step_index);

-- Cursor paths (one per step, stores full trajectory)
CREATE TABLE IF NOT EXISTS ux_cursor_paths (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_index INTEGER NOT NULL,
    points JSONB NOT NULL,  -- Array of {x, y, timestamp}
    total_distance_px DOUBLE PRECISION,
    direct_distance_px DOUBLE PRECISION,
    duration_ms BIGINT,
    directness DOUBLE PRECISION,
    zigzag_score DOUBLE PRECISION,
    average_speed DOUBLE PRECISION,
    max_speed DOUBLE PRECISION,
    hesitation_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT ux_cursor_paths_execution_step_unique UNIQUE(execution_id, step_index)
);

CREATE INDEX IF NOT EXISTS idx_ux_cursor_execution ON ux_cursor_paths(execution_id);

-- Computed execution metrics (materialized for fast queries)
CREATE TABLE IF NOT EXISTS ux_execution_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    computed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    total_duration_ms BIGINT,
    step_count INTEGER,
    successful_steps INTEGER,
    failed_steps INTEGER,
    total_retries INTEGER,
    avg_step_duration_ms DOUBLE PRECISION,
    total_cursor_distance DOUBLE PRECISION,
    overall_friction_score DOUBLE PRECISION,
    friction_signals JSONB DEFAULT '[]',
    step_metrics JSONB DEFAULT '[]',
    summary JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT ux_execution_metrics_execution_unique UNIQUE(execution_id)
);

CREATE INDEX IF NOT EXISTS idx_ux_metrics_workflow ON ux_execution_metrics(workflow_id);
CREATE INDEX IF NOT EXISTS idx_ux_metrics_friction ON ux_execution_metrics(overall_friction_score);
CREATE INDEX IF NOT EXISTS idx_ux_metrics_computed_at ON ux_execution_metrics(computed_at DESC);

-- Insert sample workflow templates
INSERT INTO workflow_templates (name, category, description, flow_definition, icon, tags) VALUES
    ('Basic Navigation', 'getting-started', 'Simple navigation and screenshot workflow', 
     '{"nodes": [{"id": "1", "type": "navigate"}, {"id": "2", "type": "screenshot"}], "edges": [{"source": "1", "target": "2"}]}',
     'globe', ARRAY['basic', 'navigation', 'screenshot']),
    
    ('Form Submission', 'automation', 'Fill and submit a web form',
     '{"nodes": [{"id": "1", "type": "navigate"}, {"id": "2", "type": "type"}, {"id": "3", "type": "click"}], "edges": [{"source": "1", "target": "2"}, {"source": "2", "target": "3"}]}',
     'form', ARRAY['form', 'input', 'automation']),
    
    ('Data Extraction', 'data-collection', 'Extract structured data from a webpage',
     '{"nodes": [{"id": "1", "type": "navigate"}, {"id": "2", "type": "wait"}, {"id": "3", "type": "extract"}], "edges": [{"source": "1", "target": "2"}, {"source": "2", "target": "3"}]}',
     'database', ARRAY['scraping', 'data', 'extraction'])
ON CONFLICT (name) DO NOTHING;

-- Grant permissions (adjust as needed for your setup)
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO your_app_user;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO your_app_user;
