
-- Migration 004: Add analytics fields to api_usage_logs
-- Adds tracking for requests, data volume, errors, and user identification

-- Add new columns to api_usage_logs






-- Create indexes for analytics queries
CREATE INDEX IF NOT EXISTS idx_usage_logs_user_id ON api_usage_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_timestamp ON api_usage_logs(timestamp);
CREATE INDEX IF NOT EXISTS idx_usage_logs_api_timestamp ON api_usage_logs(api_id, timestamp);

-- Add version column to apis table for tracking API versions


-- CREATE TABLE IF NOT EXISTS for API version history
CREATE TABLE IF NOT EXISTS api_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    change_summary TEXT,
    breaking_changes BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_api_versions_api_id ON api_versions(api_id);
CREATE INDEX IF NOT EXISTS idx_api_versions_created ON api_versions(created_at);
-- Migration: Add webhook support for API update notifications
-- This adds tables and functionality for webhook subscriptions and event delivery

-- Webhook subscriptions table
CREATE TABLE IF NOT EXISTS webhook_subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    url VARCHAR(500) NOT NULL,
    events TEXT[] NOT NULL, -- Array of event types to subscribe to
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_triggered TIMESTAMP,
    failure_count INTEGER DEFAULT 0,
    
    -- Additional metadata
    description TEXT,
    headers JSONB, -- Custom headers to include in webhook requests
    retry_policy JSONB, -- Custom retry configuration
    
    -- Constraints
    CONSTRAINT valid_url CHECK (url ~ '^https?://'),
    CONSTRAINT valid_events CHECK (array_length(events, 1) > 0)
);

-- Webhook delivery logs for audit and debugging
CREATE TABLE IF NOT EXISTS webhook_delivery_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    webhook_id UUID REFERENCES webhook_subscriptions(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB NOT NULL,
    delivered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    response_status INTEGER,
    response_time_ms INTEGER,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0
);

-- Index for faster webhook lookups by event type
CREATE INDEX IF NOT EXISTS idx_webhook_events ON webhook_subscriptions USING GIN(events);
CREATE INDEX IF NOT EXISTS idx_webhook_active ON webhook_subscriptions(active) WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_webhook_logs_delivered ON webhook_delivery_logs(delivered_at DESC);

-- Function to trigger webhooks when APIs are modified
CREATE OR REPLACE FUNCTION trigger_api_webhooks()
RETURNS TRIGGER AS $$
DECLARE
    event_type TEXT;
    event_data JSONB;
BEGIN
    -- Determine event type based on operation
    IF TG_OP = 'INSERT' THEN
        event_type := 'api.created';
        event_data := to_jsonb(NEW);
    ELSIF TG_OP = 'UPDATE' THEN
        -- Check for specific update types
        IF OLD.status != NEW.status AND NEW.status = 'deprecated' THEN
            event_type := 'api.deprecated';
        ELSE
            event_type := 'api.updated';
        END IF;
        event_data := jsonb_build_object(
            'old', to_jsonb(OLD),
            'new', to_jsonb(NEW),
            'changed_fields', (
                SELECT jsonb_object_agg(key, value)
                FROM jsonb_each(to_jsonb(NEW))
                WHERE to_jsonb(OLD) -> key IS DISTINCT FROM value
            )
        );
    ELSIF TG_OP = 'DELETE' THEN
        event_type := 'api.deleted';
        event_data := to_jsonb(OLD);
    END IF;
    
    -- Insert into a notifications queue (to be processed by the application)
    INSERT INTO webhook_notifications_queue (event_type, event_data)
    VALUES (event_type, event_data);
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create notification queue table
CREATE TABLE IF NOT EXISTS webhook_notifications_queue (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed BOOLEAN DEFAULT false,
    processed_at TIMESTAMP
);

-- Create triggers for API events (commented out by default to avoid automatic triggers)
-- Uncomment these to enable automatic webhook triggers on database changes
-- CREATE TRIGGER api_webhook_trigger
-- AFTER INSERT OR UPDATE OR DELETE ON apis
-- FOR EACH ROW EXECUTE FUNCTION trigger_api_webhooks();

