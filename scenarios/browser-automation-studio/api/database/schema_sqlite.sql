-- Browser Automation Studio - Simplified Schema (SQLite)
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
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    folder_path TEXT NOT NULL UNIQUE,
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
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    folder_path TEXT NOT NULL,
    file_path TEXT,  -- Relative path to JSON file on disk
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
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending',  -- pending|running|completed|failed
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,  -- Brief error summary for display
    result_path TEXT,  -- Path to detailed results JSON on disk
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
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    cron_expression TEXT NOT NULL,
    timezone TEXT DEFAULT 'UTC',
    is_active INTEGER DEFAULT 1,  -- SQLite uses INTEGER for boolean
    parameters_json TEXT DEFAULT '{}',  -- JSON string, not queried
    next_run_at TIMESTAMP,
    last_run_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_schedules_workflow_id ON schedules(workflow_id);
CREATE INDEX IF NOT EXISTS idx_schedules_active ON schedules(is_active);
CREATE INDEX IF NOT EXISTS idx_schedules_next_run ON schedules(next_run_at);

-- ============================================================================
-- EXPORTS: Metadata for exported artifacts (replays, videos, etc.)
-- ============================================================================
CREATE TABLE IF NOT EXISTS exports (
    id TEXT PRIMARY KEY,
    execution_id TEXT NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    workflow_id TEXT REFERENCES workflows(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    format TEXT NOT NULL,
    settings TEXT DEFAULT '{}',
    storage_url TEXT,
    thumbnail_url TEXT,
    file_size_bytes INTEGER,
    duration_ms INTEGER,
    frame_count INTEGER,
    ai_caption TEXT,
    ai_caption_generated_at TIMESTAMP,
    status TEXT NOT NULL DEFAULT 'pending',
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
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- ============================================================================
-- TRIGGERS: Auto-update updated_at timestamps
-- ============================================================================
CREATE TRIGGER IF NOT EXISTS update_projects_updated_at
    AFTER UPDATE ON projects
    FOR EACH ROW
    WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE projects SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS update_workflows_updated_at
    AFTER UPDATE ON workflows
    FOR EACH ROW
    WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE workflows SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS update_executions_updated_at
    AFTER UPDATE ON executions
    FOR EACH ROW
    WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE executions SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS update_schedules_updated_at
    AFTER UPDATE ON schedules
    FOR EACH ROW
    WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE schedules SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS update_exports_updated_at
    AFTER UPDATE ON exports
    FOR EACH ROW
    WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE exports SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;
