CREATE TABLE IF NOT EXISTS projects (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    folder_path TEXT NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_projects_name ON projects(name);
CREATE INDEX IF NOT EXISTS idx_projects_folder_path ON projects(folder_path);

CREATE TABLE IF NOT EXISTS workflows (
    id TEXT PRIMARY KEY,
    project_id TEXT REFERENCES projects(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    folder_path TEXT NOT NULL,
    file_path TEXT,
    version INTEGER DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, name, folder_path)
);

CREATE INDEX IF NOT EXISTS idx_workflows_project_id ON workflows(project_id);
CREATE INDEX IF NOT EXISTS idx_workflows_folder_path ON workflows(folder_path);
CREATE INDEX IF NOT EXISTS idx_workflows_name ON workflows(name);

CREATE TABLE IF NOT EXISTS project_assets (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
    file_path TEXT NOT NULL,
    file_name TEXT NOT NULL DEFAULT '',
    file_size INTEGER NOT NULL DEFAULT 0,
    mime_type TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(project_id, file_path)
);

CREATE INDEX IF NOT EXISTS idx_project_assets_project_id ON project_assets(project_id);
CREATE INDEX IF NOT EXISTS idx_project_assets_file_path ON project_assets(project_id, file_path);

CREATE TABLE IF NOT EXISTS executions (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'pending',
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    error_message TEXT,
    result_path TEXT,
    resumed_from_id TEXT REFERENCES executions(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_executions_workflow_id ON executions(workflow_id);
CREATE INDEX IF NOT EXISTS idx_executions_status ON executions(status);
CREATE INDEX IF NOT EXISTS idx_executions_started_at ON executions(started_at DESC);
-- Foreign-key checks during retention must not scan every execution per deletion.
CREATE INDEX IF NOT EXISTS idx_executions_resumed_from_id ON executions(resumed_from_id);

-- Durable idempotency claims for CaptureService.Capture. The token is owned
-- by the caller; request_hash prevents accidental reuse for different input.
-- execution_id is reserved before browser work begins so a lost response can
-- be reconciled against the normal execution owner.
CREATE TABLE IF NOT EXISTS capture_requests (
    request_token TEXT PRIMARY KEY,
    request_hash TEXT NOT NULL,
    request_json TEXT NOT NULL,
    execution_id TEXT NOT NULL UNIQUE,
    state TEXT NOT NULL CHECK (state IN ('in_progress', 'completed', 'failed')),
    response_json TEXT NOT NULL DEFAULT '',
    error TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Admission claim prevents different API processes from resuming the same checkpoint concurrently.
-- Claims advance to a new child only after the previous child reached a terminal state.
CREATE TABLE IF NOT EXISTS execution_resume_claims (
    source_execution_id TEXT PRIMARY KEY REFERENCES executions(id) ON DELETE CASCADE,
    child_execution_id TEXT NOT NULL UNIQUE REFERENCES executions(id) ON DELETE CASCADE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Upgrade existing databases: prefer an active child so deploying this admission guard
-- cannot accidentally permit a duplicate while a pre-upgrade resume is still running.
INSERT OR IGNORE INTO execution_resume_claims (source_execution_id, child_execution_id)
SELECT resumed_from_id, id
FROM executions
WHERE resumed_from_id IS NOT NULL
ORDER BY CASE WHEN status IN ('pending', 'running') THEN 0 ELSE 1 END, started_at DESC;

CREATE TABLE IF NOT EXISTS schedules (
    id TEXT PRIMARY KEY,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    cron_expression TEXT NOT NULL,
    timezone TEXT DEFAULT 'UTC',
    is_active INTEGER DEFAULT 1,
    parameters_json TEXT DEFAULT '{}',
    next_run_at TIMESTAMP,
    last_run_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_schedules_workflow_id ON schedules(workflow_id);
CREATE INDEX IF NOT EXISTS idx_schedules_active ON schedules(is_active);
CREATE INDEX IF NOT EXISTS idx_schedules_next_run ON schedules(next_run_at);

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

CREATE TABLE IF NOT EXISTS settings (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);
