-- Scenario Generator V1 - Comprehensive Database Schema
-- PostgreSQL database for autonomous scenario generation pipeline

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =============================================
-- CAMPAIGN MANAGEMENT TABLES
-- =============================================

-- Campaigns organize scenarios by business category or client
CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Visual organization
    color_theme VARCHAR(20) DEFAULT 'blue',
    icon_name VARCHAR(50) DEFAULT 'folder',
    
    -- Business context
    client_name VARCHAR(255),
    target_market VARCHAR(100),
    business_category VARCHAR(50),
    
    -- Status and metrics
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'completed', 'archived')),
    scenario_count INTEGER DEFAULT 0,
    success_rate DECIMAL(5,2) DEFAULT 0,
    total_revenue_potential DECIMAL(12,2) DEFAULT 0,
    
    -- Audit fields
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(100) DEFAULT 'system'
);

CREATE INDEX idx_campaigns_status ON campaigns(status);
CREATE INDEX idx_campaigns_business_category ON campaigns(business_category);
CREATE INDEX idx_campaigns_created_at ON campaigns(created_at DESC);

-- =============================================
-- SCENARIO LIFECYCLE TABLES
-- =============================================

-- Main scenarios table tracking the complete generation lifecycle
CREATE TABLE IF NOT EXISTS scenarios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    campaign_id UUID REFERENCES campaigns(id) ON DELETE CASCADE,
    
    -- Identification
    scenario_id VARCHAR(100) UNIQUE NOT NULL,
    scenario_name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Input and requirements
    user_request TEXT NOT NULL,
    complexity VARCHAR(20) NOT NULL CHECK (complexity IN ('simple', 'intermediate', 'advanced')),
    category VARCHAR(50) NOT NULL,
    required_resources JSONB NOT NULL DEFAULT '[]'::jsonb,
    optional_resources JSONB DEFAULT '[]'::jsonb,
    
    -- Generation configuration
    planning_iterations INTEGER DEFAULT 3,
    implementation_iterations INTEGER DEFAULT 2,
    validation_iterations INTEGER DEFAULT 5,
    
    -- Status tracking
    status VARCHAR(20) DEFAULT 'requested' CHECK (status IN (
        'requested', 'planning', 'implementing', 'validating', 'deploying', 
        'completed', 'failed', 'archived'
    )),
    current_phase VARCHAR(20) DEFAULT 'planning',
    progress_percent INTEGER DEFAULT 0 CHECK (progress_percent >= 0 AND progress_percent <= 100),
    
    -- Timing metrics
    requested_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    total_duration_minutes INTEGER,
    
    -- Storage locations
    minio_bucket VARCHAR(100) DEFAULT 'scenario-artifacts',
    plan_path VARCHAR(500),
    implementation_path VARCHAR(500),
    final_scenario_path VARCHAR(500),
    
    -- Business metrics
    estimated_revenue JSONB DEFAULT '{"min": 0, "max": 0, "currency": "USD"}'::jsonb,
    
    -- Error handling
    error_message TEXT,
    retry_count INTEGER DEFAULT 0,
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb,
    tags TEXT[] DEFAULT '{}',
    
    -- Audit
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_scenarios_campaign_id ON scenarios(campaign_id);
CREATE INDEX idx_scenarios_status ON scenarios(status);
CREATE INDEX idx_scenarios_category ON scenarios(category);
CREATE INDEX idx_scenarios_requested_at ON scenarios(requested_at DESC);
CREATE INDEX idx_scenarios_resources ON scenarios USING GIN(required_resources);
CREATE INDEX idx_scenarios_tags ON scenarios USING GIN(tags);

-- =============================================
-- GENERATION PIPELINE TRACKING
-- =============================================

-- Detailed logs of all Claude Code interactions
CREATE TABLE IF NOT EXISTS claude_interactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_id UUID REFERENCES scenarios(id) ON DELETE CASCADE,
    
    -- Interaction context
    phase VARCHAR(20) NOT NULL CHECK (phase IN ('planning', 'implementation', 'validation', 'debugging')),
    iteration_number INTEGER NOT NULL,
    interaction_type VARCHAR(20) NOT NULL CHECK (interaction_type IN ('prompt', 'response', 'error')),
    
    -- Content
    prompt_template VARCHAR(100), -- references prompt file used
    prompt_content TEXT,
    response_content TEXT,
    
    -- Timing
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    
    -- Quality metrics
    success BOOLEAN DEFAULT true,
    confidence_score DECIMAL(3,2), -- Claude's confidence in response
    
    -- Token usage (for cost tracking)
    input_tokens INTEGER,
    output_tokens INTEGER,
    estimated_cost_usd DECIMAL(8,6),
    
    -- Metadata
    claude_model VARCHAR(50),
    temperature DECIMAL(3,2),
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_claude_interactions_scenario_id ON claude_interactions(scenario_id);
CREATE INDEX idx_claude_interactions_phase ON claude_interactions(phase);
CREATE INDEX idx_claude_interactions_started_at ON claude_interactions(started_at DESC);

