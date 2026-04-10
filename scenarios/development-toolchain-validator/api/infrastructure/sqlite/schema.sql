-- Development Toolchain Validator Schema (SQLite)
-- Stores reference scenarios, skill connections, expectations, and validation results

-- Reference Scenario Registry
CREATE TABLE IF NOT EXISTS reference_scenarios (
    id TEXT PRIMARY KEY,
    slug TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    template TEXT NOT NULL,
    path TEXT NOT NULL,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reference_scenarios_slug ON reference_scenarios(slug);
CREATE INDEX IF NOT EXISTS idx_reference_scenarios_template ON reference_scenarios(template);

-- Skill Connections
CREATE TABLE IF NOT EXISTS skill_connections (
    id TEXT PRIMARY KEY,
    reference_id TEXT NOT NULL REFERENCES reference_scenarios(id) ON DELETE CASCADE,
    skill_id TEXT NOT NULL,
    skill_version TEXT,
    skill_content_hash TEXT,
    connected_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(reference_id, skill_id)
);

CREATE INDEX IF NOT EXISTS idx_skill_connections_reference ON skill_connections(reference_id);
CREATE INDEX IF NOT EXISTS idx_skill_connections_skill ON skill_connections(skill_id);

-- Structural Expectations
CREATE TABLE IF NOT EXISTS structural_expectations (
    id TEXT PRIMARY KEY,
    connection_id TEXT NOT NULL REFERENCES skill_connections(id) ON DELETE CASCADE,
    type TEXT NOT NULL CHECK(type IN ('folder', 'file', 'content_snippet')),
    pattern TEXT NOT NULL,
    required INTEGER NOT NULL DEFAULT 1,
    expected_content TEXT,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_structural_expectations_connection ON structural_expectations(connection_id);

-- CLI Tool Assertions
CREATE TABLE IF NOT EXISTS cli_assertions (
    id TEXT PRIMARY KEY,
    connection_id TEXT NOT NULL REFERENCES skill_connections(id) ON DELETE CASCADE,
    command TEXT NOT NULL,
    json_path TEXT NOT NULL,
    operator TEXT NOT NULL CHECK(operator IN ('eq', 'neq', 'gt', 'gte', 'lt', 'lte', 'exists', 'contains', 'matches', 'between')),
    expected_value TEXT,
    description TEXT,
    created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cli_assertions_connection ON cli_assertions(connection_id);

-- Validation Runs (history tables - no repo interfaces yet)
CREATE TABLE IF NOT EXISTS validation_runs (
    id TEXT PRIMARY KEY,
    reference_id TEXT NOT NULL REFERENCES reference_scenarios(id) ON DELETE CASCADE,
    started_at DATETIME DEFAULT CURRENT_TIMESTAMP,
    completed_at DATETIME,
    overall_status TEXT NOT NULL DEFAULT 'pass' CHECK(overall_status IN ('pass', 'fail', 'error', 'skip'))
);

CREATE INDEX IF NOT EXISTS idx_validation_runs_reference ON validation_runs(reference_id);
CREATE INDEX IF NOT EXISTS idx_validation_runs_started ON validation_runs(started_at DESC);

CREATE TABLE IF NOT EXISTS structural_results (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES validation_runs(id) ON DELETE CASCADE,
    expectation_id TEXT NOT NULL REFERENCES structural_expectations(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK(status IN ('pass', 'fail', 'error', 'skip')),
    actual_value TEXT,
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS idx_structural_results_run ON structural_results(run_id);

CREATE TABLE IF NOT EXISTS cli_results (
    id TEXT PRIMARY KEY,
    run_id TEXT NOT NULL REFERENCES validation_runs(id) ON DELETE CASCADE,
    assertion_id TEXT NOT NULL REFERENCES cli_assertions(id) ON DELETE CASCADE,
    status TEXT NOT NULL CHECK(status IN ('pass', 'fail', 'error', 'skip')),
    actual_value TEXT,
    error_message TEXT,
    execution_time_ms INTEGER
);

CREATE INDEX IF NOT EXISTS idx_cli_results_run ON cli_results(run_id);