-- Sample webhook event types documentation
COMMENT ON TABLE webhook_subscriptions IS 'Stores webhook subscriptions for API update notifications. 
Supported event types:
- api.created: New API added to the library
- api.updated: API information updated
- api.deleted: API removed from the library
- api.deprecated: API marked as deprecated
- api.configured: API credentials configured
- note.added: Note or gotcha added to an API
- version.added: New version tracked for an API
- price.updated: Pricing information updated';

-- Add sample webhook for testing (disabled by default)
-- INSERT INTO webhook_subscriptions (url, events, description)
-- VALUES (
--     'https://webhook.site/unique-url',
--     ARRAY['api.created', 'api.updated', 'api.deprecated'],
--     'Test webhook for API updates'
-- );
-- Migration: Add API health monitoring and uptime tracking
-- This adds tables and functionality for monitoring API availability and performance

-- Health check results table
CREATE TABLE IF NOT EXISTS api_health_checks (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    check_time TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    response_time_ms INTEGER,
    status_code INTEGER,
    healthy BOOLEAN NOT NULL,
    error_message TEXT,
    
    -- Additional details
    endpoint_checked VARCHAR(500), -- Specific endpoint if not base URL
    headers_sent JSONB, -- Headers used in the check
    response_headers JSONB, -- Headers received in response
    
    -- Indexes for performance
    CONSTRAINT valid_response_time CHECK (response_time_ms >= 0)
);

-- Aggregated health metrics table
CREATE TABLE IF NOT EXISTS api_health_metrics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID UNIQUE NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    uptime_percentage NUMERIC(5,2),
    avg_response_time_ms INTEGER,
    total_checks INTEGER DEFAULT 0,
    successful_checks INTEGER DEFAULT 0,
    consecutive_failures INTEGER DEFAULT 0,
    current_status VARCHAR(20) DEFAULT 'unknown' CHECK (current_status IN ('healthy', 'degraded', 'unhealthy', 'unknown')),
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Time-based metrics
    last_24h_uptime NUMERIC(5,2),
    last_7d_uptime NUMERIC(5,2),
    last_30d_uptime NUMERIC(5,2),
    
    -- Performance metrics
    p50_response_time_ms INTEGER, -- Median
    p95_response_time_ms INTEGER, -- 95th percentile
    p99_response_time_ms INTEGER  -- 99th percentile
);

-- Scheduled health check configurations
CREATE TABLE IF NOT EXISTS health_check_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID UNIQUE NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    enabled BOOLEAN DEFAULT true,
    check_interval_minutes INTEGER DEFAULT 5,
    timeout_seconds INTEGER DEFAULT 10,
    retry_count INTEGER DEFAULT 2,
    custom_endpoint VARCHAR(500), -- Use this instead of base_url if specified
    custom_headers JSONB, -- Additional headers to send
    expected_status_codes INTEGER[], -- List of acceptable status codes
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Outage tracking table
CREATE TABLE IF NOT EXISTS api_outages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    started_at TIMESTAMP NOT NULL,
    ended_at TIMESTAMP,
    duration_minutes INTEGER,
    consecutive_failures INTEGER,
    error_summary TEXT,
    resolution_notes TEXT,
    severity VARCHAR(20) CHECK (severity IN ('minor', 'major', 'critical')),
    
    -- Calculate severity based on duration
    CONSTRAINT calculate_severity CHECK (
        severity = CASE 
            WHEN duration_minutes < 5 THEN 'minor'
            WHEN duration_minutes < 60 THEN 'major'
            ELSE 'critical'
        END OR duration_minutes IS NULL
    )
);

-- Create indexes for faster queries
CREATE INDEX IF NOT EXISTS idx_health_checks_api_time ON api_health_checks(api_id, check_time DESC);
CREATE INDEX IF NOT EXISTS idx_health_checks_healthy ON api_health_checks(api_id, healthy);
CREATE INDEX IF NOT EXISTS idx_health_metrics_status ON api_health_metrics(current_status);
CREATE INDEX IF NOT EXISTS idx_outages_api ON api_outages(api_id, started_at DESC);

-- Function to calculate uptime percentages
CREATE OR REPLACE FUNCTION calculate_uptime_percentage(
    p_api_id UUID,
    p_hours INTEGER DEFAULT 24
) RETURNS NUMERIC AS $$
DECLARE
    v_total_checks INTEGER;
    v_successful_checks INTEGER;
    v_uptime_percentage NUMERIC(5,2);
