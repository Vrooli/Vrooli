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
