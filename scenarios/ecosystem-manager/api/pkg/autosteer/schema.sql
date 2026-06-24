-- Auto Steer controller tables — owned by pkg/autosteer, applied via
-- database.EnsureSchemas at boot (see pkg/dbschema). SQLite dialect:
-- forward-only declarative (CREATE TABLE IF NOT EXISTS), JSON payloads in TEXT
-- columns, app-generated string IDs, RFC3339/SQLite timestamps. Last-updated is
-- maintained in application code (ExecutionStateManager.Save), not a trigger.

-- Profile executions (historical tracking) for analytics and learning.
CREATE TABLE IF NOT EXISTS profile_executions (
    id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL,
    task_id TEXT NOT NULL,
    scenario_name TEXT NOT NULL,
    phase_breakdown TEXT,
    total_iterations INTEGER DEFAULT 0,
    total_duration_ms INTEGER,
    user_rating INTEGER CHECK (user_rating IS NULL OR (user_rating >= 1 AND user_rating <= 5)),
    user_comments TEXT,
    user_feedback_at TIMESTAMP,
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT profile_executions_task_unique UNIQUE (task_id)
);

CREATE INDEX IF NOT EXISTS idx_profile_executions_profile_id ON profile_executions(profile_id);
CREATE INDEX IF NOT EXISTS idx_profile_executions_scenario ON profile_executions(scenario_name);
CREATE INDEX IF NOT EXISTS idx_profile_executions_executed_at ON profile_executions(executed_at DESC);

-- Active controller state (objective-controller shape: one row per in-flight task).
CREATE TABLE IF NOT EXISTS profile_execution_state (
    task_id TEXT PRIMARY KEY,
    profile_id TEXT NOT NULL,
    iteration INTEGER NOT NULL DEFAULT 0,
    current_skill TEXT NOT NULL DEFAULT '',
    current_rationale TEXT NOT NULL DEFAULT '',
    findings TEXT,
    score_history TEXT,
    trace TEXT,
    metrics TEXT,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_profile_execution_state_profile_id ON profile_execution_state(profile_id);

-- Per-iteration decision trace. Persists the controller's reasoning so it
-- survives run finalization (the live state.Trace is deleted on finish).
CREATE TABLE IF NOT EXISTS decision_trace (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    task_id TEXT NOT NULL,
    profile_id TEXT NOT NULL,
    scenario_name TEXT NOT NULL DEFAULT '',
    iteration INTEGER NOT NULL,
    chosen_skill TEXT NOT NULL DEFAULT '',
    heaviest_dimension TEXT NOT NULL DEFAULT '',
    rationale TEXT NOT NULL DEFAULT '',
    dimension_scores TEXT,
    fingerprint TEXT NOT NULL DEFAULT '',
    score_before REAL NOT NULL DEFAULT 0,
    score_after REAL NOT NULL DEFAULT 0,
    realized_delta REAL NOT NULL DEFAULT 0,
    gaming_cause TEXT NOT NULL DEFAULT '',
    halt_reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_decision_trace_task ON decision_trace(task_id, iteration);

-- Structured feedback annotations for completed runs.
CREATE TABLE IF NOT EXISTS execution_feedback_entries (
    id TEXT PRIMARY KEY,
    execution_task_id TEXT NOT NULL,
    category TEXT NOT NULL,
    severity TEXT NOT NULL,
    suggested_action TEXT,
    comments TEXT,
    metadata TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_execution_feedback_entries_task_id ON execution_feedback_entries(execution_task_id);
