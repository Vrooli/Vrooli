-- Funnel Builder Schema
-- Database schema for storing funnels, steps, leads, and analytics

-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS funnel_builder;

-- Set search path
SET search_path TO funnel_builder;

-- Enum types
CREATE TYPE funnel_status AS ENUM ('draft', 'active', 'archived');
CREATE TYPE step_type AS ENUM ('quiz', 'form', 'content', 'cta');
CREATE TYPE field_type AS ENUM ('text', 'email', 'tel', 'number', 'textarea', 'select', 'checkbox');

-- Funnels table
CREATE TABLE IF NOT EXISTS funnels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID, -- For multi-tenant support via scenario-authenticator
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    description TEXT,
    settings JSONB DEFAULT '{}'::jsonb,
    status funnel_status DEFAULT 'draft',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID,
    UNIQUE(tenant_id, slug)
);

-- Funnel steps table
CREATE TABLE IF NOT EXISTS funnel_steps (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    funnel_id UUID NOT NULL REFERENCES funnels(id) ON DELETE CASCADE,
    type step_type NOT NULL,
    position INTEGER NOT NULL,
    title VARCHAR(255) NOT NULL,
    content JSONB NOT NULL DEFAULT '{}'::jsonb,
    branching_rules JSONB DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(funnel_id, position)
);

-- Leads table
CREATE TABLE IF NOT EXISTS leads (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    funnel_id UUID NOT NULL REFERENCES funnels(id) ON DELETE CASCADE,
    tenant_id UUID,
    session_id UUID NOT NULL,
    email VARCHAR(255),
    phone VARCHAR(50),
    name VARCHAR(255),
    data JSONB DEFAULT '{}'::jsonb, -- All captured form data
    source VARCHAR(255), -- utm_source, referrer, etc.
    ip_address INET,
    user_agent TEXT,
    completed BOOLEAN DEFAULT FALSE,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Step responses table (tracks individual step completions)
CREATE TABLE IF NOT EXISTS step_responses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    step_id UUID NOT NULL REFERENCES funnel_steps(id) ON DELETE CASCADE,
    response JSONB NOT NULL DEFAULT '{}'::jsonb,
    duration_ms INTEGER, -- Time spent on step in milliseconds
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Analytics events table (for detailed tracking)
CREATE TABLE IF NOT EXISTS analytics_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    funnel_id UUID NOT NULL REFERENCES funnels(id) ON DELETE CASCADE,
    lead_id UUID REFERENCES leads(id) ON DELETE SET NULL,
    event_type VARCHAR(50) NOT NULL, -- 'view', 'start', 'step_complete', 'abandon', 'complete'
    step_id UUID REFERENCES funnel_steps(id) ON DELETE SET NULL,
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

-- Funnel templates table
CREATE TABLE IF NOT EXISTS funnel_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL,
    description TEXT,
    category VARCHAR(100),
    thumbnail_url TEXT,
    template_data JSONB NOT NULL, -- Complete funnel structure
    metrics JSONB DEFAULT '{}'::jsonb, -- avg conversion rate, typical steps, etc.
    is_public BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- A/B test variants table
CREATE TABLE IF NOT EXISTS ab_test_variants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    funnel_id UUID NOT NULL REFERENCES funnels(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    traffic_percentage INTEGER DEFAULT 50,
    is_control BOOLEAN DEFAULT FALSE,
    variant_data JSONB NOT NULL, -- Overrides for specific steps/settings
    metrics JSONB DEFAULT '{}'::jsonb, -- Performance metrics
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX idx_funnels_tenant_id ON funnels(tenant_id);
CREATE INDEX idx_funnels_status ON funnels(status);
CREATE INDEX idx_funnels_slug ON funnels(slug);
CREATE INDEX idx_funnel_steps_funnel_id ON funnel_steps(funnel_id);
CREATE INDEX idx_funnel_steps_position ON funnel_steps(funnel_id, position);
CREATE INDEX idx_leads_funnel_id ON leads(funnel_id);
CREATE INDEX idx_leads_email ON leads(email);
CREATE INDEX idx_leads_session_id ON leads(session_id);
CREATE INDEX idx_leads_completed ON leads(completed);
CREATE INDEX idx_leads_created_at ON leads(created_at);
CREATE INDEX idx_step_responses_lead_id ON step_responses(lead_id);
CREATE INDEX idx_step_responses_step_id ON step_responses(step_id);
CREATE INDEX idx_analytics_events_funnel_id ON analytics_events(funnel_id);
CREATE INDEX idx_analytics_events_lead_id ON analytics_events(lead_id);
CREATE INDEX idx_analytics_events_event_type ON analytics_events(event_type);
CREATE INDEX idx_analytics_events_created_at ON analytics_events(created_at);
CREATE INDEX idx_ab_test_variants_funnel_id ON ab_test_variants(funnel_id);

-- Triggers for updated_at
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER update_funnels_updated_at
    BEFORE UPDATE ON funnels
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_funnel_steps_updated_at
    BEFORE UPDATE ON funnel_steps
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_leads_updated_at
    BEFORE UPDATE ON leads
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_funnel_templates_updated_at
    BEFORE UPDATE ON funnel_templates
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_ab_test_variants_updated_at
    BEFORE UPDATE ON ab_test_variants
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at();

-- Views for analytics
CREATE OR REPLACE VIEW funnel_performance AS
SELECT 
    f.id,
    f.name,
    f.status,
    COUNT(DISTINCT l.id) as total_leads,
    COUNT(DISTINCT CASE WHEN l.completed THEN l.id END) as completed_leads,
    ROUND(100.0 * COUNT(DISTINCT CASE WHEN l.completed THEN l.id END) / NULLIF(COUNT(DISTINCT l.id), 0), 2) as conversion_rate,
    AVG(EXTRACT(EPOCH FROM (l.completed_at - l.created_at))) as avg_completion_time_seconds
FROM funnels f
LEFT JOIN leads l ON f.id = l.funnel_id
GROUP BY f.id, f.name, f.status;

CREATE OR REPLACE VIEW step_performance AS
SELECT
    fs.id,
    fs.funnel_id,
    fs.title,
    fs.type,
    fs.position,
    COUNT(DISTINCT sr.lead_id) as responses,
    AVG(sr.duration_ms) as avg_duration_ms,
    COUNT(DISTINCT l.id) as total_reached,
    ROUND(100.0 * COUNT(DISTINCT sr.lead_id) / NULLIF(COUNT(DISTINCT l.id), 0), 2) as completion_rate
FROM funnel_steps fs
LEFT JOIN step_responses sr ON fs.id = sr.step_id
LEFT JOIN leads l ON sr.lead_id = l.id
GROUP BY fs.id, fs.funnel_id, fs.title, fs.type, fs.position;

-- Grant permissions (adjust based on your user setup)
GRANT USAGE ON SCHEMA funnel_builder TO PUBLIC;
GRANT ALL ON ALL TABLES IN SCHEMA funnel_builder TO PUBLIC;
GRANT ALL ON ALL SEQUENCES IN SCHEMA funnel_builder TO PUBLIC;