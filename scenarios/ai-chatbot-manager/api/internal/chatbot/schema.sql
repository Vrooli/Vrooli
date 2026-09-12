-- AI Chatbot Manager Database Schema
-- This schema supports multi-tenant chatbot management with conversation tracking and analytics

-- Create extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

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


-- Chatbots table - stores chatbot configurations and settings
CREATE TABLE IF NOT EXISTS chatbots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID REFERENCES tenants(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    personality TEXT NOT NULL DEFAULT 'You are a helpful assistant.',
    knowledge_base TEXT,
    model_config JSONB DEFAULT '{"model": "chat.default", "temperature": 0.7, "max_tokens": 1000}',
    widget_config JSONB DEFAULT '{"theme": "light", "position": "bottom-right", "primaryColor": "#007bff"}',
    escalation_config JSONB DEFAULT '{"enabled": false, "threshold": 0.5, "webhook_url": null, "email": null}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Ensure name is unique and not empty
    CONSTRAINT chatbots_name_not_empty CHECK (length(trim(name)) > 0)
);

-- Create A/B testing table for personality experiments
CREATE TABLE IF NOT EXISTS ab_tests (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chatbot_id UUID NOT NULL REFERENCES chatbots(id) ON DELETE CASCADE,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    variant_a JSONB NOT NULL,
    variant_b JSONB NOT NULL,
    traffic_split FLOAT DEFAULT 0.5 CHECK (traffic_split >= 0 AND traffic_split <= 1),
    metrics JSONB DEFAULT '{}',
    status VARCHAR(50) DEFAULT 'draft',
    started_at TIMESTAMP WITH TIME ZONE,
    ended_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Conversations table - tracks individual chat sessions
CREATE TABLE IF NOT EXISTS conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chatbot_id UUID NOT NULL REFERENCES chatbots(id) ON DELETE CASCADE,
    ab_test_variant VARCHAR(1) CHECK (ab_test_variant IN ('A', 'B')),
    ab_test_id UUID REFERENCES ab_tests(id) ON DELETE SET NULL,
    session_id VARCHAR(255) NOT NULL, -- Client-side generated session identifier
    user_ip INET,
    user_agent TEXT,
    started_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    ended_at TIMESTAMP WITH TIME ZONE,
    lead_captured BOOLEAN DEFAULT false,
    lead_data JSONB, -- Store captured lead information (email, phone, name, etc.)
    conversation_rating INTEGER CHECK (conversation_rating BETWEEN 1 AND 5),
    
    -- Indexes for performance
    CONSTRAINT conversations_session_chatbot UNIQUE (session_id, chatbot_id)
);

-- Messages table - stores individual messages in conversations  
CREATE TABLE IF NOT EXISTS messages (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role VARCHAR(20) NOT NULL CHECK (role IN ('user', 'assistant')),
    content TEXT NOT NULL,
    metadata JSONB, -- Store additional message metadata (confidence, intent, etc.)
    timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Ensure content is not empty
    CONSTRAINT messages_content_not_empty CHECK (length(trim(content)) > 0)
);

-- Daily analytics table - aggregated metrics per chatbot per day
CREATE TABLE IF NOT EXISTS daily_analytics (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chatbot_id UUID NOT NULL REFERENCES chatbots(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    total_conversations INTEGER DEFAULT 0,
    total_messages INTEGER DEFAULT 0,
    leads_captured INTEGER DEFAULT 0,
    avg_conversation_length FLOAT DEFAULT 0.0,
    avg_response_time FLOAT DEFAULT 0.0,
    engagement_score FLOAT DEFAULT 0.0,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- One record per chatbot per day
    CONSTRAINT daily_analytics_chatbot_date UNIQUE (chatbot_id, date)
);

-- Intent patterns table - track common user intents for training
CREATE TABLE IF NOT EXISTS intent_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chatbot_id UUID NOT NULL REFERENCES chatbots(id) ON DELETE CASCADE,
    pattern TEXT NOT NULL,
    intent_name VARCHAR(255) NOT NULL,
    confidence FLOAT NOT NULL DEFAULT 0.0,
    occurrence_count INTEGER DEFAULT 1,
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Escalations table - tracks when human intervention was requested
CREATE TABLE IF NOT EXISTS escalations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    chatbot_id UUID NOT NULL REFERENCES chatbots(id) ON DELETE CASCADE,
    reason TEXT NOT NULL,
    confidence_score FLOAT,
    escalation_type VARCHAR(50) DEFAULT 'low_confidence',
    status VARCHAR(50) DEFAULT 'pending',
    escalated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    resolved_at TIMESTAMP WITH TIME ZONE,
    resolution_notes TEXT,
    webhook_response JSONB,
    email_sent BOOLEAN DEFAULT false
);

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


-- Indexes for performance optimization
CREATE INDEX IF NOT EXISTS idx_conversations_chatbot_id ON conversations(chatbot_id);
CREATE INDEX IF NOT EXISTS idx_conversations_started_at ON conversations(started_at);
CREATE INDEX IF NOT EXISTS idx_conversations_session_id ON conversations(session_id);

CREATE INDEX IF NOT EXISTS idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX IF NOT EXISTS idx_messages_timestamp ON messages(timestamp);
CREATE INDEX IF NOT EXISTS idx_messages_role ON messages(role);

CREATE INDEX IF NOT EXISTS idx_daily_analytics_chatbot_id ON daily_analytics(chatbot_id);
CREATE INDEX IF NOT EXISTS idx_daily_analytics_date ON daily_analytics(date);

CREATE INDEX IF NOT EXISTS idx_intent_patterns_chatbot_id ON intent_patterns(chatbot_id);
CREATE INDEX IF NOT EXISTS idx_intent_patterns_intent_name ON intent_patterns(intent_name);

