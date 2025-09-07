-- Migration: 001_initial_schema.sql
-- Email Triage Initial Database Schema
-- Created: 2025-01-09
-- Description: Creates all tables, indexes, and functions for the email triage system

-- Ensure we have the required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Users table (lightweight reference table for multi-tenancy)
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    email_profiles JSONB DEFAULT '[]'::jsonb,
    preferences JSONB DEFAULT '{}'::jsonb,
    plan_type VARCHAR(50) DEFAULT 'free' CHECK (plan_type IN ('free', 'pro', 'business')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Email accounts table
CREATE TABLE email_accounts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email_address VARCHAR(255) NOT NULL,
    imap_settings JSONB NOT NULL,
    smtp_settings JSONB NOT NULL,
    last_sync TIMESTAMP WITH TIME ZONE,
    sync_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(user_id, email_address)
);

-- Triage rules table
CREATE TABLE triage_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    conditions JSONB NOT NULL,
    actions JSONB NOT NULL,
    priority INTEGER DEFAULT 100 CHECK (priority >= 0 AND priority <= 1000),
    enabled BOOLEAN DEFAULT true,
    created_by_ai BOOLEAN DEFAULT false,
    ai_confidence DECIMAL(3,2) DEFAULT 0.0 CHECK (ai_confidence >= 0.0 AND ai_confidence <= 1.0),
    match_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Processed emails table
CREATE TABLE processed_emails (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    account_id UUID NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    message_id VARCHAR(500) NOT NULL,
    subject TEXT,
    sender_email VARCHAR(255),
    recipient_emails TEXT[],
    body_preview TEXT,
    full_body TEXT,
    priority_score DECIMAL(3,2) DEFAULT 0.5 CHECK (priority_score >= 0.0 AND priority_score <= 1.0),
    vector_id UUID,
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    actions_taken JSONB DEFAULT '[]'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb,
    
    UNIQUE(account_id, message_id)
);

-- Rule execution log table
CREATE TABLE rule_executions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    rule_id UUID NOT NULL REFERENCES triage_rules(id) ON DELETE CASCADE,
    email_id UUID NOT NULL REFERENCES processed_emails(id) ON DELETE CASCADE,
    matched BOOLEAN NOT NULL,
    confidence_score DECIMAL(3,2),
    execution_time_ms INTEGER,
    actions_executed JSONB DEFAULT '[]'::jsonb,
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Activity log table
CREATE TABLE activity_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    activity_type VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    details JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Usage tracking table
CREATE TABLE usage_tracking (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tracking_month DATE NOT NULL,
    emails_processed INTEGER DEFAULT 0,
    rules_executed INTEGER DEFAULT 0,
    api_calls INTEGER DEFAULT 0,
    search_queries INTEGER DEFAULT 0,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    UNIQUE(user_id, tracking_month)
);

-- Create indexes for performance
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

CREATE INDEX idx_activity_logs_user_id ON activity_logs(user_id);
CREATE INDEX idx_activity_logs_created_at ON activity_logs(created_at DESC);

CREATE INDEX idx_usage_tracking_user_month ON usage_tracking(user_id, tracking_month DESC);

-- Create trigger function for updating timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers
CREATE TRIGGER update_users_updated_at BEFORE UPDATE ON users FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_email_accounts_updated_at BEFORE UPDATE ON email_accounts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_triage_rules_updated_at BEFORE UPDATE ON triage_rules FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_usage_tracking_updated_at BEFORE UPDATE ON usage_tracking FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create utility functions
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

CREATE OR REPLACE FUNCTION log_user_activity(p_user_id UUID, p_activity_type VARCHAR(50), p_description TEXT, p_details JSONB DEFAULT '{}'::jsonb)
RETURNS VOID AS $$
BEGIN
    INSERT INTO activity_logs (user_id, activity_type, description, details)
    VALUES (p_user_id, p_activity_type, p_description, p_details);
END;
$$ language 'plpgsql';

-- Insert sample data for testing
INSERT INTO users (id, plan_type) VALUES 
    ('00000000-0000-0000-0000-000000000001', 'free'),
    ('00000000-0000-0000-0000-000000000002', 'pro')
ON CONFLICT (id) DO NOTHING;