CREATE TABLE IF NOT EXISTS analysis_operations (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    apply_graph INTEGER NOT NULL,
    signature TEXT NOT NULL,
    state TEXT NOT NULL,
    created_at TEXT NOT NULL,
    started_at TEXT,
    completed_at TEXT,
    total INTEGER NOT NULL,
    completed INTEGER NOT NULL DEFAULT 0,
    failures INTEGER NOT NULL DEFAULT 0,
    error TEXT NOT NULL DEFAULT '',
    graph_json TEXT,
    details_pruned INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS analysis_operations_active_signature
    ON analysis_operations(signature) WHERE state IN ('queued', 'running');
CREATE INDEX IF NOT EXISTS analysis_operations_state ON analysis_operations(state);
CREATE INDEX IF NOT EXISTS analysis_operations_details ON analysis_operations(details_pruned);

CREATE TABLE IF NOT EXISTS analysis_operation_requests (
    caller_id TEXT PRIMARY KEY,
    operation_id TEXT NOT NULL REFERENCES analysis_operations(id),
    signature TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS analysis_operation_requests_operation
    ON analysis_operation_requests(operation_id);

CREATE TABLE IF NOT EXISTS analysis_operation_scenarios (
    operation_id TEXT NOT NULL REFERENCES analysis_operations(id),
    scenario TEXT NOT NULL,
    state TEXT NOT NULL,
    detected_resources INTEGER NOT NULL DEFAULT 0,
    scenario_dependencies INTEGER NOT NULL DEFAULT 0,
    deployment_dependencies INTEGER NOT NULL DEFAULT 0,
    duration_ns INTEGER NOT NULL DEFAULT 0,
    error TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (operation_id, scenario)
);
