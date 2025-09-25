-- Email Triage Database Schema
-- Multi-tenant email management and AI-powered triage system

-- Extensions for UUID generation and JSON operations
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users table (references scenario-authenticator users)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email_profiles JSONB DEFAULT '[]'::jsonb,
    preferences JSONB DEFAULT '{}'::jsonb,
    plan_type VARCHAR(50) DEFAULT 'free' CHECK (plan_type IN ('free', 'pro', 'business')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Indexes for performance
    CONSTRAINT valid_plan_type CHECK (plan_type IN ('free', 'pro', 'business'))
);

-- Email accounts table
CREATE TABLE IF NOT EXISTS email_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_address VARCHAR(255) NOT NULL,
    imap_settings JSONB NOT NULL, -- Encrypted IMAP configuration
    smtp_settings JSONB NOT NULL, -- Encrypted SMTP configuration
    last_sync TIMESTAMP WITH TIME ZONE,
    sync_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints and indexes
    UNIQUE(user_id, email_address) -- One account per email per user
);

-- Triage rules table for AI-generated and manual rules
CREATE TABLE IF NOT EXISTS triage_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    conditions JSONB NOT NULL, -- Array of rule conditions
    actions JSONB NOT NULL,    -- Array of actions to take
    priority INTEGER DEFAULT 100, -- Lower numbers = higher priority
    enabled BOOLEAN DEFAULT true,
    created_by_ai BOOLEAN DEFAULT false,
    ai_confidence DECIMAL(3,2) DEFAULT 0.0 CHECK (ai_confidence >= 0.0 AND ai_confidence <= 1.0),
    match_count INTEGER DEFAULT 0, -- Number of times rule has matched
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints and indexes
    CONSTRAINT valid_priority CHECK (priority >= 0 AND priority <= 1000)
);

-- Processed emails table
CREATE TABLE IF NOT EXISTS processed_emails (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id UUID NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    message_id VARCHAR(500) NOT NULL, -- Original email Message-ID header
    subject TEXT,
    sender_email VARCHAR(255),
    recipient_emails TEXT[], -- Array of recipient email addresses
    body_preview TEXT, -- First 500 characters of body
    full_body TEXT,    -- Complete email body content
    priority_score DECIMAL(3,2) DEFAULT 0.5 CHECK (priority_score >= 0.0 AND priority_score <= 1.0),
    vector_id UUID, -- Reference to Qdrant vector for semantic search
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    actions_taken JSONB DEFAULT '[]'::jsonb, -- Array of actions performed
    metadata JSONB DEFAULT '{}'::jsonb, -- Additional email metadata
    
    -- Constraints and indexes
    UNIQUE(account_id, message_id) -- Prevent duplicate processing
);

-- Rule execution log for performance analytics
CREATE TABLE IF NOT EXISTS rule_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rule_id UUID NOT NULL REFERENCES triage_rules(id) ON DELETE CASCADE,
    email_id UUID NOT NULL REFERENCES processed_emails(id) ON DELETE CASCADE,
    matched BOOLEAN NOT NULL,
    confidence_score DECIMAL(3,2),
    execution_time_ms INTEGER, -- Time taken to evaluate rule
    actions_executed JSONB DEFAULT '[]'::jsonb,
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Indexes for analytics
    dummy_field BOOLEAN DEFAULT NULL -- Removed inline index
);

-- User activity log for audit and analytics
CREATE TABLE IF NOT EXISTS activity_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    activity_type VARCHAR(50) NOT NULL, -- rule_created, email_processed, account_added, etc.
    description TEXT NOT NULL,
    details JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Indexes (created separately below)
    dummy_field BOOLEAN DEFAULT NULL
);

-- Usage tracking for subscription limits
CREATE TABLE IF NOT EXISTS usage_tracking (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tracking_month DATE NOT NULL, -- First day of the month
    emails_processed INTEGER DEFAULT 0,
    rules_executed INTEGER DEFAULT 0,
    api_calls INTEGER DEFAULT 0,
    search_queries INTEGER DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Constraints and indexes
    UNIQUE(user_id, tracking_month) -- One record per user per month
);

-- Functions and triggers for automatic timestamp updates
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at columns
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_email_accounts_updated_at BEFORE UPDATE ON email_accounts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_triage_rules_updated_at BEFORE UPDATE ON triage_rules FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_usage_tracking_updated_at BEFORE UPDATE ON usage_tracking FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to automatically create/update usage tracking
CREATE OR REPLACE FUNCTION upsert_usage_tracking(p_user_id UUID, p_emails INTEGER DEFAULT 0, p_rules INTEGER DEFAULT 0, p_api_calls INTEGER DEFAULT 0, p_searches INTEGER DEFAULT 0)
RETURNS VOID AS $$
BEGIN
    INSERT INTO usage_tracking (user_id, tracking_month, emails_processed, rules_executed, api_calls, search_queries)
    VALUES (p_user_id, DATE_TRUNC('month', NOW())::DATE, p_emails, p_rules, p_api_calls, p_searches)
    ON CONFLICT (user_id, tracking_month)
    DO UPDATE SET 
        emails_processed = usage_tracking.emails_processed + p_emails,
        rules_executed = usage_tracking.rules_executed + p_rules,
        api_calls = usage_tracking.api_calls + p_api_calls,
        search_queries = usage_tracking.search_queries + p_searches,
        updated_at = NOW();
