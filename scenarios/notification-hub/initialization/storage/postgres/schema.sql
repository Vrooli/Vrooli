-- Notification Hub PostgreSQL Schema
-- Multi-tenant notification management system

-- Enable UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable trigram matching for search
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- =============================================================================
-- CORE PROFILE MANAGEMENT
-- =============================================================================

-- Profiles represent organizations/tenants in the multi-tenant system
CREATE TABLE profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) UNIQUE NOT NULL,
    
    -- API authentication
    api_key_hash VARCHAR(255) NOT NULL UNIQUE,
    api_key_prefix VARCHAR(20) NOT NULL,
    
    -- Profile settings and configuration
    settings JSONB DEFAULT '{}',
    
    -- Billing and subscription info
    plan VARCHAR(50) DEFAULT 'free',
    billing_email VARCHAR(255),
    
    -- Status tracking
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'suspended', 'inactive')),
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Indexes
    CONSTRAINT profiles_name_length CHECK (char_length(name) >= 2),
    CONSTRAINT profiles_slug_format CHECK (slug ~ '^[a-z0-9-]+$')
);

-- Profile usage limits and quotas
CREATE TABLE profile_limits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    
    -- Notification type limits
    notification_type VARCHAR(20) NOT NULL CHECK (notification_type IN ('email', 'sms', 'push', 'all')),
    
    -- Rate limits
    hourly_limit INTEGER DEFAULT 1000,
    daily_limit INTEGER DEFAULT 10000,
    monthly_limit INTEGER DEFAULT 100000,
    
    -- Cost limits (in cents)
    hourly_cost_limit INTEGER DEFAULT 5000, -- $50
    daily_cost_limit INTEGER DEFAULT 20000, -- $200  
    monthly_cost_limit INTEGER DEFAULT 500000, -- $5000
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Ensure one limit per type per profile
    UNIQUE(profile_id, notification_type)
);

-- Provider configurations per profile
CREATE TABLE profile_providers (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    
    -- Provider details
    channel VARCHAR(20) NOT NULL CHECK (channel IN ('email', 'sms', 'push')),
    provider VARCHAR(50) NOT NULL, -- 'sendgrid', 'twilio', 'firebase', etc.
    
    -- Provider configuration (encrypted)
    config JSONB NOT NULL, -- API keys, settings, etc.
    
    -- Routing configuration
    priority INTEGER DEFAULT 1, -- Lower number = higher priority
    enabled BOOLEAN DEFAULT true,
    
    -- Provider status and health
    last_success_at TIMESTAMPTZ,
    last_failure_at TIMESTAMPTZ,
    failure_count INTEGER DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Unique provider per channel per profile
    UNIQUE(profile_id, channel, provider)
);

-- =============================================================================
-- CONTACT MANAGEMENT
-- =============================================================================

-- Contacts represent notification recipients within each profile
CREATE TABLE contacts (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    
    -- Contact identification
    external_id VARCHAR(255), -- Client's user ID
    identifier VARCHAR(255) NOT NULL, -- Email, phone, or primary identifier
    
    -- Contact metadata
    first_name VARCHAR(100),
    last_name VARCHAR(100),
    timezone VARCHAR(50) DEFAULT 'UTC',
    locale VARCHAR(10) DEFAULT 'en-US',
    
    -- Contact preferences
    preferences JSONB DEFAULT '{}',
    
    -- Status tracking
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'bounced', 'unsubscribed')),
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_contacted_at TIMESTAMPTZ,
    
    -- Indexes for fast lookups
    UNIQUE(profile_id, identifier),
    INDEX ON (profile_id, external_id),
    INDEX ON (profile_id, status)
);

