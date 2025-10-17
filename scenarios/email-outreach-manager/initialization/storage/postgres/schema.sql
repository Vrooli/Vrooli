-- Email Outreach Manager Database Schema
-- This schema supports campaign management, template generation, and analytics

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Campaigns table
CREATE TABLE IF NOT EXISTS campaigns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    template_id UUID,
    status VARCHAR(50) NOT NULL DEFAULT 'draft',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    send_schedule TIMESTAMP,
    total_recipients INTEGER NOT NULL DEFAULT 0,
    sent_count INTEGER NOT NULL DEFAULT 0,
    analytics JSONB DEFAULT '{}'::jsonb,
    CONSTRAINT valid_status CHECK (status IN ('draft', 'scheduled', 'sending', 'completed', 'paused'))
);

-- Templates table
CREATE TABLE IF NOT EXISTS templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    subject VARCHAR(500) NOT NULL,
    html_content TEXT NOT NULL,
    text_content TEXT NOT NULL,
    ai_generated_from TEXT,
    personalization_fields TEXT[],
    style_category VARCHAR(50) NOT NULL DEFAULT 'professional',
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    CONSTRAINT valid_style CHECK (style_category IN ('professional', 'creative', 'casual'))
);

-- Email recipients table
CREATE TABLE IF NOT EXISTS email_recipients (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    pronouns VARCHAR(50),
    contact_book_id UUID,
    personalization_data JSONB DEFAULT '{}'::jsonb,
    personalization_level VARCHAR(50) NOT NULL DEFAULT 'template_only',
    send_status VARCHAR(50) NOT NULL DEFAULT 'pending',
    sent_at TIMESTAMP,
    opened_at TIMESTAMP,
    clicked_at TIMESTAMP,
    CONSTRAINT valid_personalization_level CHECK (personalization_level IN ('full', 'partial', 'template_only')),
    CONSTRAINT valid_send_status CHECK (send_status IN ('pending', 'sent', 'failed', 'bounced'))
);

-- Drip sequences table
CREATE TABLE IF NOT EXISTS drip_sequences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    campaign_id UUID NOT NULL REFERENCES campaigns(id) ON DELETE CASCADE,
    step_number INTEGER NOT NULL,
    delay_days INTEGER NOT NULL,
    template_id UUID REFERENCES templates(id),
    condition_type VARCHAR(50) NOT NULL DEFAULT 'time_based',
    condition_criteria JSONB DEFAULT '{}'::jsonb,
    CONSTRAINT valid_condition_type CHECK (condition_type IN ('time_based', 'engagement_based'))
);

-- Add foreign key for template_id in campaigns (can't add inline due to circular dependency)
ALTER TABLE campaigns
    ADD CONSTRAINT fk_campaigns_template
    FOREIGN KEY (template_id)
    REFERENCES templates(id)
    ON DELETE SET NULL;

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns(status);
CREATE INDEX IF NOT EXISTS idx_campaigns_created_at ON campaigns(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_email_recipients_campaign_id ON email_recipients(campaign_id);
CREATE INDEX IF NOT EXISTS idx_email_recipients_send_status ON email_recipients(send_status);
CREATE INDEX IF NOT EXISTS idx_drip_sequences_campaign_id ON drip_sequences(campaign_id);
CREATE INDEX IF NOT EXISTS idx_templates_style_category ON templates(style_category);
CREATE INDEX IF NOT EXISTS idx_templates_created_at ON templates(created_at DESC);

-- Comments for documentation
COMMENT ON TABLE campaigns IS 'Email marketing campaigns with AI-generated templates';
COMMENT ON TABLE templates IS 'Email templates with personalization support';
COMMENT ON TABLE email_recipients IS 'Campaign recipients with personalization tiers';
COMMENT ON TABLE drip_sequences IS 'Multi-step drip campaign sequences';