-- Generation steps tracking individual workflow steps
CREATE TABLE IF NOT EXISTS generation_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_id UUID REFERENCES scenarios(id) ON DELETE CASCADE,
    
    -- Step identification
    phase VARCHAR(20) NOT NULL,
    step_name VARCHAR(100) NOT NULL,
    step_order INTEGER NOT NULL,
    
    -- Status
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'running', 'completed', 'failed', 'skipped')),
    
    -- Timing
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    
    -- Results
    output_data JSONB DEFAULT '{}'::jsonb,
    error_message TEXT,
    
    -- Metadata
    metadata JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_generation_steps_scenario_id ON generation_steps(scenario_id);
CREATE INDEX idx_generation_steps_phase ON generation_steps(phase);

-- =============================================
-- PATTERN ANALYSIS & LEARNING
-- =============================================

-- Issues and solutions encountered during generation
CREATE TABLE IF NOT EXISTS improvement_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Pattern identification
    issue_category VARCHAR(50) NOT NULL,
    issue_description TEXT NOT NULL,
    solution_description TEXT NOT NULL,
    
    -- Context
    scenario_phase VARCHAR(20) NOT NULL,
    complexity VARCHAR(20),
    resources_involved JSONB DEFAULT '[]'::jsonb,
    
    -- Pattern strength
    occurrence_count INTEGER DEFAULT 1,
    success_rate DECIMAL(5,2) DEFAULT 0,
    confidence_score DECIMAL(3,2) DEFAULT 0,
    
    -- Examples
    example_scenarios JSONB DEFAULT '[]'::jsonb,
    
    -- Learning metadata
    first_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    is_active BOOLEAN DEFAULT true,
    
    -- Auto-generated insights
    suggested_improvements TEXT,
    prevention_strategies TEXT
);

CREATE INDEX idx_improvement_patterns_category ON improvement_patterns(issue_category);
CREATE INDEX idx_improvement_patterns_phase ON improvement_patterns(scenario_phase);
CREATE INDEX idx_improvement_patterns_confidence ON improvement_patterns(confidence_score DESC);

-- Validation results from scenario-to-app.sh
CREATE TABLE IF NOT EXISTS validation_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_id UUID REFERENCES scenarios(id) ON DELETE CASCADE,
    
    -- Validation context
    validation_attempt INTEGER NOT NULL,
    validation_type VARCHAR(20) DEFAULT 'dry_run' CHECK (validation_type IN ('dry_run', 'full_deploy')),
    
    -- Results
    success BOOLEAN NOT NULL,
    exit_code INTEGER,
    stdout_output TEXT,
    stderr_output TEXT,
    
    -- Analysis
    issues_found JSONB DEFAULT '[]'::jsonb,
    fixes_applied JSONB DEFAULT '[]'::jsonb,
    
    -- Timing
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    duration_ms INTEGER,
    
    -- Metadata
    script_version VARCHAR(20),
    environment_info JSONB DEFAULT '{}'::jsonb
);

CREATE INDEX idx_validation_results_scenario_id ON validation_results(scenario_id);
CREATE INDEX idx_validation_results_success ON validation_results(success);

-- =============================================
-- ANALYTICS AND REPORTING
-- =============================================

-- Daily analytics aggregated data
CREATE TABLE IF NOT EXISTS daily_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    
    -- Time dimension
    analytics_date DATE NOT NULL,
    
    -- Volume metrics
    scenarios_requested INTEGER DEFAULT 0,
    scenarios_completed INTEGER DEFAULT 0,
    scenarios_failed INTEGER DEFAULT 0,
    
    -- Performance metrics
    avg_generation_time_minutes DECIMAL(8,2),
    avg_planning_iterations DECIMAL(4,2),
    avg_implementation_iterations DECIMAL(4,2),
    avg_validation_iterations DECIMAL(4,2),
    
    -- Quality metrics
    success_rate DECIMAL(5,2),
    first_time_validation_success_rate DECIMAL(5,2),
    
    -- Resource utilization
    most_used_resources JSONB DEFAULT '[]'::jsonb,
    resource_combinations JSONB DEFAULT '{}'::jsonb,
    
    -- Categories and complexity
    category_breakdown JSONB DEFAULT '{}'::jsonb,
    complexity_breakdown JSONB DEFAULT '{}'::jsonb,
    
    -- Revenue potential
    estimated_revenue_total DECIMAL(12,2) DEFAULT 0,
    
    -- Claude usage
    total_claude_interactions INTEGER DEFAULT 0,
    total_tokens_used INTEGER DEFAULT 0,
    total_claude_cost_usd DECIMAL(8,2) DEFAULT 0,
    
    UNIQUE(analytics_date)
);