-- Contact channels (email, phone, push tokens) for each contact
CREATE TABLE contact_channels (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    
    -- Channel details
    type VARCHAR(20) NOT NULL CHECK (type IN ('email', 'sms', 'push')),
    value VARCHAR(500) NOT NULL, -- Email address, phone number, or push token
    
    -- Verification status
    verified BOOLEAN DEFAULT false,
    verified_at TIMESTAMPTZ,
    
    -- Channel-specific preferences
    preferences JSONB DEFAULT '{}',
    
    -- Channel status
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'inactive', 'bounced', 'invalid')),
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_used_at TIMESTAMPTZ,
    
    -- Indexes
    UNIQUE(contact_id, type, value),
    INDEX ON (type, value),
    INDEX ON (contact_id, status)
);

-- Unsubscribe management
CREATE TABLE unsubscribes (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    
    -- Unsubscribe scope
    scope VARCHAR(20) DEFAULT 'all' CHECK (scope IN ('all', 'email', 'sms', 'push', 'marketing', 'transactional')),
    
    -- Unsubscribe details
    reason VARCHAR(500),
    source VARCHAR(50) DEFAULT 'user_request', -- 'user_request', 'bounce', 'complaint', 'admin'
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Ensure no duplicate unsubscribes
    UNIQUE(contact_id, profile_id, scope)
);

-- =============================================================================
-- TEMPLATE SYSTEM
-- =============================================================================

-- Notification templates
CREATE TABLE templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    
    -- Template identification
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL,
    
    -- Template configuration
    channels TEXT[] NOT NULL DEFAULT '{}', -- Which channels this template supports
    category VARCHAR(50) DEFAULT 'general',
    
    -- Template content
    subject VARCHAR(500),
    content JSONB NOT NULL, -- Channel-specific content
    variables JSONB DEFAULT '{}', -- Expected variables and defaults
    
    -- Template settings
    settings JSONB DEFAULT '{}',
    
    -- Status
    status VARCHAR(20) DEFAULT 'draft' CHECK (status IN ('draft', 'active', 'archived')),
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(profile_id, slug),
    INDEX ON (profile_id, status),
    INDEX ON (profile_id, category)
);

-- Template versions for A/B testing and rollbacks
CREATE TABLE template_versions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    template_id UUID NOT NULL REFERENCES templates(id) ON DELETE CASCADE,
    
    -- Version details
    version INTEGER NOT NULL,
    name VARCHAR(255),
    
    -- Version content
    subject VARCHAR(500),
    content JSONB NOT NULL,
    variables JSONB DEFAULT '{}',
    
    -- Version status
    active BOOLEAN DEFAULT false,
    
    -- A/B testing
    traffic_percentage DECIMAL(5,2) DEFAULT 100.00,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    UNIQUE(template_id, version),
    INDEX ON (template_id, active)
);

-- =============================================================================
-- NOTIFICATION MANAGEMENT
-- =============================================================================

-- Core notifications table
CREATE TABLE notifications (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    contact_id UUID NOT NULL REFERENCES contacts(id) ON DELETE CASCADE,
    template_id UUID REFERENCES templates(id) ON DELETE SET NULL,
    
    -- Notification content
    subject VARCHAR(500),
    content JSONB NOT NULL, -- Channel-specific content
    variables JSONB DEFAULT '{}', -- Template variables used
    
    -- Delivery configuration
    channels_requested TEXT[] NOT NULL, -- Which channels were requested
    channels_attempted TEXT[] DEFAULT '{}', -- Which channels were actually attempted
    priority VARCHAR(10) DEFAULT 'normal' CHECK (priority IN ('low', 'normal', 'high', 'urgent')),
    
    -- Scheduling
    scheduled_at TIMESTAMPTZ DEFAULT NOW(),
    send_after TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ,
    
    -- Status tracking
    status VARCHAR(20) DEFAULT 'pending' CHECK (status IN ('pending', 'queued', 'sending', 'sent', 'failed', 'expired', 'cancelled')),
    
    -- Metadata
    metadata JSONB DEFAULT '{}',
    external_id VARCHAR(255), -- Client's tracking ID
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    sent_at TIMESTAMPTZ,
    
    -- Indexes for fast queries
    INDEX ON (profile_id, status),
    INDEX ON (contact_id, created_at DESC),
    INDEX ON (scheduled_at, status),
    INDEX ON (profile_id, external_id)
);