-- Triggers to maintain updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

DROP TRIGGER IF EXISTS update_chatbots_updated_at ON chatbots;
CREATE TRIGGER update_chatbots_updated_at 
    BEFORE UPDATE ON chatbots 
    FOR EACH ROW 
    EXECUTE FUNCTION update_updated_at_column();

-- Function to calculate conversation engagement score
CREATE OR REPLACE FUNCTION calculate_engagement_score(conversation_uuid UUID)
RETURNS FLOAT AS $$
DECLARE
    message_count INTEGER;
    conversation_duration INTERVAL;
    user_messages INTEGER;
    score FLOAT;
BEGIN
    -- Get message count and user message count
    SELECT COUNT(*), COUNT(*) FILTER (WHERE role = 'user')
    INTO message_count, user_messages
    FROM messages 
    WHERE conversation_id = conversation_uuid;
    
    -- Get conversation duration
    SELECT (ended_at - started_at)
    INTO conversation_duration
    FROM conversations 
    WHERE id = conversation_uuid;
    
    -- Calculate base score from message exchange
    score := LEAST(user_messages * 0.5, 5.0);
    
    -- Bonus for longer conversations
    IF message_count > 10 THEN
        score := score + 1.0;
    END IF;
    
    -- Bonus for conversations lasting more than 5 minutes
    IF conversation_duration > INTERVAL '5 minutes' THEN
        score := score + 0.5;
    END IF;
    
    RETURN LEAST(score, 10.0);
END;
$$ LANGUAGE plpgsql;

-- Function to update daily analytics (called by application)
CREATE OR REPLACE FUNCTION update_daily_analytics(target_date DATE, target_chatbot_id UUID)
RETURNS VOID AS $$
DECLARE
    analytics_data RECORD;
BEGIN
    -- Calculate analytics for the given date and chatbot
    SELECT 
        COUNT(DISTINCT c.id) as conversations,
        COUNT(m.id) as messages,
        COUNT(*) FILTER (WHERE c.lead_captured = true) as leads,
        AVG(
            (SELECT COUNT(*) FROM messages WHERE conversation_id = c.id)
        ) as avg_length,
        AVG(calculate_engagement_score(c.id)) as engagement
    INTO analytics_data
    FROM conversations c
    LEFT JOIN messages m ON c.id = m.conversation_id
    WHERE c.chatbot_id = target_chatbot_id 
    AND DATE(c.started_at) = target_date;
    
    -- Insert or update daily analytics record
    INSERT INTO daily_analytics (
        chatbot_id, 
        date, 
        total_conversations, 
        total_messages, 
        leads_captured,
        avg_conversation_length,
        engagement_score
    ) VALUES (
        target_chatbot_id,
        target_date,
        COALESCE(analytics_data.conversations, 0),
        COALESCE(analytics_data.messages, 0),
        COALESCE(analytics_data.leads, 0),
        COALESCE(analytics_data.avg_length, 0.0),
        COALESCE(analytics_data.engagement, 0.0)
    )
    ON CONFLICT (chatbot_id, date)
    DO UPDATE SET
        total_conversations = EXCLUDED.total_conversations,
        total_messages = EXCLUDED.total_messages,
        leads_captured = EXCLUDED.leads_captured,
        avg_conversation_length = EXCLUDED.avg_conversation_length,
        engagement_score = EXCLUDED.engagement_score;
END;
$$ LANGUAGE plpgsql;

-- View for easy chatbot analytics access
CREATE OR REPLACE VIEW chatbot_analytics_summary AS
SELECT 
    c.id,
    c.name,
    c.is_active,
    COUNT(DISTINCT conv.id) as total_conversations,
    COUNT(DISTINCT CASE WHEN conv.lead_captured THEN conv.id END) as total_leads,
    COUNT(m.id) as total_messages,
    AVG(calculate_engagement_score(conv.id)) as avg_engagement_score,
    MAX(conv.started_at) as last_conversation_at,
    c.created_at,
    c.updated_at
FROM chatbots c
LEFT JOIN conversations conv ON c.id = conv.chatbot_id
LEFT JOIN messages m ON conv.id = m.conversation_id
GROUP BY c.id, c.name, c.is_active, c.created_at, c.updated_at;

-- Comments for documentation
COMMENT ON TABLE chatbots IS 'Stores chatbot configurations, personalities, and settings';
COMMENT ON TABLE conversations IS 'Tracks individual chat sessions with users';
COMMENT ON TABLE messages IS 'Individual messages within conversations';
COMMENT ON TABLE daily_analytics IS 'Aggregated daily metrics per chatbot for reporting';
COMMENT ON TABLE intent_patterns IS 'Machine learning patterns for intent recognition';

COMMENT ON COLUMN chatbots.personality IS 'System prompt defining chatbot behavior and personality';
COMMENT ON COLUMN chatbots.knowledge_base IS 'Domain-specific information the chatbot can reference';
COMMENT ON COLUMN chatbots.model_config IS 'Ollama model parameters (temperature, max_tokens, etc.)';
COMMENT ON COLUMN chatbots.widget_config IS 'UI customization settings for embeddable widget';

COMMENT ON COLUMN conversations.session_id IS 'Client-generated session identifier for tracking';
COMMENT ON COLUMN conversations.lead_data IS 'Captured contact information and qualification data';

COMMENT ON COLUMN messages.metadata IS 'Additional message context (confidence, intent classification, etc.)';

COMMENT ON FUNCTION calculate_engagement_score(UUID) IS 'Calculates conversation quality score based on length and interaction patterns';
COMMENT ON FUNCTION update_daily_analytics(DATE, UUID) IS 'Updates daily analytics metrics for specified date and chatbot';
