-- Test Genie Database Schema
-- Creates all tables, indexes, and constraints for the Test Genie system

-- Create database if it doesn't exist (run this separately)
-- CREATE DATABASE test_genie;

-- Connect to the test_genie database
\c test_genie;

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Test Suites table
CREATE TABLE IF NOT EXISTS test_suites (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_name VARCHAR(255) NOT NULL,
    suite_type VARCHAR(100) NOT NULL,
    test_types TEXT[], -- Array of test types: unit, integration, performance, vault, regression
    coverage_metrics JSONB DEFAULT '{}',
    generated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_executed TIMESTAMP,
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'deprecated', 'maintenance_required')),
    created_by VARCHAR(100),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Indexes
    CONSTRAINT unique_scenario_suite_type UNIQUE (scenario_name, suite_type)
);

CREATE INDEX IF NOT EXISTS idx_test_suites_scenario_name ON test_suites(scenario_name);
CREATE INDEX IF NOT EXISTS idx_test_suites_status ON test_suites(status);
CREATE INDEX IF NOT EXISTS idx_test_suites_generated_at ON test_suites(generated_at DESC);

-- Test Cases table
CREATE TABLE IF NOT EXISTS test_cases (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    suite_id UUID REFERENCES test_suites(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    test_type VARCHAR(50) NOT NULL CHECK (test_type IN ('unit', 'integration', 'performance', 'vault', 'regression')),
    test_code TEXT NOT NULL,
    expected_result TEXT,
    execution_timeout INTEGER DEFAULT 30,
    dependencies JSONB DEFAULT '[]',
    tags JSONB DEFAULT '[]',
    priority VARCHAR(20) DEFAULT 'medium' CHECK (priority IN ('critical', 'high', 'medium', 'low')),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT test_case_timeout_positive CHECK (execution_timeout > 0),
    CONSTRAINT unique_test_case_per_suite UNIQUE (suite_id, name)
);

CREATE INDEX IF NOT EXISTS idx_test_cases_suite_id ON test_cases(suite_id);
CREATE INDEX IF NOT EXISTS idx_test_cases_type ON test_cases(test_type);
CREATE INDEX IF NOT EXISTS idx_test_cases_priority ON test_cases(priority);
CREATE INDEX IF NOT EXISTS idx_test_cases_name_trgm ON test_cases USING GIN (name gin_trgm_ops);

-- Test Executions table
CREATE TABLE IF NOT EXISTS test_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    suite_id UUID REFERENCES test_suites(id),
    execution_type VARCHAR(50) NOT NULL CHECK (execution_type IN ('full', 'smoke', 'regression', 'performance', 'manual')),
    start_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    end_time TIMESTAMP,
    status VARCHAR(50) DEFAULT 'running' CHECK (status IN ('queued', 'running', 'completed', 'failed', 'cancelled')),
    performance_metrics JSONB DEFAULT '{}',
    environment VARCHAR(100) DEFAULT 'local',
    triggered_by VARCHAR(100),
    total_tests INTEGER DEFAULT 0,
    passed_tests INTEGER DEFAULT 0,
    failed_tests INTEGER DEFAULT 0,
    skipped_tests INTEGER DEFAULT 0,
    execution_notes TEXT,
    
    -- Constraints
    CONSTRAINT execution_end_after_start CHECK (end_time IS NULL OR end_time >= start_time),
    CONSTRAINT test_counts_valid CHECK (
        total_tests >= 0 AND 
        passed_tests >= 0 AND 
        failed_tests >= 0 AND 
        skipped_tests >= 0 AND
        total_tests >= (passed_tests + failed_tests + skipped_tests)
    )
);

CREATE INDEX IF NOT EXISTS idx_test_executions_suite_id ON test_executions(suite_id);
CREATE INDEX IF NOT EXISTS idx_test_executions_status ON test_executions(status);
CREATE INDEX IF NOT EXISTS idx_test_executions_start_time ON test_executions(start_time DESC);
CREATE INDEX IF NOT EXISTS idx_test_executions_environment ON test_executions(environment);

-- Test Results table
CREATE TABLE IF NOT EXISTS test_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    execution_id UUID REFERENCES test_executions(id) ON DELETE CASCADE,
    test_case_id UUID REFERENCES test_cases(id),
    status VARCHAR(50) NOT NULL CHECK (status IN ('passed', 'failed', 'skipped', 'error')),
    duration DECIMAL(10,3), -- Duration in seconds
    error_message TEXT,
    stack_trace TEXT,
    assertions JSONB DEFAULT '[]',
    artifacts JSONB DEFAULT '{}', -- Screenshots, logs, performance data
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    
    -- Constraints
    CONSTRAINT duration_non_negative CHECK (duration IS NULL OR duration >= 0),
    CONSTRAINT completion_after_start CHECK (completed_at IS NULL OR started_at IS NULL OR completed_at >= started_at)
);