BEGIN
    SELECT 
        COUNT(*),
        COUNT(*) FILTER (WHERE healthy = true)
    INTO v_total_checks, v_successful_checks
    FROM api_health_checks
    WHERE api_id = p_api_id
    AND check_time > NOW() - (p_hours || ' hours')::INTERVAL;
    
    IF v_total_checks > 0 THEN
        v_uptime_percentage := (v_successful_checks::NUMERIC / v_total_checks) * 100;
    ELSE
        v_uptime_percentage := NULL;
    END IF;
    
    RETURN v_uptime_percentage;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update metrics after each health check
CREATE OR REPLACE FUNCTION update_health_metrics_trigger()
RETURNS TRIGGER AS $$
BEGIN
    -- Update or insert health metrics
    INSERT INTO api_health_metrics (
        api_id,
        uptime_percentage,
        last_24h_uptime,
        last_7d_uptime,
        last_30d_uptime,
        total_checks,
        successful_checks,
        consecutive_failures,
        current_status,
        last_updated
    )
    VALUES (
        NEW.api_id,
        calculate_uptime_percentage(NEW.api_id, 24*7), -- 7 day average
        calculate_uptime_percentage(NEW.api_id, 24),
        calculate_uptime_percentage(NEW.api_id, 24*7),
        calculate_uptime_percentage(NEW.api_id, 24*30),
        1,
        CASE WHEN NEW.healthy THEN 1 ELSE 0 END,
        CASE WHEN NEW.healthy THEN 0 ELSE 1 END,
        CASE WHEN NEW.healthy THEN 'healthy' ELSE 'degraded' END,
        NOW()
    )
    ON CONFLICT (api_id) DO UPDATE SET
        uptime_percentage = calculate_uptime_percentage(NEW.api_id, 24*7),
        last_24h_uptime = calculate_uptime_percentage(NEW.api_id, 24),
        last_7d_uptime = calculate_uptime_percentage(NEW.api_id, 24*7),
        last_30d_uptime = calculate_uptime_percentage(NEW.api_id, 24*30),
        total_checks = api_health_metrics.total_checks + 1,
        successful_checks = api_health_metrics.successful_checks + CASE WHEN NEW.healthy THEN 1 ELSE 0 END,
        consecutive_failures = CASE 
            WHEN NEW.healthy THEN 0 
            ELSE api_health_metrics.consecutive_failures + 1 
        END,
        current_status = CASE
            WHEN NEW.healthy THEN 'healthy'
            WHEN api_health_metrics.consecutive_failures >= 2 THEN 'unhealthy'
            ELSE 'degraded'
        END,
        last_updated = NOW();
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create the trigger
DROP TRIGGER IF EXISTS update_health_metrics ON api_health_checks;
CREATE TRIGGER update_health_metrics
AFTER INSERT ON api_health_checks
FOR EACH ROW EXECUTE FUNCTION update_health_metrics_trigger();

-- Sample health check configurations for testing
-- INSERT INTO health_check_configs (api_id, check_interval_minutes, custom_headers)
-- SELECT id, 5, '{"User-Agent": "API-Library-Health-Monitor/1.0"}'::jsonb
-- FROM apis
-- WHERE status = 'active' AND base_url IS NOT NULL
-- LIMIT 3;

-- View for current API health status
CREATE OR REPLACE VIEW api_health_status AS
SELECT 
    a.id,
    a.name,
    a.provider,
    a.category,
    m.current_status,
    m.uptime_percentage,
    m.avg_response_time_ms,
    m.consecutive_failures,
    m.last_updated,
    CASE 
        WHEN m.consecutive_failures >= 3 THEN 'Alert: API is down'
        WHEN m.consecutive_failures >= 1 THEN 'Warning: API may be experiencing issues'
        ELSE 'OK'
    END AS alert_status
FROM apis a
LEFT JOIN api_health_metrics m ON a.id = m.api_id
WHERE a.status IN ('active', 'beta');

