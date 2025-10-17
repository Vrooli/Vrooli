-- Email Triage Tables Setup
-- This script creates the necessary tables for the email-triage scenario
-- while being compatible with the existing database schema

-- Ensure email_accounts table exists
CREATE TABLE IF NOT EXISTS email_accounts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID DEFAULT '00000000-0000-0000-0000-000000000001',
    email_address VARCHAR(255) NOT NULL,
    imap_settings JSONB NOT NULL,
    smtp_settings JSONB NOT NULL,
    last_sync TIMESTAMP WITH TIME ZONE,
    sync_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(user_id, email_address)
);

-- Create triage_rules table if it doesn't exist
CREATE TABLE IF NOT EXISTS triage_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID DEFAULT '00000000-0000-0000-0000-000000000001',
    name VARCHAR(255) NOT NULL,
    description TEXT NOT NULL,
    conditions JSONB NOT NULL,
    actions JSONB NOT NULL,
    priority INTEGER DEFAULT 100,
    enabled BOOLEAN DEFAULT true,
    created_by_ai BOOLEAN DEFAULT false,
    ai_confidence DECIMAL(3,2) DEFAULT 0.0,
    match_count INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create processed_emails table if it doesn't exist
CREATE TABLE IF NOT EXISTS processed_emails (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    account_id UUID NOT NULL REFERENCES email_accounts(id) ON DELETE CASCADE,
    message_id VARCHAR(500) NOT NULL,
    subject TEXT,
    sender_email VARCHAR(255),
    recipient_emails TEXT[],
    body_preview TEXT,
    full_body TEXT,
    priority_score DECIMAL(3,2) DEFAULT 0.5,
    vector_id UUID,
    processed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    actions_taken JSONB DEFAULT '[]'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb,
    UNIQUE(account_id, message_id)
);

-- Create rule_executions table for analytics
CREATE TABLE IF NOT EXISTS rule_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_id UUID NOT NULL REFERENCES triage_rules(id) ON DELETE CASCADE,
    email_id UUID NOT NULL REFERENCES processed_emails(id) ON DELETE CASCADE,
    matched BOOLEAN NOT NULL,
    confidence_score DECIMAL(3,2),
    execution_time_ms INTEGER,
    actions_executed JSONB DEFAULT '[]'::jsonb,
    executed_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create activity_logs table for audit
CREATE TABLE IF NOT EXISTS activity_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID DEFAULT '00000000-0000-0000-0000-000000000001',
    activity_type VARCHAR(50) NOT NULL,
    description TEXT NOT NULL,
    details JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX IF NOT EXISTS idx_email_accounts_user_id ON email_accounts(user_id);
CREATE INDEX IF NOT EXISTS idx_triage_rules_user_id ON triage_rules(user_id);
CREATE INDEX IF NOT EXISTS idx_processed_emails_account_id ON processed_emails(account_id);
CREATE INDEX IF NOT EXISTS idx_processed_emails_message_id ON processed_emails(message_id);
CREATE INDEX IF NOT EXISTS idx_rule_executions_rule_id ON rule_executions(rule_id);
CREATE INDEX IF NOT EXISTS idx_activity_logs_user_id ON activity_logs(user_id);

-- Note: The existing users table has different columns
-- We'll use a default user ID for development mode testing