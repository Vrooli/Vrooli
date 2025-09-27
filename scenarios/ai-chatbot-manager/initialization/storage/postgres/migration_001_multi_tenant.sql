-- Migration: Add Multi-Tenant Support to AI Chatbot Manager
-- Version: 001
-- Description: Adds tenant/organization support for multi-tenant architecture

-- Create tenants table to support multi-tenant architecture
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) UNIQUE NOT NULL, -- URL-friendly identifier
    description TEXT,
    config JSONB DEFAULT '{}', -- Tenant-specific configuration
    plan VARCHAR(50) DEFAULT 'starter', -- starter, professional, enterprise
    max_chatbots INTEGER DEFAULT 3,
    max_conversations_per_month INTEGER DEFAULT 1000,
    api_key VARCHAR(255) UNIQUE DEFAULT md5(random()::text || clock_timestamp()::text),
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT tenants_name_not_empty CHECK (length(trim(name)) > 0),
    CONSTRAINT tenants_slug_format CHECK (slug ~ '^[a-z0-9-]+$')
);

-- Create tenant users table for user management within tenants
CREATE TABLE IF NOT EXISTS tenant_users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    role VARCHAR(50) DEFAULT 'member', -- owner, admin, member, viewer
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT tenant_users_email_unique_per_tenant UNIQUE (tenant_id, email),
    CONSTRAINT tenant_users_email_format CHECK (email ~* '^[A-Za-z0-9._%+-]+@[A-Za-z0-9.-]+\.[A-Z|a-z]{2,}$')
);

-- Add tenant_id to chatbots table if it doesn't exist
DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='chatbots' AND column_name='tenant_id') THEN
        ALTER TABLE chatbots 
        ADD COLUMN tenant_id UUID REFERENCES tenants(id) ON DELETE CASCADE;
        
        -- Create a default tenant for existing chatbots
        INSERT INTO tenants (name, slug, plan, max_chatbots, max_conversations_per_month) 
        VALUES ('Default Tenant', 'default', 'enterprise', 100, 100000)
        ON CONFLICT (slug) DO NOTHING;
        
        -- Assign existing chatbots to the default tenant
        UPDATE chatbots 
        SET tenant_id = (SELECT id FROM tenants WHERE slug = 'default')
        WHERE tenant_id IS NULL;
        
        -- Now make tenant_id NOT NULL
        ALTER TABLE chatbots ALTER COLUMN tenant_id SET NOT NULL;
    END IF;
END $$;

-- Create tenant usage tracking table
CREATE TABLE IF NOT EXISTS tenant_usage (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    month DATE NOT NULL, -- First day of the month
    chatbot_count INTEGER DEFAULT 0,
    conversation_count INTEGER DEFAULT 0,
    message_count INTEGER DEFAULT 0,
    leads_captured INTEGER DEFAULT 0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT tenant_usage_unique_month UNIQUE (tenant_id, month)
);

-- Create API keys table for better key management
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    key_hash VARCHAR(255) UNIQUE NOT NULL, -- Store hashed API key
    name VARCHAR(255),
    description TEXT,
    permissions JSONB DEFAULT '["read", "write"]',
    last_used_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    CONSTRAINT api_keys_name_not_empty CHECK (length(trim(name)) > 0)
);

