-- Browser Automation Studio Database Schema
-- This schema defines the complete database structure for the browser automation platform.
-- It includes tables for projects, workflows, executions, artifacts, and related entities.

-- Create projects table
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    folder_path VARCHAR(500) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create workflow folders table
CREATE TABLE IF NOT EXISTS workflow_folders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    path VARCHAR(500) NOT NULL UNIQUE,
    parent_id UUID REFERENCES workflow_folders(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create workflows table
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    folder_path VARCHAR(500) NOT NULL,
    flow_definition JSONB DEFAULT '{}',
    description TEXT,
    tags TEXT[] DEFAULT '{}',
    version INTEGER DEFAULT 1,
    is_template BOOLEAN DEFAULT FALSE,
    created_by VARCHAR(255),
    last_change_source VARCHAR(255) DEFAULT 'manual',
    last_change_description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, folder_path)
);

-- Ensure newer workflow metadata columns exist for legacy databases
ALTER TABLE workflows
    ADD COLUMN IF NOT EXISTS last_change_source VARCHAR(255) DEFAULT 'manual';
ALTER TABLE workflows
    ALTER COLUMN last_change_source SET DEFAULT 'manual';
UPDATE workflows SET last_change_source = 'manual' WHERE last_change_source IS NULL;

ALTER TABLE workflows
    ADD COLUMN IF NOT EXISTS last_change_description TEXT;
ALTER TABLE workflows
    ALTER COLUMN last_change_description SET DEFAULT '';
UPDATE workflows SET last_change_description = '' WHERE last_change_description IS NULL;

ALTER TABLE workflows
    ADD COLUMN IF NOT EXISTS created_by VARCHAR(255);

ALTER TABLE workflows
    ADD COLUMN IF NOT EXISTS project_id UUID;

-- Create workflow versions table
CREATE TABLE IF NOT EXISTS workflow_versions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    flow_definition JSONB NOT NULL,
    change_description TEXT,
    created_by VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workflow_id, version)
);

-- Create executions table
CREATE TABLE IF NOT EXISTS executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    workflow_version INTEGER,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    trigger_type VARCHAR(50) NOT NULL DEFAULT 'manual',
    trigger_metadata JSONB DEFAULT '{}',
    parameters JSONB DEFAULT '{}',
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    last_heartbeat TIMESTAMP,
    error TEXT,
    result JSONB,
    progress INTEGER DEFAULT 0,
    current_step VARCHAR(255)
);

-- Create execution logs table
CREATE TABLE IF NOT EXISTS execution_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    level VARCHAR(20) NOT NULL DEFAULT 'info',
    step_name VARCHAR(255),
    message TEXT NOT NULL,
    metadata JSONB DEFAULT '{}'
);

-- Create screenshots table
CREATE TABLE IF NOT EXISTS screenshots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_name VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    storage_url VARCHAR(1000) NOT NULL,
    thumbnail_url VARCHAR(1000),
    width INTEGER,
    height INTEGER,
    size_bytes BIGINT,
    metadata JSONB DEFAULT '{}'
);

-- Create execution steps table
CREATE TABLE IF NOT EXISTS execution_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_index INTEGER NOT NULL,
    node_id VARCHAR(255) NOT NULL,
    step_type VARCHAR(50) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'running',
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    error TEXT,
    input JSONB DEFAULT '{}',
    output JSONB DEFAULT '{}',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_execution_step UNIQUE (execution_id, step_index),
    CONSTRAINT chk_execution_step_status CHECK (status IN ('pending','running','completed','failed'))
);

-- Create execution artifacts table
CREATE TABLE IF NOT EXISTS execution_artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_id UUID REFERENCES execution_steps(id) ON DELETE CASCADE,
    step_index INTEGER,
    artifact_type VARCHAR(100) NOT NULL,
    label VARCHAR(255),
    storage_url VARCHAR(1000),
    thumbnail_url VARCHAR(1000),
    content_type VARCHAR(100),
    size_bytes BIGINT,
    payload JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create extracted data table
CREATE TABLE IF NOT EXISTS extracted_data (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_name VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    data_key VARCHAR(255) NOT NULL,
    data_value JSONB NOT NULL,
    data_type VARCHAR(50),
    metadata JSONB DEFAULT '{}'
);

-- Create workflow schedules table
CREATE TABLE IF NOT EXISTS workflow_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    cron_expression VARCHAR(100) NOT NULL,
    timezone VARCHAR(50) DEFAULT 'UTC',
    is_active BOOLEAN DEFAULT TRUE,
    parameters JSONB DEFAULT '{}',
    next_run_at TIMESTAMP,
    last_run_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create workflow templates table
CREATE TABLE IF NOT EXISTS workflow_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    category VARCHAR(100) NOT NULL,
    description TEXT,
    flow_definition JSONB NOT NULL,
    icon VARCHAR(100),
    example_parameters JSONB DEFAULT '{}',
    tags TEXT[] DEFAULT '{}',
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create AI generations table
CREATE TABLE IF NOT EXISTS ai_generations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID REFERENCES workflows(id) ON DELETE SET NULL,
    prompt TEXT NOT NULL,
    generated_flow JSONB NOT NULL,
    model VARCHAR(100),
    generation_time_ms INTEGER,
    success BOOLEAN DEFAULT FALSE,
    error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX IF NOT EXISTS idx_workflows_project_id ON workflows(project_id);
CREATE INDEX IF NOT EXISTS idx_workflows_folder_path ON workflows(folder_path);
CREATE INDEX IF NOT EXISTS idx_executions_workflow_id ON executions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);
CREATE INDEX IF NOT EXISTS idx_executions_last_heartbeat ON executions(last_heartbeat);
CREATE INDEX IF NOT EXISTS idx_execution_logs_execution_id ON execution_logs(execution_id);
CREATE INDEX IF NOT EXISTS idx_screenshots_execution_id ON screenshots(execution_id);
CREATE INDEX IF NOT EXISTS idx_execution_steps_execution ON execution_steps(execution_id);
CREATE INDEX IF NOT EXISTS idx_execution_artifacts_execution ON execution_artifacts(execution_id);
CREATE INDEX IF NOT EXISTS idx_execution_artifacts_step ON execution_artifacts(step_id);
CREATE INDEX IF NOT EXISTS idx_extracted_data_execution_id ON extracted_data(execution_id);
CREATE INDEX IF NOT EXISTS idx_workflow_schedules_active ON workflow_schedules(is_active);
CREATE INDEX IF NOT EXISTS idx_ai_generations_workflow_id ON ai_generations(workflow_id);

-- Create updated_at trigger function
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at
DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
CREATE TRIGGER update_projects_updated_at BEFORE UPDATE ON projects FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_workflow_folders_updated_at ON workflow_folders;
CREATE TRIGGER update_workflow_folders_updated_at BEFORE UPDATE ON workflow_folders FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_workflows_updated_at ON workflows;
CREATE TRIGGER update_workflows_updated_at BEFORE UPDATE ON workflows FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_workflow_schedules_updated_at ON workflow_schedules;
CREATE TRIGGER update_workflow_schedules_updated_at BEFORE UPDATE ON workflow_schedules FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_workflow_templates_updated_at ON workflow_templates;
CREATE TRIGGER update_workflow_templates_updated_at BEFORE UPDATE ON workflow_templates FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_execution_steps_updated_at ON execution_steps;
CREATE TRIGGER update_execution_steps_updated_at BEFORE UPDATE ON execution_steps FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_execution_artifacts_updated_at ON execution_artifacts;
CREATE TRIGGER update_execution_artifacts_updated_at BEFORE UPDATE ON execution_artifacts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
