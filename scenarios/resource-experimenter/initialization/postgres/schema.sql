-- Resource Experimenter - Simple Database Schema
-- PostgreSQL database for AI-powered resource experimentation

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Experiments table - tracks resource integration experiments
CREATE TABLE IF NOT EXISTS experiments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    prompt TEXT NOT NULL,
    
    -- Resource details
    target_scenario VARCHAR(255) NOT NULL,
    new_resource VARCHAR(255) NOT NULL,
    existing_resources TEXT[] DEFAULT '{}',
    
    -- Generated content
    files_generated JSONB DEFAULT '{}',
    modifications_made JSONB DEFAULT '{}',
    
    -- Status tracking
    status VARCHAR(50) DEFAULT 'requested' CHECK (status IN ('requested', 'running', 'completed', 'failed')),
    experiment_id VARCHAR(255),
    
    -- Claude Code integration
    claude_prompt TEXT,
    claude_response TEXT,
    generation_error TEXT,
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_experiments_status ON experiments(status);
CREATE INDEX IF NOT EXISTS idx_experiments_created_at ON experiments(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_experiments_target_scenario ON experiments(target_scenario);
CREATE INDEX IF NOT EXISTS idx_experiments_new_resource ON experiments(new_resource);
CREATE INDEX IF NOT EXISTS idx_experiments_experiment_id ON experiments(experiment_id);

-- Experiment logs table - tracks AI interactions
CREATE TABLE IF NOT EXISTS experiment_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    experiment_id UUID REFERENCES experiments(id) ON DELETE CASCADE,
    
    -- Log entry details
    step VARCHAR(100) NOT NULL,
    prompt TEXT NOT NULL,
    response TEXT,
    success BOOLEAN DEFAULT false,
    error_message TEXT,
    
    -- Timing
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_seconds INTEGER
);

CREATE INDEX IF NOT EXISTS idx_experiment_logs_experiment_id ON experiment_logs(experiment_id);
CREATE INDEX IF NOT EXISTS idx_experiment_logs_step ON experiment_logs(step);
CREATE INDEX IF NOT EXISTS idx_experiment_logs_started_at ON experiment_logs(started_at DESC);

-- Experiment templates table - common patterns
CREATE TABLE IF NOT EXISTS experiment_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    
    -- Template content
    prompt_template TEXT NOT NULL,
    target_scenario_pattern VARCHAR(255), -- e.g., "*-dashboard", "research-*"
    resource_category VARCHAR(50), -- ai, storage, automation, etc.
    
    -- Usage tracking
    usage_count INTEGER DEFAULT 0,
    success_rate DECIMAL(5,2) DEFAULT 0,
    
    -- Template metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true
);

CREATE INDEX IF NOT EXISTS idx_experiment_templates_resource_category ON experiment_templates(resource_category);
CREATE INDEX IF NOT EXISTS idx_experiment_templates_active ON experiment_templates(is_active) WHERE is_active = true;

-- Scenarios catalog - available scenarios to experiment with
CREATE TABLE IF NOT EXISTS available_scenarios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255),
    description TEXT,
    path TEXT NOT NULL,
    
    -- Current resources used
    current_resources TEXT[] DEFAULT '{}',
    resource_categories TEXT[] DEFAULT '{}',
    
    -- Suitability for experimentation
    experimentation_friendly BOOLEAN DEFAULT true,
    complexity_level VARCHAR(20) DEFAULT 'intermediate' CHECK (complexity_level IN ('simple', 'intermediate', 'advanced')),
    last_experiment_date TIMESTAMP WITH TIME ZONE,
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_available_scenarios_experimentation_friendly ON available_scenarios(experimentation_friendly);
CREATE INDEX IF NOT EXISTS idx_available_scenarios_complexity_level ON available_scenarios(complexity_level);

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for automatic timestamp updates
CREATE TRIGGER update_experiments_updated_at BEFORE UPDATE ON experiments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_experiment_templates_updated_at BEFORE UPDATE ON experiment_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_available_scenarios_updated_at BEFORE UPDATE ON available_scenarios
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();