-- Create A/B testing table for personality experiments
CREATE TABLE IF NOT EXISTS ab_tests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chatbot_id UUID NOT NULL REFERENCES chatbots(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    variant_a JSONB NOT NULL, -- Control personality/config
    variant_b JSONB NOT NULL, -- Test personality/config
    traffic_split FLOAT DEFAULT 0.5 CHECK (traffic_split >= 0 AND traffic_split <= 1),
    metrics JSONB DEFAULT '{}', -- Track conversion rates, engagement, etc.
    status VARCHAR(50) DEFAULT 'draft', -- draft, running, paused, completed
    started_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create A/B test results table
CREATE TABLE IF NOT EXISTS ab_test_results (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    ab_test_id UUID NOT NULL REFERENCES ab_tests(id) ON DELETE CASCADE,
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    variant VARCHAR(1) NOT NULL CHECK (variant IN ('A', 'B')),
    conversion BOOLEAN DEFAULT false,
    engagement_score FLOAT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Add variant column to conversations for A/B testing
DO $$ 
BEGIN 
    IF NOT EXISTS (SELECT 1 FROM information_schema.columns 
                   WHERE table_name='conversations' AND column_name='ab_test_variant') THEN
        ALTER TABLE conversations 
        ADD COLUMN ab_test_variant VARCHAR(1) CHECK (ab_test_variant IN ('A', 'B'));
        
        ALTER TABLE conversations 
        ADD COLUMN ab_test_id UUID REFERENCES ab_tests(id) ON DELETE SET NULL;
    END IF;
END $$;

-- Create CRM integrations table
CREATE TABLE IF NOT EXISTS crm_integrations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    chatbot_id UUID REFERENCES chatbots(id) ON DELETE CASCADE, -- Null for tenant-wide integration
    type VARCHAR(50) NOT NULL, -- salesforce, hubspot, pipedrive, webhook
    config JSONB NOT NULL, -- API endpoints, credentials, field mappings
    sync_enabled BOOLEAN DEFAULT true,
    last_sync_at TIMESTAMP WITH TIME ZONE,
    sync_status VARCHAR(50) DEFAULT 'pending',
    sync_errors JSONB,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create CRM sync log table
CREATE TABLE IF NOT EXISTS crm_sync_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    integration_id UUID NOT NULL REFERENCES crm_integrations(id) ON DELETE CASCADE,
    conversation_id UUID REFERENCES conversations(id) ON DELETE CASCADE,
    action VARCHAR(50) NOT NULL, -- create_lead, update_lead, sync_conversation
    status VARCHAR(50) NOT NULL, -- success, failed, pending
    request_data JSONB,
    response_data JSONB,
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for multi-tenant queries
CREATE INDEX IF NOT EXISTS idx_chatbots_tenant_id ON chatbots(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_users_tenant_id ON tenant_users(tenant_id);
CREATE INDEX IF NOT EXISTS idx_tenant_usage_tenant_month ON tenant_usage(tenant_id, month);
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ab_tests_tenant_id ON ab_tests(tenant_id);
CREATE INDEX IF NOT EXISTS idx_ab_tests_chatbot_id ON ab_tests(chatbot_id);
CREATE INDEX IF NOT EXISTS idx_crm_integrations_tenant_id ON crm_integrations(tenant_id);

-- Create view for tenant dashboard metrics
CREATE OR REPLACE VIEW tenant_metrics AS
SELECT 
    t.id as tenant_id,
    t.name as tenant_name,
    t.plan,
    COUNT(DISTINCT c.id) as chatbot_count,
    COUNT(DISTINCT conv.id) FILTER (WHERE conv.started_at >= date_trunc('month', CURRENT_DATE)) as monthly_conversations,
    COUNT(DISTINCT conv.id) FILTER (WHERE conv.lead_captured = true AND conv.started_at >= date_trunc('month', CURRENT_DATE)) as monthly_leads,
    t.max_chatbots,
    t.max_conversations_per_month
FROM tenants t
LEFT JOIN chatbots c ON t.id = c.tenant_id AND c.is_active = true
LEFT JOIN conversations conv ON c.id = conv.chatbot_id
GROUP BY t.id, t.name, t.plan, t.max_chatbots, t.max_conversations_per_month;

-- Create trigger to update tenant usage
CREATE OR REPLACE FUNCTION update_tenant_usage()
RETURNS TRIGGER AS $$
BEGIN
    -- Update usage when a new conversation is created
    INSERT INTO tenant_usage (tenant_id, month, conversation_count)
    SELECT 
        c.tenant_id,
        date_trunc('month', NEW.started_at)::date,
        1
    FROM chatbots c
    WHERE c.id = NEW.chatbot_id
    ON CONFLICT (tenant_id, month) 
    DO UPDATE SET 
        conversation_count = tenant_usage.conversation_count + 1,
        updated_at = NOW();
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply trigger if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_trigger WHERE tgname = 'trigger_update_tenant_usage') THEN
        CREATE TRIGGER trigger_update_tenant_usage
        AFTER INSERT ON conversations
        FOR EACH ROW
        EXECUTE FUNCTION update_tenant_usage();
    END IF;
END $$;

-- Add comments for documentation
COMMENT ON TABLE tenants IS 'Multi-tenant organizations that own chatbots';
COMMENT ON TABLE tenant_users IS 'Users within each tenant organization';
COMMENT ON TABLE tenant_usage IS 'Monthly usage tracking for billing and limits';
COMMENT ON TABLE api_keys IS 'API key management for tenant authentication';
COMMENT ON TABLE ab_tests IS 'A/B testing configurations for chatbot personalities';
COMMENT ON TABLE crm_integrations IS 'CRM system integrations for lead management';