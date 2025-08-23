-- Competitor Change Monitor Database Schema

-- Create database if not exists
CREATE DATABASE IF NOT EXISTS competitor_monitor;

-- Competitors table
CREATE TABLE IF NOT EXISTS competitors (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    importance VARCHAR(50) DEFAULT 'medium' CHECK (importance IN ('low', 'medium', 'high', 'critical')),
    is_active BOOLEAN DEFAULT true,
    metadata JSONB DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Monitoring targets table
CREATE TABLE IF NOT EXISTS monitoring_targets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    competitor_id UUID REFERENCES competitors(id) ON DELETE CASCADE,
    url TEXT NOT NULL,
    target_type VARCHAR(50) NOT NULL CHECK (target_type IN ('website', 'github', 'api', 'rss', 'social')),
    selector TEXT, -- CSS selector for specific content monitoring
    check_frequency INTEGER DEFAULT 3600, -- seconds between checks
    last_checked TIMESTAMP WITH TIME ZONE,
    last_content_hash VARCHAR(64),
    is_active BOOLEAN DEFAULT true,
    config JSONB DEFAULT '{}', -- target-specific configuration
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Change history table
CREATE TABLE IF NOT EXISTS change_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    target_id UUID REFERENCES monitoring_targets(id) ON DELETE CASCADE,
    change_type VARCHAR(50) NOT NULL CHECK (change_type IN ('content', 'structure', 'release', 'commit', 'announcement')),
    severity VARCHAR(50) DEFAULT 'low' CHECK (severity IN ('low', 'medium', 'high', 'critical')),
    title VARCHAR(500),
    description TEXT,
    old_content TEXT,
    new_content TEXT,
    diff_summary JSONB,
    ai_analysis TEXT,
    relevance_score FLOAT CHECK (relevance_score >= 0 AND relevance_score <= 1),
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    acknowledged BOOLEAN DEFAULT false,
    acknowledged_by VARCHAR(255),
    acknowledged_at TIMESTAMP WITH TIME ZONE
);

-- Alerts table (updated)
CREATE TABLE IF NOT EXISTS alerts (
    id VARCHAR(255) PRIMARY KEY,
    competitor_id VARCHAR(255),
    title VARCHAR(500),
    priority VARCHAR(50) CHECK (priority IN ('low', 'medium', 'high', 'critical')),
    url TEXT,
    category VARCHAR(100),
    summary TEXT,
    insights JSONB,
    actions JSONB,
    relevance_score INTEGER,
    status VARCHAR(50) DEFAULT 'unread' CHECK (status IN ('unread', 'read', 'acknowledged', 'dismissed')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Keywords and patterns to watch for
CREATE TABLE IF NOT EXISTS watch_keywords (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    competitor_id UUID REFERENCES competitors(id) ON DELETE CASCADE,
    keyword VARCHAR(255) NOT NULL,
    category VARCHAR(100),
    importance VARCHAR(50) DEFAULT 'medium' CHECK (importance IN ('low', 'medium', 'high', 'critical')),
    is_regex BOOLEAN DEFAULT false,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Competitive insights table
CREATE TABLE IF NOT EXISTS insights (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    competitor_id UUID REFERENCES competitors(id) ON DELETE CASCADE,
    insight_type VARCHAR(100),
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    confidence_score FLOAT CHECK (confidence_score >= 0 AND confidence_score <= 1),
    source_changes UUID[], -- Array of change_history IDs
    tags TEXT[],
    is_actionable BOOLEAN DEFAULT false,
    action_taken TEXT,
    generated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Monitoring sessions table (for tracking scraping runs)
CREATE TABLE IF NOT EXISTS monitoring_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_type VARCHAR(50) CHECK (session_type IN ('scheduled', 'manual', 'triggered')),
    targets_checked INTEGER DEFAULT 0,
    changes_detected INTEGER DEFAULT 0,
    alerts_sent INTEGER DEFAULT 0,
    errors INTEGER DEFAULT 0,
    duration_ms INTEGER,
    status VARCHAR(50) DEFAULT 'running' CHECK (status IN ('running', 'completed', 'failed')),
    started_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Change analyses table (for AI analysis results)
CREATE TABLE IF NOT EXISTS change_analyses (
    id VARCHAR(255) PRIMARY KEY,
    competitor_id VARCHAR(255),
    target_url TEXT,
    change_type VARCHAR(50),
    relevance_score INTEGER,
    change_category VARCHAR(100),
    impact_level VARCHAR(50),
    key_insights JSONB,
    recommended_actions JSONB,
    summary TEXT,
    old_content TEXT,
    new_content TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Scan history table
CREATE TABLE IF NOT EXISTS scan_history (
    id VARCHAR(255) PRIMARY KEY,
    scan_type VARCHAR(50),
    targets_checked INTEGER DEFAULT 0,
    changes_detected INTEGER DEFAULT 0,
    errors INTEGER DEFAULT 0,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE
);

-- Update monitoring_targets to add content field
ALTER TABLE monitoring_targets ADD COLUMN IF NOT EXISTS content TEXT;
ALTER TABLE monitoring_targets ADD COLUMN IF NOT EXISTS last_change_detected TIMESTAMP WITH TIME ZONE;

-- Create indexes for better performance
CREATE INDEX idx_targets_competitor ON monitoring_targets(competitor_id);
CREATE INDEX idx_targets_active ON monitoring_targets(is_active);
CREATE INDEX idx_targets_last_checked ON monitoring_targets(last_checked);
CREATE INDEX idx_changes_target ON change_history(target_id);
CREATE INDEX idx_changes_detected ON change_history(detected_at);
CREATE INDEX idx_changes_severity ON change_history(severity);
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_insights_competitor ON insights(competitor_id);
CREATE INDEX idx_keywords_competitor ON watch_keywords(competitor_id);

-- Create update trigger for updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_competitors_updated_at BEFORE UPDATE ON competitors
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_targets_updated_at BEFORE UPDATE ON monitoring_targets
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();