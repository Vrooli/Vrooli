-- Development Toolchain Validator Schema
-- Stores reference scenarios, skill connections, expectations, and validation results

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Reference Scenario Registry (OT-P0-001)
-- Stores existing scenarios as references with template associations
CREATE TABLE IF NOT EXISTS reference_scenarios (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    slug VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    template VARCHAR(100) NOT NULL,
    path VARCHAR(500) NOT NULL,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_reference_scenarios_slug ON reference_scenarios(slug);
CREATE INDEX IF NOT EXISTS idx_reference_scenarios_template ON reference_scenarios(template);

-- Skill Connections (OT-P0-002)
-- Links prompt-manager skills to reference scenarios with version tracking
CREATE TABLE IF NOT EXISTS skill_connections (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reference_id UUID NOT NULL REFERENCES reference_scenarios(id) ON DELETE CASCADE,
    skill_id VARCHAR(255) NOT NULL,
    skill_version VARCHAR(50),
    skill_content_hash VARCHAR(64),
    connected_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(reference_id, skill_id)
);

CREATE INDEX IF NOT EXISTS idx_skill_connections_reference ON skill_connections(reference_id);
CREATE INDEX IF NOT EXISTS idx_skill_connections_skill ON skill_connections(skill_id);

-- Structural Expectations (OT-P0-004)
-- Defines expected folders, files, and content patterns for each skill-reference connection
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'expectation_type') THEN
        CREATE TYPE expectation_type AS ENUM ('folder', 'file', 'content_snippet');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS structural_expectations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    connection_id UUID NOT NULL REFERENCES skill_connections(id) ON DELETE CASCADE,
    type expectation_type NOT NULL,
    pattern VARCHAR(500) NOT NULL,
    required BOOLEAN NOT NULL DEFAULT true,
    expected_content TEXT,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_structural_expectations_connection ON structural_expectations(connection_id);

-- CLI Tool Assertions (OT-P0-005)
-- Defines CLI tool commands and JSONPath assertions for validation
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'assertion_operator') THEN
        CREATE TYPE assertion_operator AS ENUM (
            'eq', 'neq', 'gt', 'gte', 'lt', 'lte',
            'exists', 'contains', 'matches', 'between'
        );
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS cli_assertions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    connection_id UUID NOT NULL REFERENCES skill_connections(id) ON DELETE CASCADE,
    command VARCHAR(1000) NOT NULL,
    json_path VARCHAR(500) NOT NULL,
    operator assertion_operator NOT NULL,
    expected_value JSONB,
    description TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_cli_assertions_connection ON cli_assertions(connection_id);

-- Validation Results (for history and trend detection)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_type WHERE typname = 'validation_status') THEN
        CREATE TYPE validation_status AS ENUM ('pass', 'fail', 'error', 'skip');
    END IF;
END$$;

CREATE TABLE IF NOT EXISTS validation_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    reference_id UUID NOT NULL REFERENCES reference_scenarios(id) ON DELETE CASCADE,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE,
    overall_status validation_status NOT NULL DEFAULT 'pass'
);

CREATE INDEX IF NOT EXISTS idx_validation_runs_reference ON validation_runs(reference_id);
CREATE INDEX IF NOT EXISTS idx_validation_runs_started ON validation_runs(started_at DESC);

CREATE TABLE IF NOT EXISTS structural_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL REFERENCES validation_runs(id) ON DELETE CASCADE,
    expectation_id UUID NOT NULL REFERENCES structural_expectations(id) ON DELETE CASCADE,
    status validation_status NOT NULL,
    actual_value TEXT,
    error_message TEXT
);

CREATE INDEX IF NOT EXISTS idx_structural_results_run ON structural_results(run_id);

CREATE TABLE IF NOT EXISTS cli_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_id UUID NOT NULL REFERENCES validation_runs(id) ON DELETE CASCADE,
    assertion_id UUID NOT NULL REFERENCES cli_assertions(id) ON DELETE CASCADE,
    status validation_status NOT NULL,
    actual_value JSONB,
    error_message TEXT,
    execution_time_ms INTEGER
);

CREATE INDEX IF NOT EXISTS idx_cli_results_run ON cli_results(run_id);

-- Trigger to update timestamps
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS reference_scenarios_updated_at ON reference_scenarios;
CREATE TRIGGER reference_scenarios_updated_at
    BEFORE UPDATE ON reference_scenarios
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

DROP TRIGGER IF EXISTS skill_connections_updated_at ON skill_connections;
CREATE TRIGGER skill_connections_updated_at
    BEFORE UPDATE ON skill_connections
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();