END;
$$ language 'plpgsql';

-- Function to log user activity
CREATE OR REPLACE FUNCTION log_user_activity(p_user_id UUID, p_activity_type VARCHAR(50), p_description TEXT, p_details JSONB DEFAULT '{}'::jsonb)
RETURNS VOID AS $$
BEGIN
    INSERT INTO activity_logs (user_id, activity_type, description, details)
    VALUES (p_user_id, p_activity_type, p_description, p_details);
END;
$$ language 'plpgsql';

-- Indexes for common query patterns
CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_processed_emails_user_search 
    ON processed_emails(account_id, processed_at DESC, priority_score DESC);

CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_triage_rules_user_enabled_priority 
    ON triage_rules(user_id, enabled, priority) WHERE enabled = true;

-- Row Level Security (RLS) for multi-tenant isolation
ALTER TABLE users ENABLE ROW LEVEL SECURITY;
ALTER TABLE email_accounts ENABLE ROW LEVEL SECURITY;
ALTER TABLE triage_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE processed_emails ENABLE ROW LEVEL SECURITY;
ALTER TABLE rule_executions ENABLE ROW LEVEL SECURITY;
ALTER TABLE activity_logs ENABLE ROW LEVEL SECURITY;
ALTER TABLE usage_tracking ENABLE ROW LEVEL SECURITY;

-- RLS Policies (will be activated when application implements user context)
-- For now, commented out to allow development without full auth integration

-- CREATE POLICY users_isolation ON users FOR ALL TO app_role USING (id = current_setting('app.current_user_id')::UUID);
-- CREATE POLICY email_accounts_isolation ON email_accounts FOR ALL TO app_role USING (user_id = current_setting('app.current_user_id')::UUID);
-- CREATE POLICY triage_rules_isolation ON triage_rules FOR ALL TO app_role USING (user_id = current_setting('app.current_user_id')::UUID);
-- CREATE POLICY processed_emails_isolation ON processed_emails FOR ALL TO app_role USING (account_id IN (SELECT id FROM email_accounts WHERE user_id = current_setting('app.current_user_id')::UUID));

-- Initial data and defaults
INSERT INTO users (id, plan_type) VALUES 
    ('00000000-0000-0000-0000-000000000001', 'free'),
    ('00000000-0000-0000-0000-000000000002', 'pro')
ON CONFLICT (id) DO NOTHING;

-- Create indexes separately (PostgreSQL syntax)
CREATE INDEX idx_email_accounts_user_id ON email_accounts(user_id);
CREATE INDEX idx_email_accounts_sync_enabled ON email_accounts(sync_enabled) WHERE sync_enabled = true;

CREATE INDEX idx_triage_rules_user_id ON triage_rules(user_id);
CREATE INDEX idx_triage_rules_enabled ON triage_rules(user_id, enabled) WHERE enabled = true;
CREATE INDEX idx_triage_rules_priority ON triage_rules(user_id, priority, created_at);

CREATE INDEX idx_processed_emails_account_id ON processed_emails(account_id);
CREATE INDEX idx_processed_emails_processed_at ON processed_emails(processed_at DESC);
CREATE INDEX idx_processed_emails_priority_score ON processed_emails(priority_score DESC);
CREATE INDEX idx_processed_emails_sender ON processed_emails(sender_email);
CREATE INDEX idx_processed_emails_search ON processed_emails USING gin(to_tsvector('english', subject || ' ' || COALESCE(body_preview, '')));

CREATE INDEX idx_rule_executions_rule_id ON rule_executions(rule_id);
CREATE INDEX idx_rule_executions_executed_at ON rule_executions(executed_at DESC);
CREATE INDEX idx_rule_executions_performance ON rule_executions(rule_id, matched, execution_time_ms);

CREATE INDEX idx_activity_logs_user_id ON activity_logs(user_id);
CREATE INDEX idx_activity_logs_created_at ON activity_logs(created_at DESC);
CREATE INDEX idx_activity_logs_type ON activity_logs(activity_type);

CREATE INDEX idx_usage_tracking_user_month ON usage_tracking(user_id, tracking_month DESC);

-- Performance optimization: Analyze tables
ANALYZE users;
ANALYZE email_accounts;
ANALYZE triage_rules;
ANALYZE processed_emails;
ANALYZE rule_executions;
ANALYZE activity_logs;
ANALYZE usage_tracking;