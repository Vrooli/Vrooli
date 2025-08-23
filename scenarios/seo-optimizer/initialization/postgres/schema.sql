-- SEO Optimizer Database Schema
-- Core tables for SEO analysis and optimization

-- Websites/Projects being optimized
CREATE TABLE IF NOT EXISTS websites (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    url VARCHAR(500) NOT NULL UNIQUE,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- SEO Audits for websites
CREATE TABLE IF NOT EXISTS audits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    website_id UUID REFERENCES websites(id) ON DELETE CASCADE,
    audit_type VARCHAR(50) NOT NULL, -- 'technical', 'content', 'performance', 'mobile'
    score INTEGER CHECK (score >= 0 AND score <= 100),
    issues JSONB DEFAULT '[]',
    recommendations JSONB DEFAULT '[]',
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Keywords tracking
CREATE TABLE IF NOT EXISTS keywords (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    website_id UUID REFERENCES websites(id) ON DELETE CASCADE,
    keyword VARCHAR(255) NOT NULL,
    search_volume INTEGER,
    difficulty INTEGER CHECK (difficulty >= 0 AND difficulty <= 100),
    opportunity_score DECIMAL(5,2),
    current_rank INTEGER,
    target_rank INTEGER,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(website_id, keyword)
);

-- Content optimization suggestions
CREATE TABLE IF NOT EXISTS content_optimizations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    website_id UUID REFERENCES websites(id) ON DELETE CASCADE,
    page_url VARCHAR(500) NOT NULL,
    title_suggestion TEXT,
    meta_description TEXT,
    heading_suggestions JSONB DEFAULT '[]',
    keyword_density JSONB DEFAULT '{}',
    readability_score DECIMAL(5,2),
    content_suggestions JSONB DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Competitor analysis
CREATE TABLE IF NOT EXISTS competitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    website_id UUID REFERENCES websites(id) ON DELETE CASCADE,
    competitor_url VARCHAR(500) NOT NULL,
    competitor_name VARCHAR(255),
    domain_authority INTEGER,
    organic_traffic_estimate INTEGER,
    top_keywords JSONB DEFAULT '[]',
    backlink_count INTEGER,
    analysis_data JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Ranking history
CREATE TABLE IF NOT EXISTS ranking_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    keyword_id UUID REFERENCES keywords(id) ON DELETE CASCADE,
    position INTEGER,
    search_engine VARCHAR(50) DEFAULT 'google',
    location VARCHAR(100),
    device_type VARCHAR(20) DEFAULT 'desktop',
    recorded_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Backlinks tracking
CREATE TABLE IF NOT EXISTS backlinks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    website_id UUID REFERENCES websites(id) ON DELETE CASCADE,
    source_url VARCHAR(500) NOT NULL,
    target_url VARCHAR(500) NOT NULL,
    anchor_text TEXT,
    domain_authority INTEGER,
    link_type VARCHAR(50), -- 'dofollow', 'nofollow', 'sponsored', 'ugc'
    first_seen TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_checked TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    is_active BOOLEAN DEFAULT true
);

-- Technical SEO issues
CREATE TABLE IF NOT EXISTS technical_issues (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    website_id UUID REFERENCES websites(id) ON DELETE CASCADE,
    page_url VARCHAR(500),
    issue_type VARCHAR(100) NOT NULL,
    severity VARCHAR(20) NOT NULL, -- 'critical', 'high', 'medium', 'low'
    description TEXT,
    recommendation TEXT,
    status VARCHAR(20) DEFAULT 'open', -- 'open', 'fixed', 'ignored'
    detected_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    resolved_at TIMESTAMP
);

-- Create indexes for better performance
CREATE INDEX idx_audits_website_id ON audits(website_id);
CREATE INDEX idx_audits_created_at ON audits(created_at DESC);
CREATE INDEX idx_keywords_website_id ON keywords(website_id);
CREATE INDEX idx_keywords_keyword ON keywords(keyword);
CREATE INDEX idx_ranking_history_keyword_id ON ranking_history(keyword_id);
CREATE INDEX idx_ranking_history_recorded_at ON ranking_history(recorded_at DESC);
CREATE INDEX idx_competitors_website_id ON competitors(website_id);
CREATE INDEX idx_technical_issues_website_id ON technical_issues(website_id);
CREATE INDEX idx_technical_issues_status ON technical_issues(status);

-- Add update trigger for updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_websites_updated_at BEFORE UPDATE ON websites
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_keywords_updated_at BEFORE UPDATE ON keywords
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_competitors_updated_at BEFORE UPDATE ON competitors
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();