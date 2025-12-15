-- Browser Automation Studio - Simplified Schema
--
-- Design principle: Database is an INDEX, not the source of truth.
-- - Workflows live on disk as JSON files
-- - Execution results live on disk as JSON files
-- - Database provides queryable indexes for:
--   1. Active/recent executions (status filtering)
--   2. Scheduled runs (next_run_at queries)
--   3. Project/workflow lookups (by name/path)
--   4. User settings (key-value store)

-- ============================================================================
-- PROJECTS: Top-level containers for workflows
-- ============================================================================
CREATE TABLE IF NOT EXISTS projects (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    folder_path VARCHAR(500) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_projects_name ON projects(name);
CREATE INDEX IF NOT EXISTS idx_projects_folder_path ON projects(folder_path);

-- ============================================================================
-- WORKFLOWS: Index of workflow files on disk
-- ============================================================================
-- Note: flow_definition, inputs, outputs, etc. are NOT stored here.
-- They live in JSON files on disk. This table is just for lookups.
CREATE TABLE IF NOT EXISTS workflows (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id UUID REFERENCES projects(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    folder_path VARCHAR(500) NOT NULL,
    file_path VARCHAR(1000),  -- Relative path to JSON file on disk
    version INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(name, folder_path)
);

CREATE INDEX IF NOT EXISTS idx_workflows_project_id ON workflows(project_id);
CREATE INDEX IF NOT EXISTS idx_workflows_folder_path ON workflows(folder_path);
CREATE INDEX IF NOT EXISTS idx_workflows_name ON workflows(name);

-- ============================================================================
-- EXECUTIONS: Track workflow runs (queryable for status/recent)
-- ============================================================================
-- Note: Detailed step data, logs, and artifacts live in JSON files on disk.
-- This table only stores what we need to query.
CREATE TABLE IF NOT EXISTS executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',  -- pending|running|completed|failed
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,  -- Brief error summary for display
    result_path VARCHAR(1000),  -- Path to detailed results JSON on disk
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_executions_workflow_id ON executions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);
CREATE INDEX IF NOT EXISTS idx_executions_started_at ON executions(started_at DESC);

-- ============================================================================
-- SCHEDULES: Cron-based workflow scheduling
-- ============================================================================
-- This MUST be in the database for efficient next-run queries.
CREATE TABLE IF NOT EXISTS schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    cron_expression VARCHAR(100) NOT NULL,
    timezone VARCHAR(50) DEFAULT 'UTC',
    is_active BOOLEAN DEFAULT TRUE,
    parameters_json TEXT DEFAULT '{}',  -- JSON string, not queried
    next_run_at TIMESTAMP,
    last_run_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_schedules_workflow_id ON schedules(workflow_id);
CREATE INDEX IF NOT EXISTS idx_schedules_active ON schedules(is_active);
CREATE INDEX IF NOT EXISTS idx_schedules_next_run ON schedules(next_run_at) WHERE is_active = TRUE;

-- ============================================================================
-- EXPORTS: Metadata for exported artifacts (replays, videos, etc.)
-- ============================================================================
CREATE TABLE IF NOT EXISTS exports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    execution_id UUID NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    workflow_id UUID REFERENCES workflows(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    format VARCHAR(50) NOT NULL,
    settings JSONB DEFAULT '{}'::jsonb,
    storage_url TEXT,
    thumbnail_url TEXT,
    file_size_bytes BIGINT,
    duration_ms INTEGER,
    frame_count INTEGER,
    ai_caption TEXT,
    ai_caption_generated_at TIMESTAMP,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_exports_execution_id ON exports(execution_id);
CREATE INDEX IF NOT EXISTS idx_exports_workflow_id ON exports(workflow_id);
CREATE INDEX IF NOT EXISTS idx_exports_created_at ON exports(created_at DESC);

-- ============================================================================
-- SETTINGS: Key-value store for user preferences
-- ============================================================================
CREATE TABLE IF NOT EXISTS settings (
    key VARCHAR(255) PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- TRIGGERS: Auto-update timestamps
-- ============================================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_projects_updated_at ON projects;
CREATE TRIGGER update_projects_updated_at
    BEFORE UPDATE ON projects
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_workflows_updated_at ON workflows;
CREATE TRIGGER update_workflows_updated_at
    BEFORE UPDATE ON workflows
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_executions_updated_at ON executions;
CREATE TRIGGER update_executions_updated_at
    BEFORE UPDATE ON executions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_schedules_updated_at ON schedules;
CREATE TRIGGER update_schedules_updated_at
    BEFORE UPDATE ON schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_exports_updated_at ON exports;
CREATE TRIGGER update_exports_updated_at
    BEFORE UPDATE ON exports
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_settings_updated_at ON settings;
CREATE TRIGGER update_settings_updated_at
    BEFORE UPDATE ON settings
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
