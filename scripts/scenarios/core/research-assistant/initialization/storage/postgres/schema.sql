-- Research Assistant Database Schema
-- Purpose: Store research reports, schedules, chat history, and knowledge management

-- Create schema for research assistant
CREATE SCHEMA IF NOT EXISTS research_assistant;

-- Set search path
SET search_path TO research_assistant;

-- ==========================================
-- Core Tables
-- ==========================================

-- Research Reports Table
CREATE TABLE IF NOT EXISTS reports (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(500) NOT NULL,
    topic TEXT NOT NULL,
    
    -- Report configuration
    depth VARCHAR(20) NOT NULL DEFAULT 'standard', -- quick, standard, deep
    target_length INTEGER DEFAULT 5, -- pages (1-10)
    language VARCHAR(10) DEFAULT 'en',
    
    -- Content
    markdown_content TEXT,
    summary TEXT,
    key_findings JSONB, -- Array of key findings with confidence scores
    
    -- Metadata
    sources_count INTEGER DEFAULT 0,
    word_count INTEGER DEFAULT 0,
    confidence_score DECIMAL(3,2), -- 0.00 to 1.00
    
    -- File references
    pdf_url TEXT, -- MinIO URL for PDF version
    assets_folder TEXT, -- MinIO folder for images/charts
    
    -- Timestamps
    requested_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    processing_time_seconds INTEGER,
    
    -- Status
    status VARCHAR(20) DEFAULT 'pending', -- pending, processing, completed, failed
    error_message TEXT,
    
    -- User context
    requested_by VARCHAR(255),
    organization VARCHAR(255),
    
    -- Scheduling reference
    schedule_id UUID,
    
    -- Vector embedding reference
    embedding_id VARCHAR(100), -- Qdrant point ID
    
    -- Search and filtering
    tags TEXT[],
    category VARCHAR(100),
    is_archived BOOLEAN DEFAULT FALSE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Report Sources Table (Citations)
CREATE TABLE IF NOT EXISTS report_sources (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    report_id UUID NOT NULL REFERENCES reports(id) ON DELETE CASCADE,
    
    -- Source information
    url TEXT NOT NULL,
    title TEXT,
    author TEXT,
    publication_date DATE,
    domain VARCHAR(255),
    
    -- Content
    extracted_text TEXT,
    relevant_quotes JSONB, -- Array of relevant quotes used in report
    
    -- Quality metrics
    credibility_score DECIMAL(3,2), -- 0.00 to 1.00
    relevance_score DECIMAL(3,2), -- 0.00 to 1.00
    
    -- Processing
    extraction_method VARCHAR(50), -- searxng, browserless, manual
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Cache reference
    cached_content_url TEXT, -- MinIO URL for cached version
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Report Schedules Table
CREATE TABLE IF NOT EXISTS report_schedules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Schedule configuration
    cron_expression VARCHAR(100) NOT NULL, -- e.g., "0 9 * * *" for daily at 9am
    timezone VARCHAR(50) DEFAULT 'UTC',
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Report configuration template
    topic_template TEXT NOT NULL, -- Can include variables like {{DATE}}, {{WEEK}}
    depth VARCHAR(20) DEFAULT 'standard',
    target_length INTEGER DEFAULT 5,
    
    -- Advanced options
    search_filters JSONB, -- Specific domains, date ranges, etc.
    custom_prompts JSONB, -- Additional instructions for report generation
    
    -- Delivery settings
    notification_emails TEXT[],
    notification_webhooks TEXT[],
    auto_publish BOOLEAN DEFAULT FALSE,
    
    -- Execution tracking
    last_run_at TIMESTAMP WITH TIME ZONE,
    next_run_at TIMESTAMP WITH TIME ZONE,
    total_runs INTEGER DEFAULT 0,
    successful_runs INTEGER DEFAULT 0,
    failed_runs INTEGER DEFAULT 0,
    
    -- User context
    created_by VARCHAR(255),
    organization VARCHAR(255),
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Chat Conversations Table
CREATE TABLE IF NOT EXISTS chat_conversations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255),
    
    -- Context
    user_id VARCHAR(255),
    organization VARCHAR(255),
    
    -- Conversation state
    is_active BOOLEAN DEFAULT TRUE,
    last_message_at TIMESTAMP WITH TIME ZONE,
    message_count INTEGER DEFAULT 0,
    
    -- Associated reports
    related_report_ids UUID[],
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Chat Messages Table
CREATE TABLE IF NOT EXISTS chat_messages (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES chat_conversations(id) ON DELETE CASCADE,
    
    -- Message content
    role VARCHAR(20) NOT NULL, -- user, assistant, system
    content TEXT NOT NULL,
    
    -- Context used for response
    context_sources JSONB, -- Report chunks and sources used
    confidence_score DECIMAL(3,2),
    
    -- Actions triggered
    triggered_report_id UUID REFERENCES reports(id),
    triggered_action VARCHAR(50), -- new_report, update_schedule, export_pdf, etc.
    
    -- Metadata
    tokens_used INTEGER,
    processing_time_ms INTEGER,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Report Templates Table
CREATE TABLE IF NOT EXISTS report_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL UNIQUE,
    description TEXT,
    
    -- Template configuration
    category VARCHAR(100),
    default_depth VARCHAR(20) DEFAULT 'standard',
    default_length INTEGER DEFAULT 5,
    
    -- Structure template
    sections JSONB NOT NULL, -- Array of section definitions
    
    -- Search configuration
    required_sources INTEGER DEFAULT 10,
    search_queries JSONB, -- Predefined search queries
    domain_whitelist TEXT[],
    domain_blacklist TEXT[],
    
    -- Prompts and instructions
    system_prompt TEXT,
    analysis_prompts JSONB,
    
    -- Usage tracking
    times_used INTEGER DEFAULT 0,
    last_used_at TIMESTAMP WITH TIME ZONE,
    
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Search History Table
CREATE TABLE IF NOT EXISTS search_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    query TEXT NOT NULL,
    
    -- Search configuration
    engines_used TEXT[],
    filters_applied JSONB,
    
    -- Results
    results_count INTEGER,
    results_summary JSONB,
    
    -- Usage context
    report_id UUID REFERENCES reports(id),
    user_id VARCHAR(255),
    
    -- Performance
    search_time_ms INTEGER,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- User Preferences Table
CREATE TABLE IF NOT EXISTS user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) UNIQUE NOT NULL,
    
    -- Display preferences
    theme VARCHAR(20) DEFAULT 'light',
    language VARCHAR(10) DEFAULT 'en',
    timezone VARCHAR(50) DEFAULT 'UTC',
    
    -- Report defaults
    default_depth VARCHAR(20) DEFAULT 'standard',
    default_length INTEGER DEFAULT 5,
    default_template_id UUID REFERENCES report_templates(id),
    
    -- Notification preferences
    email_notifications BOOLEAN DEFAULT TRUE,
    notification_frequency VARCHAR(20) DEFAULT 'immediate', -- immediate, daily, weekly
    
    -- Search preferences
    preferred_search_engines TEXT[],
    blocked_domains TEXT[],
    
    -- Chat preferences
    chat_personality VARCHAR(50) DEFAULT 'professional', -- professional, friendly, concise
    show_sources BOOLEAN DEFAULT TRUE,
    
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ==========================================
-- Indexes for Performance
-- ==========================================

-- Reports indexes
CREATE INDEX idx_reports_status ON reports(status);
CREATE INDEX idx_reports_created_at ON reports(created_at DESC);
CREATE INDEX idx_reports_schedule_id ON reports(schedule_id);
CREATE INDEX idx_reports_organization ON reports(organization);
CREATE INDEX idx_reports_tags ON reports USING GIN(tags);
CREATE INDEX idx_reports_category ON reports(category);
CREATE INDEX idx_reports_archived ON reports(is_archived);

-- Report sources indexes
CREATE INDEX idx_sources_report_id ON report_sources(report_id);
CREATE INDEX idx_sources_domain ON report_sources(domain);
CREATE INDEX idx_sources_credibility ON report_sources(credibility_score);

-- Schedules indexes
CREATE INDEX idx_schedules_active ON report_schedules(is_active);
CREATE INDEX idx_schedules_next_run ON report_schedules(next_run_at);
CREATE INDEX idx_schedules_organization ON report_schedules(organization);

-- Chat indexes
CREATE INDEX idx_conversations_user ON chat_conversations(user_id);
CREATE INDEX idx_conversations_active ON chat_conversations(is_active);
CREATE INDEX idx_messages_conversation ON chat_messages(conversation_id);
CREATE INDEX idx_messages_created ON chat_messages(created_at);

-- Search history indexes
CREATE INDEX idx_search_query ON search_history(query);
CREATE INDEX idx_search_report ON search_history(report_id);
CREATE INDEX idx_search_user ON search_history(user_id);

-- ==========================================
-- Functions and Triggers
-- ==========================================

-- Update timestamp trigger
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply update trigger to tables with updated_at
CREATE TRIGGER update_reports_timestamp BEFORE UPDATE ON reports
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_schedules_timestamp BEFORE UPDATE ON report_schedules
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_conversations_timestamp BEFORE UPDATE ON chat_conversations
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_templates_timestamp BEFORE UPDATE ON report_templates
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_preferences_timestamp BEFORE UPDATE ON user_preferences
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Function to calculate next run time for schedules
CREATE OR REPLACE FUNCTION calculate_next_run(cron_exp VARCHAR, tz VARCHAR)
RETURNS TIMESTAMP WITH TIME ZONE AS $$
BEGIN
    -- This would integrate with a cron parser
    -- For now, return a placeholder
    RETURN NOW() + INTERVAL '1 day';
END;
$$ LANGUAGE plpgsql;

-- ==========================================
-- Views for Common Queries
-- ==========================================

-- Active reports view
CREATE VIEW active_reports AS
SELECT 
    r.*,
    COUNT(rs.id) as source_count,
    AVG(rs.credibility_score) as avg_source_credibility
FROM reports r
LEFT JOIN report_sources rs ON rs.report_id = r.id
WHERE r.status = 'completed' 
    AND r.is_archived = FALSE
GROUP BY r.id
ORDER BY r.created_at DESC;

-- Schedule status view
CREATE VIEW schedule_status AS
SELECT 
    s.*,
    COUNT(r.id) as total_reports,
    MAX(r.created_at) as last_report_date
FROM report_schedules s
LEFT JOIN reports r ON r.schedule_id = s.id
WHERE s.is_active = TRUE
GROUP BY s.id
ORDER BY s.next_run_at ASC;

-- ==========================================
-- Initial Permissions
-- ==========================================

-- Grant usage on schema
GRANT USAGE ON SCHEMA research_assistant TO PUBLIC;

-- Grant appropriate permissions on tables
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA research_assistant TO PUBLIC;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA research_assistant TO PUBLIC;