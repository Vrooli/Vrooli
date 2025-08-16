-- Agent Dashboard Schema
-- Tracks all active AI agents and their states

CREATE SCHEMA IF NOT EXISTS agent_dashboard;

-- Agent registry table
CREATE TABLE IF NOT EXISTS agent_dashboard.agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    type VARCHAR(100) NOT NULL, -- huginn, claude-code, agent-s2, n8n-workflow, etc.
    description TEXT,
    status VARCHAR(50) DEFAULT 'inactive', -- active, inactive, error, starting, stopping
    last_heartbeat TIMESTAMP,
    configuration JSONB DEFAULT '{}',
    capabilities JSONB DEFAULT '[]', -- array of capabilities this agent has
    metrics JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Agent activity logs
CREATE TABLE IF NOT EXISTS agent_dashboard.agent_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID REFERENCES agent_dashboard.agents(id) ON DELETE CASCADE,
    level VARCHAR(20) NOT NULL, -- debug, info, warning, error, critical
    message TEXT NOT NULL,
    context JSONB DEFAULT '{}',
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Agent tasks/jobs
CREATE TABLE IF NOT EXISTS agent_dashboard.agent_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID REFERENCES agent_dashboard.agents(id) ON DELETE CASCADE,
    task_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) DEFAULT 'pending', -- pending, running, completed, failed, cancelled
    input_data JSONB DEFAULT '{}',
    output_data JSONB DEFAULT '{}',
    error_message TEXT,
    started_at TIMESTAMP,
    completed_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Agent performance metrics
CREATE TABLE IF NOT EXISTS agent_dashboard.agent_metrics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID REFERENCES agent_dashboard.agents(id) ON DELETE CASCADE,
    metric_type VARCHAR(100) NOT NULL, -- cpu, memory, response_time, success_rate, etc.
    value NUMERIC NOT NULL,
    unit VARCHAR(50),
    tags JSONB DEFAULT '{}',
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Agent dependencies and relationships
CREATE TABLE IF NOT EXISTS agent_dashboard.agent_relationships (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_agent_id UUID REFERENCES agent_dashboard.agents(id) ON DELETE CASCADE,
    child_agent_id UUID REFERENCES agent_dashboard.agents(id) ON DELETE CASCADE,
    relationship_type VARCHAR(100) NOT NULL, -- depends_on, triggers, collaborates_with
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(parent_agent_id, child_agent_id, relationship_type)
);

-- Agent sessions (for tracking continuous agent runs)
CREATE TABLE IF NOT EXISTS agent_dashboard.agent_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id UUID REFERENCES agent_dashboard.agents(id) ON DELETE CASCADE,
    session_token VARCHAR(255) UNIQUE,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    total_tasks_completed INTEGER DEFAULT 0,
    total_errors INTEGER DEFAULT 0,
    metadata JSONB DEFAULT '{}'
);

-- Orchestration logs for tracking agent coordination
CREATE TABLE IF NOT EXISTS agent_dashboard.orchestration_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    orchestration_type VARCHAR(100) NOT NULL, -- optimize, scale, balance, emergency
    status VARCHAR(50) DEFAULT 'initiated', -- initiated, running, completed, failed
    affected_agents JSONB DEFAULT '[]', -- array of agent IDs affected
    configuration JSONB DEFAULT '{}',
    result JSONB DEFAULT '{}',
    error_message TEXT,
    initiated_by VARCHAR(255), -- user or system
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_agents_status ON agent_dashboard.agents(status);
CREATE INDEX idx_agents_type ON agent_dashboard.agents(type);
CREATE INDEX idx_agent_logs_agent_timestamp ON agent_dashboard.agent_logs(agent_id, timestamp DESC);
CREATE INDEX idx_agent_tasks_agent_status ON agent_dashboard.agent_tasks(agent_id, status);
CREATE INDEX idx_agent_metrics_agent_type_time ON agent_dashboard.agent_metrics(agent_id, metric_type, recorded_at DESC);
CREATE INDEX idx_orchestration_logs_type_status ON agent_dashboard.orchestration_logs(orchestration_type, status);

-- Triggers for updated_at
CREATE OR REPLACE FUNCTION agent_dashboard.update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_agents_updated_at BEFORE UPDATE ON agent_dashboard.agents
    FOR EACH ROW EXECUTE FUNCTION agent_dashboard.update_updated_at_column();