COMMENT ON TABLE api_health_checks IS 'Stores individual health check results for APIs';
COMMENT ON TABLE api_health_metrics IS 'Aggregated health metrics and uptime statistics for APIs';
COMMENT ON TABLE health_check_configs IS 'Configuration for automated health monitoring of APIs';
COMMENT ON TABLE api_outages IS 'Tracks API outages and downtime incidents';
-- API Library Database Schema
-- Stores metadata about external APIs, their capabilities, pricing, and integration patterns

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Main APIs table
CREATE TABLE IF NOT EXISTS apis (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    provider VARCHAR(255) NOT NULL,
    description TEXT,
    base_url VARCHAR(500),
    documentation_url VARCHAR(500),
    pricing_url VARCHAR(500),
    category VARCHAR(100),
    status VARCHAR(50) DEFAULT 'active' CHECK (status IN ('active', 'deprecated', 'sunset', 'beta')),
    sunset_date DATE,
    auth_type VARCHAR(50) CHECK (auth_type IN ('api_key', 'oauth2', 'basic', 'bearer', 'none', 'custom')),
    
    -- Metadata tracking
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_refreshed TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    source_url VARCHAR(500),
    
    -- Additional metadata
    tags TEXT[], -- Array of tags for categorization
    capabilities TEXT[], -- Array of capability keywords
    
    -- Search optimization (added as separate column, will be populated via trigger)
    search_vector tsvector
);

-- Endpoints table
CREATE TABLE IF NOT EXISTS endpoints (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    path VARCHAR(500) NOT NULL,
    method VARCHAR(10) NOT NULL CHECK (method IN ('GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS')),
    description TEXT,
    
    -- Rate limiting information
    rate_limit_requests INTEGER,
    rate_limit_period VARCHAR(50), -- e.g., "minute", "hour", "day"
    
    -- Request/Response schemas (stored as JSONB)
    request_schema JSONB,
    response_schema JSONB,
    
    -- Authentication requirements
    requires_auth BOOLEAN DEFAULT true,
    auth_details JSONB, -- Additional auth requirements specific to endpoint
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(api_id, path, method)
);

-- Pricing tiers table
CREATE TABLE IF NOT EXISTS pricing_tiers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    
    -- Pricing models
    price_per_request DECIMAL(10, 6),
    price_per_mb DECIMAL(10, 6),
    price_per_minute DECIMAL(10, 6),
    monthly_cost DECIMAL(10, 2),
    annual_cost DECIMAL(10, 2),
    
    -- Limits
    free_tier_requests INTEGER,
    free_tier_mb INTEGER,
    max_requests_per_month INTEGER,
    
    -- Additional pricing details as JSONB for flexibility
    pricing_details JSONB,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(api_id, name)
);

-- Notes and gotchas table
CREATE TABLE IF NOT EXISTS notes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    content TEXT NOT NULL,
    type VARCHAR(50) NOT NULL CHECK (type IN ('gotcha', 'tip', 'warning', 'example', 'success', 'failure')),
    
    -- Optional reference to specific endpoint
    endpoint_id UUID REFERENCES endpoints(id) ON DELETE CASCADE,
    
    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255) DEFAULT 'system',
    scenario_source VARCHAR(255), -- Which scenario discovered this
    
    -- Voting system for usefulness
    helpful_count INTEGER DEFAULT 0,
    not_helpful_count INTEGER DEFAULT 0
);

-- API credentials configuration tracking
CREATE TABLE IF NOT EXISTS api_credentials (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    
    -- Configuration status
    is_configured BOOLEAN DEFAULT false,
    configuration_date TIMESTAMP,
    last_verified TIMESTAMP,
    
    -- Environment where configured
    environment VARCHAR(50) DEFAULT 'development' CHECK (environment IN ('development', 'staging', 'production', 'all')),
    
    -- Usage tracking
    last_used TIMESTAMP,
    usage_count INTEGER DEFAULT 0,
    
    -- Notes about configuration (not the actual credentials!)
    configuration_notes TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(api_id, environment)
);

-- API alternatives and relationships
CREATE TABLE IF NOT EXISTS api_relationships (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    related_api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    relationship_type VARCHAR(50) NOT NULL CHECK (relationship_type IN ('alternative', 'complement', 'upgrade', 'migration_target', 'deprecated_by')),
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    CHECK (api_id != related_api_id),
    UNIQUE(api_id, related_api_id, relationship_type)
);