-- Delivery tracking per channel
CREATE TABLE notification_deliveries (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    notification_id UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    
    -- Delivery details
    channel VARCHAR(20) NOT NULL CHECK (channel IN ('email', 'sms', 'push')),
    provider VARCHAR(50) NOT NULL,
    
    -- Delivery status
    status VARCHAR(20) NOT NULL CHECK (status IN ('queued', 'sent', 'delivered', 'failed', 'bounced', 'complained')),
    
    -- Provider response
    provider_message_id VARCHAR(255),
    provider_response JSONB,
    
    -- Timing
    sent_at TIMESTAMPTZ,
    delivered_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    
    -- Cost tracking
    cost_cents INTEGER, -- Cost in cents
    
    -- Error details
    error_message TEXT,
    error_code VARCHAR(50),
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Indexes
    INDEX ON (notification_id, channel),
    INDEX ON (provider, provider_message_id),
    INDEX ON (status, delivered_at)
);

-- Notification events (opens, clicks, unsubscribes)
CREATE TABLE notification_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    notification_id UUID NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,
    
    -- Event details
    event_type VARCHAR(20) NOT NULL CHECK (event_type IN ('sent', 'delivered', 'opened', 'clicked', 'bounced', 'complained', 'unsubscribed')),
    channel VARCHAR(20) NOT NULL,
    
    -- Event metadata
    user_agent TEXT,
    ip_address INET,
    location JSONB, -- Geographic data
    
    -- Event data
    data JSONB DEFAULT '{}',
    
    -- Timestamps
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    
    -- Indexes
    INDEX ON (notification_id, event_type),
    INDEX ON (event_type, timestamp),
    INDEX ON (timestamp DESC)
);

-- =============================================================================
-- ANALYTICS AND REPORTING
-- =============================================================================

-- Aggregated analytics per profile per day
CREATE TABLE profile_analytics_daily (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES profiles(id) ON DELETE CASCADE,
    
    -- Date for aggregation
    date DATE NOT NULL,
    
    -- Notification counts
    notifications_sent INTEGER DEFAULT 0,
    notifications_delivered INTEGER DEFAULT 0,
    notifications_failed INTEGER DEFAULT 0,
    
    -- Channel breakdown
    email_sent INTEGER DEFAULT 0,
    email_delivered INTEGER DEFAULT 0,
    email_opened INTEGER DEFAULT 0,
    email_clicked INTEGER DEFAULT 0,
    email_bounced INTEGER DEFAULT 0,
    
    sms_sent INTEGER DEFAULT 0,
    sms_delivered INTEGER DEFAULT 0,
    sms_failed INTEGER DEFAULT 0,
    
    push_sent INTEGER DEFAULT 0,
    push_delivered INTEGER DEFAULT 0,
    push_opened INTEGER DEFAULT 0,
    push_failed INTEGER DEFAULT 0,
    
    -- Cost tracking
    total_cost_cents INTEGER DEFAULT 0,
    email_cost_cents INTEGER DEFAULT 0,
    sms_cost_cents INTEGER DEFAULT 0,
    push_cost_cents INTEGER DEFAULT 0,
    
    -- Timestamps
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Unique per profile per date
    UNIQUE(profile_id, date),
    INDEX ON (profile_id, date DESC)
);

-- =============================================================================
-- FUNCTIONS AND TRIGGERS
-- =============================================================================

