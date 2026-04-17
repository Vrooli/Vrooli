-- Browser Automation Studio - Database Schema
--
-- Design principle: Database is an INDEX, not the source of truth.
-- - Workflows live on disk as JSON files
-- - Execution results live on disk as JSON files
-- - Database provides queryable indexes for:
--   1. Active/recent executions (status filtering)
--   2. Scheduled runs (next_run_at queries)
--   3. Project/workflow lookups (by name/path)
--   4. User settings (key-value store)
--   5. Credit usage tracking
--   6. UX metrics (interaction traces, cursor paths, execution-level scores)

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
    UNIQUE(project_id, name, folder_path)
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
    resumed_from_id TEXT REFERENCES executions(id),  -- Links to parent execution if this is a resume
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
    settings TEXT DEFAULT '{}',  -- JSON as TEXT (SQLite has no JSONB)
    storage_url TEXT,
    thumbnail_url TEXT,
    file_size_bytes INTEGER,  -- SQLite uses 64-bit integers natively
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
-- RECORDING SESSIONS: Browser recording sessions
-- ============================================================================
-- Persists recording sessions with their associated actions.
-- Sessions may optionally link to a SessionProfile for state restoration.
-- DOC: docs/architecture/recording.md#recording-session
CREATE TABLE IF NOT EXISTS recording_sessions (
    id TEXT PRIMARY KEY,
    profile_id TEXT,  -- Optional link to session profile
    status TEXT NOT NULL DEFAULT 'active',  -- active | closed
    viewport_width INTEGER,
    viewport_height INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    closed_at TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_recording_sessions_profile ON recording_sessions(profile_id);
CREATE INDEX IF NOT EXISTS idx_recording_sessions_status ON recording_sessions(status);
CREATE INDEX IF NOT EXISTS idx_recording_sessions_created_at ON recording_sessions(created_at DESC);

-- ============================================================================
-- RECORDING ACTIONS: User actions captured during recording
-- ============================================================================
-- Each action belongs to a session and records user interactions.
-- Complex fields (selector, element_meta, bounding_box, payload) are JSON.
-- DOC: docs/architecture/recording.md#recording-action
CREATE TABLE IF NOT EXISTS recording_actions (
    id TEXT PRIMARY KEY,
    session_id TEXT NOT NULL REFERENCES recording_sessions(id) ON DELETE CASCADE,
    page_id TEXT NOT NULL,
    sequence_num INTEGER NOT NULL,
    action_type TEXT NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    duration_ms INTEGER,
    selector TEXT,      -- JSON: SelectorSet with primary and candidates
    element_meta TEXT,  -- JSON: ElementMeta with tag, class, aria info
    bounding_box TEXT,  -- JSON: {x, y, width, height}
    payload TEXT,       -- JSON: action-specific data
    url TEXT,
    page_title TEXT,
    confidence REAL DEFAULT 1.0,
    source TEXT DEFAULT 'auto',  -- auto | manual | ai_suggested
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(session_id, sequence_num)
);

CREATE INDEX IF NOT EXISTS idx_recording_actions_session ON recording_actions(session_id);
CREATE INDEX IF NOT EXISTS idx_recording_actions_page ON recording_actions(page_id);
CREATE INDEX IF NOT EXISTS idx_recording_actions_type ON recording_actions(action_type);
CREATE INDEX IF NOT EXISTS idx_recording_actions_timestamp ON recording_actions(timestamp);

-- ============================================================================
-- UNIFIED CREDIT SYSTEM TABLES
-- Single credit pool model for all operations (AI, executions, exports).
-- ============================================================================

-- Unified Credit Usage Tracking
-- Tracks credit usage per user per billing month with operation type breakdown
CREATE TABLE IF NOT EXISTS credit_usage (
    id TEXT PRIMARY KEY,
    user_identity TEXT NOT NULL,
    billing_month TEXT NOT NULL,  -- Format: YYYY-MM

    -- Single unified credit pool totals
    total_credits_used INTEGER NOT NULL DEFAULT 0,
    total_operations INTEGER NOT NULL DEFAULT 0,

    -- Breakdown by operation type for analytics/UI display
    -- Format: {"ai.workflow_generate": 15, "execution.run": 42}
    credits_by_operation TEXT NOT NULL DEFAULT '{}',  -- JSON as TEXT
    operations_by_type TEXT NOT NULL DEFAULT '{}',  -- JSON as TEXT

    last_operation_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

    UNIQUE (user_identity, billing_month)
);

CREATE INDEX IF NOT EXISTS idx_credit_usage_user ON credit_usage(user_identity);
CREATE INDEX IF NOT EXISTS idx_credit_usage_month ON credit_usage(billing_month);

-- Unified Operation Log
-- Detailed audit log for all credit-consuming operations
CREATE TABLE IF NOT EXISTS operation_log (
    id TEXT PRIMARY KEY,
    user_identity TEXT NOT NULL,
    operation_type TEXT NOT NULL,  -- e.g., 'ai.workflow_generate', 'execution.run'
    credits_charged INTEGER NOT NULL DEFAULT 0,
    success INTEGER NOT NULL DEFAULT 1,  -- SQLite uses INTEGER for boolean

    -- Flexible metadata for operation-specific details
    metadata TEXT DEFAULT '{}',  -- JSON as TEXT

    error_message TEXT,
    duration_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_operation_log_user ON operation_log(user_identity);
CREATE INDEX IF NOT EXISTS idx_operation_log_type ON operation_log(operation_type);
CREATE INDEX IF NOT EXISTS idx_operation_log_created ON operation_log(created_at);

-- ============================================================================
-- UX METRICS: Per-step interaction traces, cursor paths, and execution-level scores
-- Backs services/uxmetrics. Repository code passes its own id only for traces;
-- cursor_paths and execution_metrics rely on the DEFAULT random hex id.
-- ============================================================================

CREATE TABLE IF NOT EXISTS ux_interaction_traces (
    id TEXT PRIMARY KEY,
    execution_id TEXT NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_index INTEGER NOT NULL,
    action_type TEXT NOT NULL,
    element_id TEXT,
    selector TEXT,
    position_x REAL,
    position_y REAL,
    timestamp TIMESTAMP NOT NULL,
    duration_ms INTEGER,
    success INTEGER NOT NULL DEFAULT 1,
    metadata TEXT DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_ux_traces_execution ON ux_interaction_traces(execution_id);
CREATE INDEX IF NOT EXISTS idx_ux_traces_step ON ux_interaction_traces(execution_id, step_index);

CREATE TABLE IF NOT EXISTS ux_cursor_paths (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    execution_id TEXT NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    step_index INTEGER NOT NULL,
    points TEXT NOT NULL,
    total_distance_px REAL,
    direct_distance_px REAL,
    duration_ms INTEGER,
    directness REAL,
    zigzag_score REAL,
    average_speed REAL,
    max_speed REAL,
    hesitation_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(execution_id, step_index)
);

CREATE INDEX IF NOT EXISTS idx_ux_cursor_execution ON ux_cursor_paths(execution_id);

CREATE TABLE IF NOT EXISTS ux_execution_metrics (
    id TEXT PRIMARY KEY DEFAULT (lower(hex(randomblob(16)))),
    execution_id TEXT NOT NULL REFERENCES executions(id) ON DELETE CASCADE,
    workflow_id TEXT NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    computed_at TIMESTAMP NOT NULL,
    total_duration_ms INTEGER,
    step_count INTEGER,
    successful_steps INTEGER,
    failed_steps INTEGER,
    total_retries INTEGER,
    avg_step_duration_ms REAL,
    total_cursor_distance REAL,
    overall_friction_score REAL,
    friction_signals TEXT DEFAULT '[]',
    step_metrics TEXT DEFAULT '[]',
    summary TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(execution_id)
);

CREATE INDEX IF NOT EXISTS idx_ux_metrics_workflow ON ux_execution_metrics(workflow_id);
CREATE INDEX IF NOT EXISTS idx_ux_metrics_friction ON ux_execution_metrics(overall_friction_score);

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

CREATE TRIGGER IF NOT EXISTS update_settings_updated_at
    AFTER UPDATE ON settings
    FOR EACH ROW
    WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE settings SET updated_at = CURRENT_TIMESTAMP WHERE key = OLD.key;
END;

CREATE TRIGGER IF NOT EXISTS update_credit_usage_updated_at
    AFTER UPDATE ON credit_usage
    FOR EACH ROW
    WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE credit_usage SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;

CREATE TRIGGER IF NOT EXISTS update_recording_sessions_updated_at
    AFTER UPDATE ON recording_sessions
    FOR EACH ROW
    WHEN NEW.updated_at = OLD.updated_at
BEGIN
    UPDATE recording_sessions SET updated_at = CURRENT_TIMESTAMP WHERE id = OLD.id;
END;