CREATE INDEX idx_daily_analytics_date ON daily_analytics(analytics_date DESC);

-- =============================================
-- FUNCTIONS AND TRIGGERS
-- =============================================

-- Function to update updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Function to update campaign metrics when scenarios change
CREATE OR REPLACE FUNCTION update_campaign_metrics()
RETURNS TRIGGER AS $$
BEGIN
    -- Update scenario count and success rate for the campaign
    UPDATE campaigns 
    SET 
        scenario_count = (
            SELECT COUNT(*) FROM scenarios WHERE campaign_id = COALESCE(NEW.campaign_id, OLD.campaign_id)
        ),
        success_rate = (
            SELECT COALESCE(
                ROUND(
                    100.0 * COUNT(CASE WHEN status = 'completed' THEN 1 END) / NULLIF(COUNT(*), 0), 
                    2
                ), 0
            )
            FROM scenarios WHERE campaign_id = COALESCE(NEW.campaign_id, OLD.campaign_id)
        ),
        total_revenue_potential = (
            SELECT COALESCE(
                SUM((estimated_revenue->>'max')::DECIMAL), 0
            )
            FROM scenarios 
            WHERE campaign_id = COALESCE(NEW.campaign_id, OLD.campaign_id) 
              AND status = 'completed'
        ),
        updated_at = NOW()
    WHERE id = COALESCE(NEW.campaign_id, OLD.campaign_id);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ language 'plpgsql';

-- Create triggers
CREATE TRIGGER update_campaigns_updated_at 
    BEFORE UPDATE ON campaigns 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_scenarios_updated_at 
    BEFORE UPDATE ON scenarios 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_campaign_metrics_on_scenario_change
    AFTER INSERT OR UPDATE OR DELETE ON scenarios
    FOR EACH ROW
    EXECUTE FUNCTION update_campaign_metrics();

-- =============================================
-- USEFUL VIEWS
-- =============================================

-- Campaign dashboard summary
CREATE OR REPLACE VIEW campaign_dashboard AS
SELECT 
    c.*,
    COUNT(s.id) as current_scenario_count,
    COUNT(CASE WHEN s.status = 'completed' THEN 1 END) as completed_scenarios,
    COUNT(CASE WHEN s.status = 'failed' THEN 1 END) as failed_scenarios,
    COUNT(CASE WHEN s.status IN ('requested', 'planning', 'implementing', 'validating') THEN 1 END) as active_scenarios,
    AVG(s.total_duration_minutes) as avg_duration_minutes,
    SUM((s.estimated_revenue->>'max')::DECIMAL) as total_estimated_revenue
FROM campaigns c
LEFT JOIN scenarios s ON c.id = s.campaign_id
GROUP BY c.id;

-- Scenario generation pipeline status
CREATE OR REPLACE VIEW scenario_pipeline_status AS
SELECT 
    s.*,
    c.name as campaign_name,
    c.color_theme as campaign_color,
    COUNT(ci.id) as total_claude_interactions,
    SUM(ci.input_tokens + ci.output_tokens) as total_tokens,
    SUM(ci.estimated_cost_usd) as total_cost_usd,
    MAX(vr.success) as last_validation_success
FROM scenarios s
LEFT JOIN campaigns c ON s.campaign_id = c.id
LEFT JOIN claude_interactions ci ON s.id = ci.scenario_id
LEFT JOIN validation_results vr ON s.id = vr.scenario_id
GROUP BY s.id, c.name, c.color_theme;

-- Pattern analysis insights
CREATE OR REPLACE VIEW pattern_insights AS
SELECT 
    issue_category,
    COUNT(*) as pattern_count,
    AVG(success_rate) as avg_success_rate,
    MAX(confidence_score) as max_confidence,
    STRING_AGG(DISTINCT scenario_phase, ', ') as affected_phases,
    array_agg(DISTINCT resources_involved) as common_resources
FROM improvement_patterns
WHERE is_active = true
GROUP BY issue_category
ORDER BY pattern_count DESC;

-- Initial success message
DO $$
BEGIN
    RAISE NOTICE 'Scenario Generator V1 comprehensive database schema created successfully';
    RAISE NOTICE 'Core tables: campaigns, scenarios, claude_interactions, generation_steps';
    RAISE NOTICE 'Analytics tables: improvement_patterns, validation_results, daily_analytics';  
    RAISE NOTICE 'Views: campaign_dashboard, scenario_pipeline_status, pattern_insights';
END $$;