-- Scenario Dependency Analyzer Database Schema
-- Stores dependency metadata, analysis results, and optimization recommendations

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Table to store individual scenario dependencies
CREATE TABLE IF NOT EXISTS scenario_dependencies (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_name VARCHAR(255) NOT NULL,
    dependency_type VARCHAR(50) NOT NULL CHECK (dependency_type IN ('resource', 'scenario', 'shared_workflow', 'cli_tool')),
    dependency_name VARCHAR(255) NOT NULL,
    required BOOLEAN NOT NULL DEFAULT true,
    purpose TEXT,
    access_method VARCHAR(255),
    configuration JSONB, -- Store any additional config like resource initialization files, ports, etc.
    discovered_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_verified TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for fast lookups by scenario
CREATE INDEX IF NOT EXISTS idx_scenario_dependencies_scenario_name ON scenario_dependencies (scenario_name);
CREATE INDEX IF NOT EXISTS idx_scenario_dependencies_type ON scenario_dependencies (dependency_type);
CREATE INDEX IF NOT EXISTS idx_scenario_dependencies_name ON scenario_dependencies (dependency_name);
CREATE INDEX IF NOT EXISTS idx_scenario_dependencies_required ON scenario_dependencies (required);

-- Table to store computed dependency graphs
CREATE TABLE IF NOT EXISTS dependency_graphs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    graph_type VARCHAR(50) NOT NULL CHECK (graph_type IN ('resource', 'scenario', 'combined', 'optimization')),
    title VARCHAR(255),
    description TEXT,
    nodes JSONB NOT NULL, -- Array of graph nodes with metadata
    edges JSONB NOT NULL, -- Array of graph edges with weights and properties
    metadata JSONB, -- Graph-level metadata like complexity scores, performance metrics
    filters JSONB, -- What filters were applied when generating this graph
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE -- For cached graphs that should be regenerated
);

-- Index for graph lookups
CREATE INDEX IF NOT EXISTS idx_dependency_graphs_type ON dependency_graphs (graph_type);
CREATE INDEX IF NOT EXISTS idx_dependency_graphs_expires ON dependency_graphs (expires_at) WHERE expires_at IS NOT NULL;

-- Table to store optimization recommendations
CREATE TABLE IF NOT EXISTS optimization_recommendations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_name VARCHAR(255) NOT NULL,
    recommendation_type VARCHAR(100) NOT NULL CHECK (
        recommendation_type IN (
            'resource_swap', 
            'dependency_reduction', 
            'merger_opportunity',
            'shared_workflow_adoption',
            'cost_optimization',
            'performance_improvement'
        )
    ),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    current_state JSONB NOT NULL, -- Current configuration/dependencies
    recommended_state JSONB NOT NULL, -- Recommended configuration/dependencies
    estimated_impact JSONB, -- Performance, cost, complexity impact estimates
    confidence_score DECIMAL(3,2) CHECK (confidence_score >= 0.0 AND confidence_score <= 1.0),
    priority VARCHAR(20) DEFAULT 'medium' CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'applied', 'rejected', 'expired')),
    applied_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for optimization recommendations
CREATE INDEX IF NOT EXISTS idx_optimization_recommendations_scenario ON optimization_recommendations (scenario_name);
CREATE INDEX IF NOT EXISTS idx_optimization_recommendations_type ON optimization_recommendations (recommendation_type);
CREATE INDEX IF NOT EXISTS idx_optimization_recommendations_status ON optimization_recommendations (status);
CREATE INDEX IF NOT EXISTS idx_optimization_recommendations_priority ON optimization_recommendations (priority);

-- Table to store analysis runs and their metadata
CREATE TABLE IF NOT EXISTS analysis_runs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    run_type VARCHAR(50) NOT NULL CHECK (run_type IN ('full_system', 'single_scenario', 'proposed_scenario', 'optimization')),
    target_scenarios TEXT[], -- Array of scenario names analyzed (empty for full system)
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    status VARCHAR(20) NOT NULL DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed', 'cancelled')),
    results_summary JSONB, -- Summary statistics like scenarios analyzed, dependencies found, etc.
    error_details TEXT, -- Error information if status is 'failed'
    triggered_by VARCHAR(100), -- What triggered this analysis (manual, event, scheduled)
    duration_ms INTEGER, -- Analysis duration in milliseconds
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for analysis runs
CREATE INDEX IF NOT EXISTS idx_analysis_runs_status ON analysis_runs (status);
CREATE INDEX IF NOT EXISTS idx_analysis_runs_type ON analysis_runs (run_type);
CREATE INDEX IF NOT EXISTS idx_analysis_runs_started ON analysis_runs (started_at);

-- Table to store scenario metadata cache for faster lookups
CREATE TABLE IF NOT EXISTS scenario_metadata (
    scenario_name VARCHAR(255) PRIMARY KEY,
    display_name VARCHAR(255),
    description TEXT,
    version VARCHAR(50),
    tags TEXT[],
    ports JSONB,
    service_config JSONB, -- Cached copy of .vrooli/service.json
    file_path VARCHAR(500), -- Path to scenario directory
    last_scanned TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_modified TIMESTAMP WITH TIME ZONE, -- File system modification time
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Index for scenario metadata
CREATE INDEX IF NOT EXISTS idx_scenario_metadata_active ON scenario_metadata (is_active);
CREATE INDEX IF NOT EXISTS idx_scenario_metadata_last_scanned ON scenario_metadata (last_scanned);

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers to automatically update updated_at columns
CREATE TRIGGER update_scenario_dependencies_updated_at 
    BEFORE UPDATE ON scenario_dependencies 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_optimization_recommendations_updated_at 
    BEFORE UPDATE ON optimization_recommendations 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_scenario_metadata_updated_at 
    BEFORE UPDATE ON scenario_metadata 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- View for easy dependency querying with human-readable information
CREATE OR REPLACE VIEW scenario_dependency_summary AS
SELECT 
    sd.scenario_name,
    sm.display_name,
    sd.dependency_type,
    sd.dependency_name,
    sd.required,
    sd.purpose,
    sd.access_method,
    sd.last_verified,
    COUNT(*) OVER (PARTITION BY sd.scenario_name) as total_dependencies,
    COUNT(*) FILTER (WHERE sd.required = true) OVER (PARTITION BY sd.scenario_name) as required_dependencies
FROM scenario_dependencies sd
LEFT JOIN scenario_metadata sm ON sd.scenario_name = sm.scenario_name
WHERE sm.is_active = true OR sm.is_active IS NULL;

-- View for optimization recommendations with priority scoring
CREATE OR REPLACE VIEW optimization_recommendations_prioritized AS
SELECT 
    or_.*,
    sm.display_name,
    CASE 
        WHEN or_.priority = 'critical' THEN 4
        WHEN or_.priority = 'high' THEN 3
        WHEN or_.priority = 'medium' THEN 2
        WHEN or_.priority = 'low' THEN 1
        ELSE 0
    END as priority_score,
    CASE
        WHEN or_.confidence_score >= 0.8 THEN 'high'
        WHEN or_.confidence_score >= 0.6 THEN 'medium'
        WHEN or_.confidence_score >= 0.4 THEN 'low'
        ELSE 'very_low'
    END as confidence_level
FROM optimization_recommendations or_
LEFT JOIN scenario_metadata sm ON or_.scenario_name = sm.scenario_name
WHERE or_.status IN ('pending', 'approved')
ORDER BY priority_score DESC, or_.confidence_score DESC, or_.created_at DESC;