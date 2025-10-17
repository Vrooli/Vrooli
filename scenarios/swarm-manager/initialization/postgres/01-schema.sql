-- Swarm Manager Database Schema

-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS swarm_manager;

-- Set search path
SET search_path TO swarm_manager;

-- Task execution history
CREATE TABLE IF NOT EXISTS task_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id VARCHAR(255) NOT NULL,
    task_title TEXT NOT NULL,
    task_type VARCHAR(50),
    scenario_used VARCHAR(100),
    status VARCHAR(20) NOT NULL, -- pending, running, success, failed
    priority_score DECIMAL(10, 2),
    estimates JSONB,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    duration_seconds INTEGER,
    commands_executed JSONB,
    output_summary TEXT,
    error_details TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Priority weight configurations
CREATE TABLE IF NOT EXISTS priority_weights (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    weight_type VARCHAR(50) NOT NULL UNIQUE,
    weight_value DECIMAL(5, 2) NOT NULL,
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Learning patterns
CREATE TABLE IF NOT EXISTS learning_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_type VARCHAR(50) NOT NULL,
    trigger_conditions JSONB,
    successful_sequences JSONB,
    failure_patterns JSONB,
    success_rate DECIMAL(5, 2),
    times_used INTEGER DEFAULT 0,
    average_impact_score DECIMAL(5, 2),
    last_used TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Agent status tracking
CREATE TABLE IF NOT EXISTS agent_status (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_name VARCHAR(100) NOT NULL,
    current_task_id VARCHAR(255),
    current_task_title TEXT,
    status VARCHAR(20), -- idle, working, error
    started_at TIMESTAMP,
    last_heartbeat TIMESTAMP,
    resource_usage JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- System metrics
CREATE TABLE IF NOT EXISTS system_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    metric_type VARCHAR(50) NOT NULL,
    metric_value DECIMAL(10, 2),
    metric_data JSONB,
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Configuration settings
CREATE TABLE IF NOT EXISTS configuration (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    setting_key VARCHAR(100) NOT NULL UNIQUE,
    setting_value TEXT,
    setting_type VARCHAR(50), -- boolean, number, string, json
    description TEXT,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Problems discovered in the system
CREATE TABLE IF NOT EXISTS problems (
    id VARCHAR(100) PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT,
    severity VARCHAR(20) NOT NULL, -- critical, high, medium, low
    frequency VARCHAR(20) NOT NULL, -- constant, frequent, occasional, rare
    impact VARCHAR(30) NOT NULL, -- system_down, degraded_performance, user_impact, cosmetic
    status VARCHAR(20) NOT NULL DEFAULT 'active', -- active, investigating, resolved, ignored
    discovered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    discovered_by VARCHAR(100) NOT NULL,
    last_occurrence TIMESTAMP,
    source_file TEXT,
    affected_components JSONB DEFAULT '[]',
    symptoms JSONB DEFAULT '[]',
    evidence JSONB DEFAULT '{}',
    related_issues JSONB DEFAULT '[]',
    priority_estimates JSONB DEFAULT '{}',
    resolution TEXT,
    resolved_at TIMESTAMP,
    resolved_by VARCHAR(100),
    tasks_created JSONB DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_task_executions_status ON task_executions(status);
CREATE INDEX IF NOT EXISTS idx_task_executions_created ON task_executions(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_task_executions_task_id ON task_executions(task_id);
CREATE INDEX IF NOT EXISTS idx_agent_status_name ON agent_status(agent_name);
CREATE INDEX IF NOT EXISTS idx_learning_patterns_type ON learning_patterns(pattern_type);
CREATE INDEX IF NOT EXISTS idx_system_metrics_type_time ON system_metrics(metric_type, recorded_at DESC);
CREATE INDEX IF NOT EXISTS idx_problems_status ON problems(status);
CREATE INDEX IF NOT EXISTS idx_problems_severity ON problems(severity);
CREATE INDEX IF NOT EXISTS idx_problems_discovered ON problems(discovered_at DESC);
CREATE INDEX IF NOT EXISTS idx_problems_source ON problems(source_file);

-- Insert default priority weights
INSERT INTO priority_weights (weight_type, weight_value, description) VALUES
    ('impact', 1.0, 'Weight for task impact score (1-10)'),
    ('urgency', 0.8, 'Weight for task urgency (critical=4, high=3, medium=2, low=1)'),
    ('success', 0.6, 'Weight for success probability (0-1)'),
    ('cost', 0.5, 'Weight for resource cost (minimal=1, moderate=2, heavy=3)')
ON CONFLICT (weight_type) DO NOTHING;

-- Insert default configuration
INSERT INTO configuration (setting_key, setting_value, setting_type, description) VALUES
    ('max_concurrent_tasks', '5', 'number', 'Maximum number of tasks that can run simultaneously'),
    ('min_backlog_size', '10', 'number', 'Minimum backlog size before generating new tasks'),
    ('task_scan_interval', '300', 'number', 'Seconds between task folder scans'),
    ('yolo_mode', 'false', 'boolean', 'Auto-approve AI-generated tasks'),
    ('max_cpu_percent', '70', 'number', 'Maximum CPU usage percentage'),
    ('max_memory_percent', '70', 'number', 'Maximum memory usage percentage'),
    ('max_claude_calls_per_hour', '100', 'number', 'Rate limit for Claude API calls')
ON CONFLICT (setting_key) DO NOTHING;

-- Function to update timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_task_executions_updated_at BEFORE UPDATE ON task_executions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_priority_weights_updated_at BEFORE UPDATE ON priority_weights
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_learning_patterns_updated_at BEFORE UPDATE ON learning_patterns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_configuration_updated_at BEFORE UPDATE ON configuration
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_problems_updated_at BEFORE UPDATE ON problems
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();