CREATE INDEX IF NOT EXISTS idx_test_results_execution_id ON test_results(execution_id);
CREATE INDEX IF NOT EXISTS idx_test_results_test_case_id ON test_results(test_case_id);
CREATE INDEX IF NOT EXISTS idx_test_results_status ON test_results(status);
CREATE INDEX IF NOT EXISTS idx_test_results_duration ON test_results(duration DESC);

-- Coverage Analysis table
CREATE TABLE IF NOT EXISTS coverage_analysis (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_name VARCHAR(255) NOT NULL,
    analysis_type VARCHAR(50) DEFAULT 'comprehensive' CHECK (analysis_type IN ('basic', 'comprehensive', 'deep')),
    overall_coverage DECIMAL(5,2),
    coverage_by_file JSONB DEFAULT '{}',
    coverage_gaps JSONB DEFAULT '{}',
    improvement_suggestions JSONB DEFAULT '[]',
    priority_areas JSONB DEFAULT '[]',
    source_paths JSONB DEFAULT '[]',
    analyzed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    analysis_duration DECIMAL(8,3),
    
    -- Constraints
    CONSTRAINT coverage_percentage_valid CHECK (overall_coverage >= 0 AND overall_coverage <= 100),
    CONSTRAINT analysis_duration_positive CHECK (analysis_duration IS NULL OR analysis_duration > 0)
);

CREATE INDEX IF NOT EXISTS idx_coverage_analysis_scenario ON coverage_analysis(scenario_name);
CREATE INDEX IF NOT EXISTS idx_coverage_analysis_analyzed_at ON coverage_analysis(analyzed_at DESC);
CREATE INDEX IF NOT EXISTS idx_coverage_analysis_coverage ON coverage_analysis(overall_coverage DESC);

-- Test Vaults table
CREATE TABLE IF NOT EXISTS test_vaults (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_name VARCHAR(255) NOT NULL,
    vault_name VARCHAR(255) NOT NULL,
    phases JSONB DEFAULT '[]', -- Array of phase names
    phase_configurations JSONB DEFAULT '{}',
    success_criteria JSONB DEFAULT '{}',
    total_timeout INTEGER DEFAULT 3600, -- Total vault timeout in seconds
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_executed TIMESTAMP,
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'archived', 'deprecated')),
    
    -- Constraints
    CONSTRAINT vault_timeout_positive CHECK (total_timeout > 0),
    CONSTRAINT unique_vault_per_scenario UNIQUE (scenario_name, vault_name)
);

CREATE INDEX IF NOT EXISTS idx_test_vaults_scenario ON test_vaults(scenario_name);
CREATE INDEX IF NOT EXISTS idx_test_vaults_status ON test_vaults(status);
CREATE INDEX IF NOT EXISTS idx_test_vaults_created_at ON test_vaults(created_at DESC);

-- Vault Executions table
CREATE TABLE IF NOT EXISTS vault_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    vault_id UUID REFERENCES test_vaults(id),
    execution_type VARCHAR(50) DEFAULT 'full',
    start_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    end_time TIMESTAMP,
    current_phase VARCHAR(100),
    completed_phases JSONB DEFAULT '[]',
    failed_phases JSONB DEFAULT '[]',
    status VARCHAR(50) DEFAULT 'running' CHECK (status IN ('queued', 'running', 'completed', 'failed', 'cancelled')),
    phase_results JSONB DEFAULT '{}',
    environment VARCHAR(100) DEFAULT 'local',
    
    -- Constraints
    CONSTRAINT vault_execution_end_after_start CHECK (end_time IS NULL OR end_time >= start_time)
);

CREATE INDEX IF NOT EXISTS idx_vault_executions_vault_id ON vault_executions(vault_id);
CREATE INDEX IF NOT EXISTS idx_vault_executions_status ON vault_executions(status);
CREATE INDEX IF NOT EXISTS idx_vault_executions_start_time ON vault_executions(start_time DESC);

-- System Metrics table (for monitoring and analytics)
CREATE TABLE IF NOT EXISTS system_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    metric_name VARCHAR(100) NOT NULL,
    metric_value DECIMAL(15,6),
    metric_unit VARCHAR(50),
    tags JSONB DEFAULT '{}',
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Indexes for time-series queries
    FOREIGN KEY (metric_name) REFERENCES (VALUES ('test_generation_time'), ('test_execution_time'), ('coverage_analysis_time'), ('api_response_time'), ('system_load')) (column1) ON DELETE RESTRICT
);