-- Research requests tracking
CREATE TABLE IF NOT EXISTS research_requests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    capability VARCHAR(500) NOT NULL,
    requirements JSONB,
    status VARCHAR(50) DEFAULT 'queued' CHECK (status IN ('queued', 'in_progress', 'completed', 'failed')),
    
    -- Results
    apis_discovered INTEGER DEFAULT 0,
    completion_time TIMESTAMP,
    error_message TEXT,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Usage analytics
CREATE TABLE IF NOT EXISTS api_usage_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID REFERENCES apis(id) ON DELETE SET NULL,
    action VARCHAR(50) NOT NULL CHECK (action IN ('search', 'view', 'configure', 'use', 'note_added')),
    scenario_name VARCHAR(255),
    search_query TEXT,
    result_count INTEGER,
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_apis_search_vector ON apis USING GIN(search_vector);
CREATE INDEX IF NOT EXISTS idx_apis_status ON apis(status);
CREATE INDEX IF NOT EXISTS idx_apis_category ON apis(category);
CREATE INDEX IF NOT EXISTS idx_apis_provider ON apis(provider);
CREATE INDEX IF NOT EXISTS idx_apis_tags ON apis USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_apis_capabilities ON apis USING GIN(capabilities);

CREATE INDEX IF NOT EXISTS idx_endpoints_api_id ON endpoints(api_id);
CREATE INDEX IF NOT EXISTS idx_pricing_api_id ON pricing_tiers(api_id);
CREATE INDEX IF NOT EXISTS idx_notes_api_id ON notes(api_id);
CREATE INDEX IF NOT EXISTS idx_notes_type ON notes(type);
CREATE INDEX IF NOT EXISTS idx_credentials_api_id ON api_credentials(api_id);
CREATE INDEX IF NOT EXISTS idx_credentials_configured ON api_credentials(is_configured);

CREATE INDEX IF NOT EXISTS idx_usage_logs_api_id ON api_usage_logs(api_id);
CREATE INDEX IF NOT EXISTS idx_usage_logs_action ON api_usage_logs(action);
CREATE INDEX IF NOT EXISTS idx_usage_logs_created ON api_usage_logs(created_at);

-- Create update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create search vector update trigger
CREATE OR REPLACE FUNCTION update_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector := 
        setweight(to_tsvector('english', coalesce(NEW.name, '')), 'A') ||
        setweight(to_tsvector('english', coalesce(NEW.provider, '')), 'B') ||
        setweight(to_tsvector('english', coalesce(NEW.description, '')), 'C') ||
        setweight(to_tsvector('english', coalesce(array_to_string(NEW.tags, ' '), '')), 'D');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS update_apis_updated_at ON apis;
CREATE TRIGGER update_apis_updated_at BEFORE UPDATE ON apis
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_apis_search_vector ON apis;
CREATE TRIGGER update_apis_search_vector BEFORE INSERT OR UPDATE ON apis
    FOR EACH ROW EXECUTE FUNCTION update_search_vector();

DROP TRIGGER IF EXISTS update_endpoints_updated_at ON endpoints;
CREATE TRIGGER update_endpoints_updated_at BEFORE UPDATE ON endpoints
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_pricing_updated_at ON pricing_tiers;
CREATE TRIGGER update_pricing_updated_at BEFORE UPDATE ON pricing_tiers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_credentials_updated_at ON api_credentials;
CREATE TRIGGER update_credentials_updated_at BEFORE UPDATE ON api_credentials
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_research_updated_at ON research_requests;
CREATE TRIGGER update_research_updated_at BEFORE UPDATE ON research_requests
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Views for common queries
CREATE OR REPLACE VIEW configured_apis AS
SELECT 
    a.*,
    c.is_configured,
    c.configuration_date,
    c.last_verified,
    c.environment
FROM apis a
JOIN api_credentials c ON a.id = c.api_id
WHERE c.is_configured = true;

CREATE OR REPLACE VIEW api_overview AS
SELECT 
    a.id,
    a.name,
    a.provider,
    a.category,
    a.status,
    COUNT(DISTINCT e.id) as endpoint_count,
    COUNT(DISTINCT p.id) as pricing_tier_count,
    COUNT(DISTINCT n.id) as note_count,
    BOOL_OR(c.is_configured) as has_credentials
