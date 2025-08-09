-- Resource Experimenter Database Schema
-- Tracks experimental resources, test results, and integration attempts

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Enum types
CREATE TYPE experiment_status AS ENUM ('planned', 'running', 'completed', 'failed', 'abandoned');
CREATE TYPE resource_type AS ENUM ('ai', 'automation', 'storage', 'execution', 'monitoring', 'search', 'communication', 'other');
CREATE TYPE integration_status AS ENUM ('not_started', 'in_progress', 'successful', 'failed', 'partial');
CREATE TYPE test_result AS ENUM ('pass', 'fail', 'skip', 'error');

-- Experimental resources being evaluated
CREATE TABLE resources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255),
    description TEXT,
    type resource_type NOT NULL,
    
    -- Source information
    source_url TEXT,
    documentation_url TEXT,
    repository_url TEXT,
    docker_image VARCHAR(500),
    helm_chart VARCHAR(500),
    
    -- Technical details
    programming_language VARCHAR(100),
    license VARCHAR(100),
    version VARCHAR(50),
    last_updated DATE,
    stars INTEGER,
    
    -- Requirements
    min_memory_mb INTEGER,
    min_cpu_cores DECIMAL(3,1),
    min_storage_gb INTEGER,
    required_ports INTEGER[],
    dependencies TEXT[],
    
    -- Integration potential
    api_type VARCHAR(50), -- REST, GraphQL, gRPC, WebSocket, etc
    authentication_methods TEXT[],
    has_sdk BOOLEAN DEFAULT FALSE,
    has_cli BOOLEAN DEFAULT FALSE,
    has_webhooks BOOLEAN DEFAULT FALSE,
    
    -- Vrooli compatibility assessment
    compatibility_score DECIMAL(3,2) DEFAULT 0.00,
    integration_complexity VARCHAR(20) CHECK (integration_complexity IN ('trivial', 'easy', 'moderate', 'hard', 'extreme')),
    estimated_integration_hours INTEGER,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Experiments conducted on resources
CREATE TABLE experiments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_id UUID REFERENCES resources(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    hypothesis TEXT,
    status experiment_status DEFAULT 'planned',
    
    -- Configuration used
    docker_compose TEXT,
    environment_vars JSONB DEFAULT '{}'::jsonb,
    config_files JSONB DEFAULT '{}'::jsonb,
    
    -- Resource usage
    container_id VARCHAR(64),
    allocated_port INTEGER,
    memory_limit_mb INTEGER,
    cpu_limit DECIMAL(3,1),
    
    -- Timing
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    duration_minutes INTEGER,
    
    -- Results
    success BOOLEAN,
    findings TEXT,
    issues_encountered TEXT[],
    performance_metrics JSONB DEFAULT '{}'::jsonb,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Individual tests run during experiments
CREATE TABLE experiment_tests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    experiment_id UUID REFERENCES experiments(id) ON DELETE CASCADE,
    test_name VARCHAR(255) NOT NULL,
    test_type VARCHAR(50), -- connectivity, api, performance, integration, stress
    
    -- Test details
    test_script TEXT,
    input_data JSONB,
    expected_output JSONB,
    actual_output JSONB,
    
    -- Results
    result test_result NOT NULL,
    error_message TEXT,
    execution_time_ms INTEGER,
    
    -- Metrics
    cpu_usage_percent DECIMAL(5,2),
    memory_usage_mb INTEGER,
    network_bytes_sent BIGINT,
    network_bytes_received BIGINT,
    
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Integration attempts with Vrooli
CREATE TABLE integration_attempts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_id UUID REFERENCES resources(id) ON DELETE CASCADE,
    version VARCHAR(50),
    status integration_status DEFAULT 'not_started',
    
    -- Integration details
    integration_approach TEXT,
    provider_script TEXT,
    config_template JSONB,
    initialization_scripts TEXT[],
    
    -- Testing
    unit_tests_passed INTEGER DEFAULT 0,
    unit_tests_total INTEGER DEFAULT 0,
    integration_tests_passed INTEGER DEFAULT 0,
    integration_tests_total INTEGER DEFAULT 0,
    
    -- Code artifacts
    pull_request_url TEXT,
    commit_hash VARCHAR(40),
    files_changed TEXT[],
    lines_added INTEGER,
    lines_removed INTEGER,
    
    -- Review
    reviewed_by VARCHAR(255),
    review_notes TEXT,
    approved BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Templates for common resource patterns
CREATE TABLE resource_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    type resource_type NOT NULL,
    description TEXT,
    
    -- Template components
    docker_compose_template TEXT,
    env_template JSONB DEFAULT '{}'::jsonb,
    init_script_template TEXT,
    health_check_template TEXT,
    test_suite_template TEXT,
    
    -- Metadata
    applicable_to TEXT[], -- List of resource names this template works with
    success_rate DECIMAL(3,2) DEFAULT 0.00,
    times_used INTEGER DEFAULT 0,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Discovery queue for potential resources to evaluate
CREATE TABLE discovery_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    source_url TEXT,
    type resource_type,
    priority INTEGER DEFAULT 5 CHECK (priority >= 1 AND priority <= 10),
    
    -- Discovery metadata
    discovered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    discovered_by VARCHAR(50), -- manual, crawler, recommendation
    tags TEXT[],
    notes TEXT,
    
    -- Processing
    processed BOOLEAN DEFAULT FALSE,
    processed_at TIMESTAMP,
    resource_id UUID REFERENCES resources(id),
    skip_reason TEXT
);

-- Sandbox environments for testing
CREATE TABLE sandboxes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    experiment_id UUID REFERENCES experiments(id) ON DELETE CASCADE,
    
    -- Environment details
    sandbox_path TEXT NOT NULL,
    docker_network VARCHAR(100),
    isolated BOOLEAN DEFAULT TRUE,
    
    -- Resource allocation
    port_range_start INTEGER,
    port_range_end INTEGER,
    volume_mounts JSONB DEFAULT '[]'::jsonb,
    
    -- Status
    active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    destroyed_at TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_resources_type ON resources(type);
CREATE INDEX idx_resources_compatibility ON resources(compatibility_score DESC);
CREATE INDEX idx_experiments_status ON experiments(status);
CREATE INDEX idx_experiments_resource ON experiments(resource_id);
CREATE INDEX idx_experiment_tests_result ON experiment_tests(result);
CREATE INDEX idx_integration_status ON integration_attempts(status);
CREATE INDEX idx_discovery_processed ON discovery_queue(processed, priority DESC);

-- Full text search
CREATE INDEX idx_resources_name_trgm ON resources USING gin (name gin_trgm_ops);
CREATE INDEX idx_resources_description_trgm ON resources USING gin (description gin_trgm_ops);

-- Update timestamp triggers
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_resources_updated_at BEFORE UPDATE ON resources
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_experiments_updated_at BEFORE UPDATE ON experiments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_integration_attempts_updated_at BEFORE UPDATE ON integration_attempts
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_resource_templates_updated_at BEFORE UPDATE ON resource_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();