-- Update updated_at timestamp automatically
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply updated_at triggers to relevant tables
CREATE TRIGGER update_profiles_updated_at BEFORE UPDATE ON profiles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_profile_limits_updated_at BEFORE UPDATE ON profile_limits FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_profile_providers_updated_at BEFORE UPDATE ON profile_providers FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_contacts_updated_at BEFORE UPDATE ON contacts FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_contact_channels_updated_at BEFORE UPDATE ON contact_channels FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_templates_updated_at BEFORE UPDATE ON templates FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_notifications_updated_at BEFORE UPDATE ON notifications FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_notification_deliveries_updated_at BEFORE UPDATE ON notification_deliveries FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to get profile by API key
CREATE OR REPLACE FUNCTION get_profile_by_api_key(api_key TEXT)
RETURNS profiles AS $$
DECLARE
    profile_record profiles;
BEGIN
    SELECT * INTO profile_record
    FROM profiles
    WHERE api_key_hash = crypt(api_key, api_key_hash)
    AND status = 'active';
    
    RETURN profile_record;
END;
$$ LANGUAGE plpgsql SECURITY DEFINER;

-- Function to check rate limits
CREATE OR REPLACE FUNCTION check_rate_limit(
    p_profile_id UUID,
    p_notification_type VARCHAR(20),
    p_count INTEGER DEFAULT 1
) RETURNS BOOLEAN AS $$
DECLARE
    hourly_count INTEGER;
    daily_count INTEGER;
    monthly_count INTEGER;
    limit_record profile_limits;
BEGIN
    -- Get current usage counts
    SELECT COALESCE(SUM(
        CASE 
            WHEN p_notification_type = 'all' THEN 1
            WHEN p_notification_type = ANY(nd.channels_attempted) THEN 1
            ELSE 0
        END
    ), 0) INTO hourly_count
    FROM notifications n
    JOIN notification_deliveries nd ON n.id = nd.notification_id
    WHERE n.profile_id = p_profile_id
    AND n.created_at >= NOW() - INTERVAL '1 hour';
    
    -- Get limits
    SELECT * INTO limit_record
    FROM profile_limits
    WHERE profile_id = p_profile_id 
    AND notification_type = COALESCE(p_notification_type, 'all')
    LIMIT 1;
    
    -- Check if within limits
    IF limit_record IS NULL OR 
       (hourly_count + p_count <= limit_record.hourly_limit) THEN
        RETURN TRUE;
    END IF;
    
    RETURN FALSE;
END;
$$ LANGUAGE plpgsql;

-- =============================================================================
-- INDEXES FOR PERFORMANCE
-- =============================================================================

-- Additional performance indexes
CREATE INDEX idx_notifications_profile_created ON notifications(profile_id, created_at DESC);
CREATE INDEX idx_notifications_status_scheduled ON notifications(status, scheduled_at) WHERE status = 'pending';
CREATE INDEX idx_notification_deliveries_provider_status ON notification_deliveries(provider, status);
CREATE INDEX idx_notification_events_type_timestamp ON notification_events(event_type, timestamp DESC);
CREATE INDEX idx_contacts_profile_status ON contacts(profile_id, status);
CREATE INDEX idx_contact_channels_type_value ON contact_channels(type, value);

-- Text search indexes
CREATE INDEX idx_profiles_name_search ON profiles USING GIN(name gin_trgm_ops);
CREATE INDEX idx_contacts_name_search ON contacts USING GIN((first_name || ' ' || last_name) gin_trgm_ops);
CREATE INDEX idx_templates_name_search ON templates USING GIN(name gin_trgm_ops);

-- JSONB indexes for efficient queries
CREATE INDEX idx_profiles_settings ON profiles USING GIN(settings);
CREATE INDEX idx_contacts_preferences ON contacts USING GIN(preferences);
CREATE INDEX idx_notifications_metadata ON notifications USING GIN(metadata);
CREATE INDEX idx_notification_deliveries_provider_response ON notification_deliveries USING GIN(provider_response);
CREATE INDEX idx_notification_events_data ON notification_events USING GIN(data);

COMMENT ON SCHEMA public IS 'Notification Hub - Multi-tenant notification management system';