FROM apis a
LEFT JOIN endpoints e ON a.id = e.api_id
LEFT JOIN pricing_tiers p ON a.id = p.api_id
LEFT JOIN notes n ON a.id = n.api_id
LEFT JOIN api_credentials c ON a.id = c.api_id
GROUP BY a.id, a.name, a.provider, a.category, a.status;
-- Integration Snippets Schema Extension
-- Adds support for storing integration recipes and code snippets for APIs

-- Integration snippets/recipes table
CREATE TABLE IF NOT EXISTS integration_snippets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    
    -- Snippet metadata
    title VARCHAR(255) NOT NULL,
    description TEXT,
    language VARCHAR(50) NOT NULL CHECK (language IN ('javascript', 'typescript', 'python', 'go', 'java', 'ruby', 'php', 'csharp', 'rust', 'bash', 'curl', 'http')),
    framework VARCHAR(100), -- e.g., 'express', 'django', 'spring', 'rails'
    
    -- The actual code snippet
    code TEXT NOT NULL,
    
    -- Additional context
    dependencies JSONB, -- Required packages/libraries
    environment_variables TEXT[], -- Required env vars (names only, not values)
    prerequisites TEXT, -- Setup requirements or prerequisites
    
    -- Categorization
    snippet_type VARCHAR(50) NOT NULL CHECK (snippet_type IN ('authentication', 'basic_request', 'pagination', 'error_handling', 'webhook', 'batch_processing', 'rate_limiting', 'complete_integration')),
    tags TEXT[],
    
    -- Quality indicators
    tested BOOLEAN DEFAULT false,
    official BOOLEAN DEFAULT false, -- From official documentation
    community_verified BOOLEAN DEFAULT false,
    usage_count INTEGER DEFAULT 0,
    
    -- Voting system
    helpful_count INTEGER DEFAULT 0,
    not_helpful_count INTEGER DEFAULT 0,
    
    -- Optional reference to specific endpoint
    endpoint_id UUID REFERENCES endpoints(id) ON DELETE CASCADE,
    
    -- Metadata
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255) DEFAULT 'system',
    source_url VARCHAR(500), -- Link to original source if applicable
    
    -- Version tracking
    version VARCHAR(50), -- API version this snippet works with
    last_verified TIMESTAMP -- When the snippet was last verified to work
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_snippets_api_id ON integration_snippets(api_id);
CREATE INDEX IF NOT EXISTS idx_snippets_language ON integration_snippets(language);
CREATE INDEX IF NOT EXISTS idx_snippets_type ON integration_snippets(snippet_type);
CREATE INDEX IF NOT EXISTS idx_snippets_tags ON integration_snippets USING GIN(tags);
CREATE INDEX IF NOT EXISTS idx_snippets_official ON integration_snippets(official);
CREATE INDEX IF NOT EXISTS idx_snippets_tested ON integration_snippets(tested);

-- Add trigger for updated_at
DROP TRIGGER IF EXISTS update_snippets_updated_at ON integration_snippets;
CREATE TRIGGER update_snippets_updated_at BEFORE UPDATE ON integration_snippets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- View for popular snippets
CREATE OR REPLACE VIEW popular_snippets AS
SELECT 
    s.id,
    s.title,
    s.description,
    s.language,
    s.snippet_type,
    a.name as api_name,
    a.provider,
    s.helpful_count,
    s.usage_count,
    s.official,
    s.tested
FROM integration_snippets s
JOIN apis a ON s.api_id = a.id
WHERE s.helpful_count > s.not_helpful_count
ORDER BY s.usage_count DESC, s.helpful_count DESC;

-- View for verified snippets
CREATE OR REPLACE VIEW verified_snippets AS
SELECT 
    s.*,
    a.name as api_name,
    a.provider
FROM integration_snippets s
JOIN apis a ON s.api_id = a.id
WHERE s.tested = true 
   OR s.official = true 
   OR s.community_verified = true;
-- Schema updates for API Library v2 features
-- Adds support for version tracking, enhanced pricing, and cost calculation

-- Add version column to apis table if not exists


