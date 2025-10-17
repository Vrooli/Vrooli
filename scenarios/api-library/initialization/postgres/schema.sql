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
CREATE INDEX idx_apis_search_vector ON apis USING GIN(search_vector);
CREATE INDEX idx_apis_status ON apis(status);
CREATE INDEX idx_apis_category ON apis(category);
CREATE INDEX idx_apis_provider ON apis(provider);
CREATE INDEX idx_apis_tags ON apis USING GIN(tags);
CREATE INDEX idx_apis_capabilities ON apis USING GIN(capabilities);

CREATE INDEX idx_endpoints_api_id ON endpoints(api_id);
CREATE INDEX idx_pricing_api_id ON pricing_tiers(api_id);
CREATE INDEX idx_notes_api_id ON notes(api_id);
CREATE INDEX idx_notes_type ON notes(type);
CREATE INDEX idx_credentials_api_id ON api_credentials(api_id);
CREATE INDEX idx_credentials_configured ON api_credentials(is_configured);

CREATE INDEX idx_usage_logs_api_id ON api_usage_logs(api_id);
CREATE INDEX idx_usage_logs_action ON api_usage_logs(action);
CREATE INDEX idx_usage_logs_created ON api_usage_logs(created_at);

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

CREATE TRIGGER update_apis_updated_at BEFORE UPDATE ON apis
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_apis_search_vector BEFORE INSERT OR UPDATE ON apis
    FOR EACH ROW EXECUTE FUNCTION update_search_vector();

CREATE TRIGGER update_endpoints_updated_at BEFORE UPDATE ON endpoints
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_pricing_updated_at BEFORE UPDATE ON pricing_tiers
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_credentials_updated_at BEFORE UPDATE ON api_credentials
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

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