-- SQLite schema for Browser Automation Studio
-- Simplified types for SQLite: UUID stored as TEXT, JSONB/TEXT[] stored as TEXT/JSON

CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT,
    folder_path TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS workflow_folders (
    id TEXT PRIMARY KEY,
    path TEXT NOT NULL UNIQUE,
    parent_id TEXT REFERENCES workflow_folders(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS workflows (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    folder_path TEXT NOT NULL,
    workflow_type TEXT NOT NULL DEFAULT 'flow',
    flow_definition TEXT DEFAULT '{}',
    inputs TEXT DEFAULT '{}',
    outputs TEXT DEFAULT '{}',
    expected_outcome TEXT DEFAULT '{}',
    workflow_metadata TEXT DEFAULT '{}',
    description TEXT,
    tags TEXT DEFAULT '[]',
    version INTEGER DEFAULT 1,
    is_template BOOLEAN DEFAULT FALSE,
    created_by TEXT,
    last_change_source TEXT DEFAULT 'manual',
    last_change_description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS workflow_versions (
    id TEXT PRIMARY KEY,
    workflow_id TEXT REFERENCES workflows(id) ON DELETE CASCADE,
    version INTEGER NOT NULL,
    flow_definition TEXT DEFAULT '{}',
    change_description TEXT,
    created_by TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(workflow_id, version)
);

CREATE TABLE IF NOT EXISTS executions (
    id TEXT PRIMARY KEY,
    workflow_id TEXT REFERENCES workflows(id) ON DELETE CASCADE,
    workflow_version INTEGER,
    status TEXT NOT NULL,
    trigger_type TEXT NOT NULL,
    trigger_metadata TEXT,
    parameters TEXT,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    last_heartbeat TIMESTAMP,
    error TEXT,
    result TEXT,
    progress INTEGER DEFAULT 0,
    current_step TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS execution_steps (
    id TEXT PRIMARY KEY,
    execution_id TEXT REFERENCES executions(id) ON DELETE CASCADE,
    step_index INTEGER NOT NULL,
    node_id TEXT,
    step_type TEXT,
    status TEXT,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    duration_ms INTEGER,
    error TEXT,
    input TEXT,
    output TEXT,
    metadata TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS screenshots (
    id TEXT PRIMARY KEY,
    execution_id TEXT NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_name TEXT NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    storage_url TEXT NOT NULL,
    thumbnail_url TEXT,
    width INTEGER,
    height INTEGER,
    size_bytes INTEGER,
    metadata TEXT DEFAULT '{}'
);

CREATE TABLE IF NOT EXISTS execution_logs (
    id TEXT PRIMARY KEY,
    execution_id TEXT REFERENCES executions(id) ON DELETE CASCADE,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    level TEXT,
    step_name TEXT,
    message TEXT,
    metadata TEXT
);

CREATE TABLE IF NOT EXISTS extracted_data (
    id TEXT PRIMARY KEY,
    execution_id TEXT REFERENCES executions(id) ON DELETE CASCADE,
    step_name TEXT,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    data_key TEXT,
    data_value TEXT,
    data_type TEXT,
    metadata TEXT
);

CREATE TABLE IF NOT EXISTS execution_artifacts (
    id TEXT PRIMARY KEY,
    execution_id TEXT REFERENCES executions(id) ON DELETE CASCADE,
    step_id TEXT REFERENCES execution_steps(id) ON DELETE SET NULL,
    step_index INTEGER,
    artifact_type TEXT,
    label TEXT,
    storage_url TEXT,
    thumbnail_url TEXT,
    content_type TEXT,
    size_bytes INTEGER,
    payload TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS exports (
    id TEXT PRIMARY KEY,
    execution_id TEXT REFERENCES executions(id) ON DELETE CASCADE,
    workflow_id TEXT REFERENCES workflows(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    format TEXT NOT NULL,
    settings TEXT,
    storage_url TEXT,
    thumbnail_url TEXT,
    file_size_bytes INTEGER,
    duration_ms INTEGER,
    frame_count INTEGER,
    ai_caption TEXT,
    ai_caption_generated_at TIMESTAMP,
    status TEXT NOT NULL,
    error TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS project_entries (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    path TEXT NOT NULL,
    kind TEXT NOT NULL,
    workflow_id TEXT REFERENCES workflows(id) ON DELETE SET NULL,
    metadata TEXT DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, path)
);

CREATE INDEX IF NOT EXISTS idx_workflows_project_id ON workflows(project_id);
CREATE INDEX IF NOT EXISTS idx_workflows_folder_path ON workflows(folder_path);
CREATE INDEX IF NOT EXISTS idx_workflow_versions_workflow_id ON workflow_versions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_project_entries_project_id ON project_entries(project_id);
CREATE INDEX IF NOT EXISTS idx_project_entries_kind ON project_entries(project_id, kind);
CREATE INDEX IF NOT EXISTS idx_project_entries_workflow_id ON project_entries(workflow_id);
CREATE INDEX IF NOT EXISTS idx_executions_workflow_id ON executions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_execution_logs_execution_id ON execution_logs(execution_id);
CREATE INDEX IF NOT EXISTS idx_screenshots_execution_id ON screenshots(execution_id);
CREATE INDEX IF NOT EXISTS idx_execution_steps_execution_id ON execution_steps(execution_id);
CREATE INDEX IF NOT EXISTS idx_extracted_data_execution_id ON extracted_data(execution_id);
CREATE INDEX IF NOT EXISTS idx_execution_artifacts_execution_id ON execution_artifacts(execution_id);
CREATE INDEX IF NOT EXISTS idx_exports_execution_id ON exports(execution_id);
CREATE INDEX IF NOT EXISTS idx_exports_workflow_id ON exports(workflow_id);
