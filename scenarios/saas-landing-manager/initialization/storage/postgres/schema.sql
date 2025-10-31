-- SaaS Landing Manager Database Schema
-- This schema supports scenario detection, landing page management, template system, and A/B testing

-- SaaS Scenarios registry table
CREATE TABLE IF NOT EXISTS saas_scenarios (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_name VARCHAR(255) UNIQUE NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    saas_type VARCHAR(50) DEFAULT 'b2c_app', -- b2b_tool, b2c_app, api_service, marketplace
    industry VARCHAR(100),
    revenue_potential VARCHAR(100),
    has_landing_page BOOLEAN DEFAULT FALSE,
    landing_page_url VARCHAR(500),
    last_scan TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    confidence_score DECIMAL(3,2) DEFAULT 0.0, -- 0.0 to 1.0
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Landing Pages table
CREATE TABLE IF NOT EXISTS landing_pages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    scenario_id UUID NOT NULL REFERENCES saas_scenarios(id) ON DELETE CASCADE,
    template_id UUID,
    variant VARCHAR(50) DEFAULT 'control', -- control, a, b, etc.
    title VARCHAR(255) NOT NULL,
    description TEXT,
    content JSONB DEFAULT '{}', -- Page content configuration
    seo_metadata JSONB DEFAULT '{}', -- Meta tags, structured data, etc.
    performance_metrics JSONB DEFAULT '{}', -- Core Web Vitals, load times, etc.
    status VARCHAR(50) DEFAULT 'draft', -- draft, active, archived
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Templates table for reusable landing page designs
CREATE TABLE IF NOT EXISTS templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    category VARCHAR(50) NOT NULL, -- base, industry, component
    saas_type VARCHAR(50), -- Target SaaS type
    industry VARCHAR(100), -- Target industry
    html_content TEXT, -- HTML template content
    css_content TEXT, -- CSS styles
    js_content TEXT, -- JavaScript functionality
    config_schema JSONB DEFAULT '{}', -- Configuration options schema
    preview_url VARCHAR(500),
    usage_count INTEGER DEFAULT 0,
    rating DECIMAL(3,2) DEFAULT 0.0, -- Average user rating
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

ALTER TABLE templates
    ADD COLUMN IF NOT EXISTS saas_type VARCHAR(50),
    ADD COLUMN IF NOT EXISTS industry VARCHAR(100),
    ADD COLUMN IF NOT EXISTS html_content TEXT,
    ADD COLUMN IF NOT EXISTS css_content TEXT,
    ADD COLUMN IF NOT EXISTS js_content TEXT,
    ADD COLUMN IF NOT EXISTS config_schema JSONB DEFAULT '{}'::jsonb,
    ADD COLUMN IF NOT EXISTS preview_url VARCHAR(500),
    ADD COLUMN IF NOT EXISTS usage_count INTEGER DEFAULT 0,
    ADD COLUMN IF NOT EXISTS rating DECIMAL(3,2) DEFAULT 0.0,
    ADD COLUMN IF NOT EXISTS created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW();

ALTER TABLE templates
    ALTER COLUMN config_schema SET DEFAULT '{}'::jsonb,
    ALTER COLUMN usage_count SET DEFAULT 0,
    ALTER COLUMN rating SET DEFAULT 0.0,
    ALTER COLUMN created_at SET DEFAULT NOW(),
    ALTER COLUMN updated_at SET DEFAULT NOW();

-- A/B Test Results table for analytics
CREATE TABLE IF NOT EXISTS ab_test_results (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    landing_page_id UUID NOT NULL REFERENCES landing_pages(id) ON DELETE CASCADE,
    variant VARCHAR(50) NOT NULL,
    metric_name VARCHAR(100) NOT NULL, -- conversion_rate, bounce_rate, time_on_page, etc.
    metric_value DECIMAL(10,4) NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    session_id VARCHAR(255),
    user_agent TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Deployment History table
CREATE TABLE IF NOT EXISTS deployment_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    landing_page_id UUID NOT NULL REFERENCES landing_pages(id) ON DELETE CASCADE,
    target_scenario VARCHAR(255) NOT NULL,
    deployment_method VARCHAR(50) NOT NULL, -- direct, claude_agent
    status VARCHAR(50) NOT NULL, -- pending, in_progress, completed, failed
    agent_session_id VARCHAR(255), -- Claude Code agent session if applicable
    error_message TEXT,
    deployed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Analytics Summary table for performance tracking
CREATE TABLE IF NOT EXISTS analytics_summary (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    landing_page_id UUID NOT NULL REFERENCES landing_pages(id) ON DELETE CASCADE,
    date_range_start DATE NOT NULL,
    date_range_end DATE NOT NULL,
    unique_visitors INTEGER DEFAULT 0,
    page_views INTEGER DEFAULT 0,
    conversions INTEGER DEFAULT 0,
    conversion_rate DECIMAL(5,4) DEFAULT 0.0000,
    bounce_rate DECIMAL(5,4) DEFAULT 0.0000,
    avg_time_on_page INTEGER DEFAULT 0, -- seconds
    core_web_vitals JSONB DEFAULT '{}', -- LCP, FID, CLS metrics
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_saas_scenarios_confidence ON saas_scenarios(confidence_score DESC);
CREATE INDEX IF NOT EXISTS idx_saas_scenarios_last_scan ON saas_scenarios(last_scan DESC);
CREATE INDEX IF NOT EXISTS idx_saas_scenarios_type ON saas_scenarios(saas_type);
CREATE INDEX IF NOT EXISTS idx_landing_pages_scenario ON landing_pages(scenario_id);
CREATE INDEX IF NOT EXISTS idx_landing_pages_status ON landing_pages(status);
CREATE INDEX IF NOT EXISTS idx_landing_pages_variant ON landing_pages(variant);
CREATE INDEX IF NOT EXISTS idx_templates_category ON templates(category);
CREATE INDEX IF NOT EXISTS idx_templates_saas_type ON templates(saas_type);
CREATE INDEX IF NOT EXISTS idx_templates_usage ON templates(usage_count DESC);
CREATE INDEX IF NOT EXISTS idx_ab_test_results_page ON ab_test_results(landing_page_id);
CREATE INDEX IF NOT EXISTS idx_ab_test_results_timestamp ON ab_test_results(timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_deployment_history_page ON deployment_history(landing_page_id);
CREATE INDEX IF NOT EXISTS idx_analytics_summary_page ON analytics_summary(landing_page_id);
CREATE INDEX IF NOT EXISTS idx_analytics_summary_date ON analytics_summary(date_range_start, date_range_end);

-- Trigger to update updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE OR REPLACE TRIGGER update_saas_scenarios_updated_at 
    BEFORE UPDATE ON saas_scenarios 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE TRIGGER update_landing_pages_updated_at 
    BEFORE UPDATE ON landing_pages 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE TRIGGER update_templates_updated_at 
    BEFORE UPDATE ON templates 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

CREATE OR REPLACE TRIGGER update_deployment_history_updated_at 
    BEFORE UPDATE ON deployment_history 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();
