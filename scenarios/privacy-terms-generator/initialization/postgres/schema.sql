-- Privacy & Terms Generator Database Schema
-- Version: 1.0.0
-- Description: Schema for storing legal templates, generated documents, and business profiles

-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS legal_generator;

-- Set search path
SET search_path TO legal_generator;

-- Business profiles table
CREATE TABLE IF NOT EXISTS business_profiles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL, -- saas, ecommerce, mobile, consulting, etc.
    jurisdictions TEXT[] NOT NULL, -- Array of jurisdiction codes
    website VARCHAR(500),
    email VARCHAR(255),
    data_collected JSONB DEFAULT '{}', -- Types of data collected
    services JSONB DEFAULT '{}', -- Services provided
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}'
);

-- Legal templates table
CREATE TABLE IF NOT EXISTS legal_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_type VARCHAR(50) NOT NULL, -- privacy, terms, cookie, eula, disclaimer
    jurisdiction VARCHAR(100) NOT NULL, -- US, EU, UK, CA, AU, etc.
    industry VARCHAR(100), -- Optional industry-specific template
    version VARCHAR(20) NOT NULL DEFAULT '1.0.0',
    source_url TEXT, -- Where template was sourced from
    fetched_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_validated TIMESTAMP WITH TIME ZONE,
    content TEXT NOT NULL, -- Template content with placeholders
    sections JSONB DEFAULT '[]', -- Structured sections for easier customization
    requirements JSONB DEFAULT '{}', -- Legal requirements this template addresses
    metadata JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    UNIQUE(template_type, jurisdiction, industry, version)
);

-- Generated documents table  
CREATE TABLE IF NOT EXISTS generated_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    business_id UUID REFERENCES business_profiles(id) ON DELETE CASCADE,
    template_id UUID REFERENCES legal_templates(id),
    document_type VARCHAR(50) NOT NULL,
    version INTEGER NOT NULL DEFAULT 1,
    generated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE, -- Optional expiration for review
    customizations JSONB DEFAULT '{}', -- Custom clauses and modifications
    content TEXT NOT NULL, -- Final generated content
    format VARCHAR(20) DEFAULT 'markdown', -- html, markdown, pdf
    status VARCHAR(50) DEFAULT 'draft', -- draft, active, archived
    metadata JSONB DEFAULT '{}',
    CONSTRAINT unique_active_document UNIQUE(business_id, document_type, version)
);

-- Document history table for tracking changes
CREATE TABLE IF NOT EXISTS document_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID REFERENCES generated_documents(id) ON DELETE CASCADE,
    changed_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    change_type VARCHAR(50) NOT NULL, -- created, updated, approved, archived
    changed_by VARCHAR(255), -- User or system that made the change
    change_summary TEXT,
    previous_content TEXT,
    new_content TEXT,
    metadata JSONB DEFAULT '{}'
);

-- Template clauses for reusable components
CREATE TABLE IF NOT EXISTS template_clauses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    clause_type VARCHAR(100) NOT NULL, -- data_collection, cookies, liability, etc.
    jurisdiction VARCHAR(100),
    content TEXT NOT NULL,
    tags TEXT[], -- For searching and categorization
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}'
);

-- Compliance requirements tracking
CREATE TABLE IF NOT EXISTS compliance_requirements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    jurisdiction VARCHAR(100) NOT NULL,
    requirement_type VARCHAR(100) NOT NULL, -- GDPR, CCPA, COPPA, etc.
    description TEXT NOT NULL,
    mandatory_clauses TEXT[], -- Required clause types
    effective_date DATE,
    source_url TEXT,
    last_checked TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}',
    UNIQUE(jurisdiction, requirement_type)
);

-- Template freshness tracking
CREATE TABLE IF NOT EXISTS template_freshness (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    template_id UUID REFERENCES legal_templates(id) ON DELETE CASCADE,
    checked_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    is_current BOOLEAN DEFAULT true,
    changes_detected TEXT,
    update_priority VARCHAR(20) DEFAULT 'low', -- low, medium, high, critical
    metadata JSONB DEFAULT '{}'
);

-- Indexes for performance
CREATE INDEX idx_business_profiles_type ON business_profiles(type);
CREATE INDEX idx_business_profiles_jurisdictions ON business_profiles USING GIN(jurisdictions);
CREATE INDEX idx_legal_templates_type_jurisdiction ON legal_templates(template_type, jurisdiction);
CREATE INDEX idx_legal_templates_active ON legal_templates(is_active);
CREATE INDEX idx_generated_documents_business ON generated_documents(business_id);
CREATE INDEX idx_generated_documents_status ON generated_documents(status);
CREATE INDEX idx_template_clauses_tags ON template_clauses USING GIN(tags);
CREATE INDEX idx_compliance_requirements_jurisdiction ON compliance_requirements(jurisdiction);

-- Functions and triggers
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply update trigger to relevant tables
CREATE TRIGGER update_business_profiles_updated_at
    BEFORE UPDATE ON business_profiles
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_template_clauses_updated_at
    BEFORE UPDATE ON template_clauses
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- View for latest active documents per business
CREATE OR REPLACE VIEW latest_documents AS
SELECT DISTINCT ON (business_id, document_type)
    gd.*,
    bp.name as business_name,
    lt.jurisdiction as template_jurisdiction
FROM generated_documents gd
JOIN business_profiles bp ON gd.business_id = bp.id
LEFT JOIN legal_templates lt ON gd.template_id = lt.id
WHERE gd.status = 'active'
ORDER BY business_id, document_type, version DESC;

-- View for template freshness status
CREATE OR REPLACE VIEW template_status AS
SELECT 
    lt.id,
    lt.template_type,
    lt.jurisdiction,
    lt.industry,
    lt.fetched_at,
    EXTRACT(DAY FROM CURRENT_TIMESTAMP - lt.fetched_at) as days_old,
    CASE 
        WHEN EXTRACT(DAY FROM CURRENT_TIMESTAMP - lt.fetched_at) > 30 THEN 'stale'
        WHEN EXTRACT(DAY FROM CURRENT_TIMESTAMP - lt.fetched_at) > 14 THEN 'aging'
        ELSE 'fresh'
    END as freshness_status,
    lt.is_active
FROM legal_templates lt;

-- Grant permissions (adjust as needed for your setup)
GRANT ALL ON SCHEMA legal_generator TO PUBLIC;
GRANT ALL ON ALL TABLES IN SCHEMA legal_generator TO PUBLIC;
GRANT ALL ON ALL SEQUENCES IN SCHEMA legal_generator TO PUBLIC;