-- Create api_versions table for version history tracking
CREATE TABLE IF NOT EXISTS api_versions (
    id UUID PRIMARY KEY,
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    version VARCHAR(50) NOT NULL,
    change_summary TEXT,
    breaking_changes BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_by VARCHAR(255) DEFAULT 'system',
    
    INDEX idx_api_versions_api_id (api_id),
    INDEX idx_api_versions_created_at (created_at)
);

-- Add updated_at column to pricing_tiers for refresh tracking


-- Create cost_calculations table to store cost analysis history
CREATE TABLE IF NOT EXISTS cost_calculations (
    id UUID PRIMARY KEY,
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    requests_per_month INTEGER NOT NULL,
    data_per_request_mb DECIMAL(10,3),
    recommended_tier VARCHAR(255),
    estimated_cost DECIMAL(12,2),
    cost_breakdown JSONB,
    savings_tip TEXT,
    alternatives JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    INDEX idx_cost_calculations_api_id (api_id),
    INDEX idx_cost_calculations_created_at (created_at)
);

-- Create pricing_refresh_log table for tracking automatic refresh
CREATE TABLE IF NOT EXISTS pricing_refresh_log (
    id UUID PRIMARY KEY,
    api_id UUID NOT NULL REFERENCES apis(id) ON DELETE CASCADE,
    pricing_url TEXT,
    refresh_status VARCHAR(50), -- success, failed, partial
    items_updated INTEGER DEFAULT 0,
    error_message TEXT,
    refreshed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    INDEX idx_pricing_refresh_api_id (api_id),
    INDEX idx_pricing_refresh_date (refreshed_at)
);

-- Add indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_apis_version ON apis(version);
CREATE INDEX IF NOT EXISTS idx_apis_last_refreshed ON apis(last_refreshed);
CREATE INDEX IF NOT EXISTS idx_pricing_tiers_updated ON pricing_tiers(updated_at);

-- Function to track API version changes
CREATE OR REPLACE FUNCTION track_api_version_change()
RETURNS TRIGGER AS $$
BEGIN
    IF OLD.version IS DISTINCT FROM NEW.version THEN
        INSERT INTO api_versions (
            id, 
            api_id, 
            version, 
            change_summary,
            breaking_changes,
            created_at
        ) VALUES (
            gen_random_uuid(),
            NEW.id,
            NEW.version,
            'Version updated from ' || COALESCE(OLD.version, 'null') || ' to ' || NEW.version,
            FALSE,
            NOW()
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger for automatic version tracking
DROP TRIGGER IF EXISTS api_version_change_trigger ON apis;
CREATE TRIGGER api_version_change_trigger
    AFTER UPDATE ON apis
    FOR EACH ROW
    EXECUTE FUNCTION track_api_version_change();

-- Function to calculate estimated costs
CREATE OR REPLACE FUNCTION calculate_api_cost(
    p_api_id UUID,
    p_requests_per_month INTEGER,
    p_data_per_request_mb DECIMAL
)
RETURNS TABLE (
    tier_name VARCHAR(255),
    estimated_cost DECIMAL(12,2),
    base_cost DECIMAL(12,2),
    request_cost DECIMAL(12,2),
    data_cost DECIMAL(12,2)
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        pt.name AS tier_name,
        (pt.monthly_cost + 
         GREATEST(0, p_requests_per_month - pt.free_tier_requests) * pt.price_per_request +
         p_requests_per_month * p_data_per_request_mb * pt.price_per_mb) AS estimated_cost,
        pt.monthly_cost AS base_cost,
        GREATEST(0, p_requests_per_month - pt.free_tier_requests) * pt.price_per_request AS request_cost,
        p_requests_per_month * p_data_per_request_mb * pt.price_per_mb AS data_cost
    FROM pricing_tiers pt
    WHERE pt.api_id = p_api_id
    ORDER BY estimated_cost ASC
    LIMIT 1;
END;
$$ LANGUAGE plpgsql;

-- Add sample data for testing (optional)
-- This can be commented out in production
INSERT INTO api_versions (id, api_id, version, change_summary, breaking_changes, created_at)
SELECT 
    gen_random_uuid(),
    id,
    '1.0.0',
    'Initial version',
    false,
    NOW() - INTERVAL '30 days'
FROM apis
WHERE NOT EXISTS (
    SELECT 1 FROM api_versions WHERE api_id = apis.id
)
LIMIT 3;