CREATE INDEX IF NOT EXISTS idx_system_metrics_name ON system_metrics(metric_name);
CREATE INDEX IF NOT EXISTS idx_system_metrics_recorded_at ON system_metrics(recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_system_metrics_name_time ON system_metrics(metric_name, recorded_at DESC);

-- Test Patterns table (for AI model patterns and templates)
CREATE TABLE IF NOT EXISTS test_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    pattern_name VARCHAR(255) NOT NULL UNIQUE,
    pattern_type VARCHAR(50) NOT NULL CHECK (pattern_type IN ('unit', 'integration', 'performance', 'vault', 'regression')),
    template_code TEXT NOT NULL,
    description TEXT,
    variables JSONB DEFAULT '{}', -- Template variables and their types
    usage_count INTEGER DEFAULT 0,
    success_rate DECIMAL(5,2) DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Constraints
    CONSTRAINT success_rate_valid CHECK (success_rate >= 0 AND success_rate <= 100),
    CONSTRAINT usage_count_non_negative CHECK (usage_count >= 0)
);

CREATE INDEX IF NOT EXISTS idx_test_patterns_type ON test_patterns(pattern_type);
CREATE INDEX IF NOT EXISTS idx_test_patterns_usage ON test_patterns(usage_count DESC);
CREATE INDEX IF NOT EXISTS idx_test_patterns_success_rate ON test_patterns(success_rate DESC);

-- Create triggers for updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Apply triggers
CREATE TRIGGER update_test_suites_updated_at BEFORE UPDATE ON test_suites 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    
CREATE TRIGGER update_test_cases_updated_at BEFORE UPDATE ON test_cases 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
    
CREATE TRIGGER update_test_patterns_updated_at BEFORE UPDATE ON test_patterns 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create views for common queries
CREATE OR REPLACE VIEW test_suite_summary AS
SELECT 
    ts.id,
    ts.scenario_name,
    ts.suite_type,
    ts.status,
    ts.generated_at,
    ts.last_executed,
    COUNT(tc.id) as total_tests,
    COUNT(tc.id) FILTER (WHERE tc.test_type = 'unit') as unit_tests,
    COUNT(tc.id) FILTER (WHERE tc.test_type = 'integration') as integration_tests,
    COUNT(tc.id) FILTER (WHERE tc.test_type = 'performance') as performance_tests,
    COUNT(tc.id) FILTER (WHERE tc.test_type = 'vault') as vault_tests,
    COUNT(tc.id) FILTER (WHERE tc.test_type = 'regression') as regression_tests,
    (ts.coverage_metrics->>'code_coverage')::DECIMAL as coverage_percentage
FROM test_suites ts
LEFT JOIN test_cases tc ON ts.id = tc.suite_id
GROUP BY ts.id, ts.scenario_name, ts.suite_type, ts.status, ts.generated_at, ts.last_executed, ts.coverage_metrics;

CREATE OR REPLACE VIEW execution_summary AS
SELECT 
    te.id,
    te.suite_id,
    ts.scenario_name,
    te.execution_type,
    te.start_time,
    te.end_time,
    te.status,
    te.environment,
    te.total_tests,
    te.passed_tests,
    te.failed_tests,
    te.skipped_tests,
    CASE 
        WHEN te.total_tests > 0 THEN ROUND((te.passed_tests::DECIMAL / te.total_tests * 100), 2)
        ELSE 0 
    END as success_rate,
    EXTRACT(EPOCH FROM (COALESCE(te.end_time, CURRENT_TIMESTAMP) - te.start_time)) as duration_seconds
FROM test_executions te
JOIN test_suites ts ON te.suite_id = ts.id;

-- Create materialized view for performance metrics (refresh periodically)
CREATE MATERIALIZED VIEW IF NOT EXISTS daily_metrics AS
SELECT 
    DATE(start_time) as date,
    COUNT(*) as total_executions,
    COUNT(*) FILTER (WHERE status = 'completed') as successful_executions,
    COUNT(*) FILTER (WHERE status = 'failed') as failed_executions,
    AVG(EXTRACT(EPOCH FROM (end_time - start_time))) FILTER (WHERE end_time IS NOT NULL) as avg_duration,
    AVG(passed_tests::DECIMAL / NULLIF(total_tests, 0) * 100) as avg_success_rate
FROM test_executions 
WHERE start_time >= CURRENT_DATE - INTERVAL '90 days'
GROUP BY DATE(start_time)
ORDER BY date DESC;

-- Create index on materialized view
CREATE UNIQUE INDEX IF NOT EXISTS idx_daily_metrics_date ON daily_metrics(date);

-- Create a function to refresh metrics
CREATE OR REPLACE FUNCTION refresh_daily_metrics()
RETURNS void AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY daily_metrics;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions (adjust as needed for your setup)
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO test_genie_user;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO test_genie_user;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO test_genie_user;

-- Insert initial system info
INSERT INTO system_metrics (metric_name, metric_value, metric_unit, tags) VALUES
    ('schema_version', 1.0, 'version', '{"component": "database", "initialized_at": "' || CURRENT_TIMESTAMP || '"}'),
    ('initial_setup', 1, 'boolean', '{"status": "completed", "timestamp": "' || CURRENT_TIMESTAMP || '"}');

-- Success message
SELECT 'Test Genie database schema initialized successfully!' as status;