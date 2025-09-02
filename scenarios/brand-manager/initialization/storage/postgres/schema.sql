-- Brand Manager Database Schema

-- Use shared vrooli database for all scenarios
CREATE DATABASE IF NOT EXISTS vrooli;
\c vrooli;

-- Brands table
CREATE TABLE brands (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    short_name VARCHAR(50),
    slogan TEXT,
    ad_copy TEXT,
    description TEXT,
    brand_colors JSONB DEFAULT '{}',
    logo_url TEXT,
    favicon_url TEXT,
    assets JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Campaigns table for organizing brands
CREATE TABLE campaigns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    brand_ids UUID[] DEFAULT '{}',
    status VARCHAR(50) DEFAULT 'draft',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Brand exports tracking
CREATE TABLE exports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id UUID REFERENCES brands(id),
    export_type VARCHAR(50), -- 'package', 'integration', 'api'
    target_app VARCHAR(255),
    export_path TEXT,
    manifest JSONB,
    status VARCHAR(50) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- Brand templates for quick generation
CREATE TABLE templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    category VARCHAR(100),
    style_config JSONB,
    color_schemes JSONB,
    font_families JSONB,
    prompt_templates JSONB,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Integration requests for Claude Code
CREATE TABLE integration_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    brand_id UUID REFERENCES brands(id),
    target_app_path TEXT NOT NULL,
    integration_type VARCHAR(50), -- 'full', 'partial', 'theme-only'
    claude_session_id TEXT,
    status VARCHAR(50) DEFAULT 'pending',
    request_payload JSONB,
    response_payload JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_brands_name ON brands(name);
CREATE INDEX idx_brands_created ON brands(created_at DESC);
CREATE INDEX idx_exports_brand ON exports(brand_id);
CREATE INDEX idx_exports_status ON exports(status);
CREATE INDEX idx_integration_status ON integration_requests(status);

-- Update trigger for updated_at
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER brands_updated_at BEFORE UPDATE ON brands
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER campaigns_updated_at BEFORE UPDATE ON campaigns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();