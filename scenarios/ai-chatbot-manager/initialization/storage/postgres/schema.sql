-- AI Chatbot Manager Database Schema
-- This schema supports multi-tenant chatbot management with conversation tracking and analytics

-- Create extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Chatbots table - stores chatbot configurations and settings
CREATE TABLE chatbots (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    personality TEXT NOT NULL DEFAULT 'You are a helpful assistant.',
    knowledge_base TEXT,
    model_config JSONB DEFAULT '{"model": "llama3.2", "temperature": 0.7, "max_tokens": 1000}',
    widget_config JSONB DEFAULT '{"theme": "light", "position": "bottom-right", "primaryColor": "#007bff"}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    
    -- Ensure name is unique and not empty
    CONSTRAINT chatbots_name_not_empty CHECK (length(trim(name)) > 0)
);

-- Conversations table - tracks individual chat sessions
CREATE TABLE conversations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chatbot_id UUID NOT NULL REFERENCES chatbots(id) ON DELETE CASCADE,
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
CREATE TABLE messages (
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
CREATE TABLE daily_analytics (
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
CREATE TABLE intent_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    chatbot_id UUID NOT NULL REFERENCES chatbots(id) ON DELETE CASCADE,
    pattern TEXT NOT NULL,
    intent_name VARCHAR(255) NOT NULL,
    confidence FLOAT NOT NULL DEFAULT 0.0,
    occurrence_count INTEGER DEFAULT 1,
    last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance optimization
CREATE INDEX idx_conversations_chatbot_id ON conversations(chatbot_id);
CREATE INDEX idx_conversations_started_at ON conversations(started_at);
CREATE INDEX idx_conversations_session_id ON conversations(session_id);

CREATE INDEX idx_messages_conversation_id ON messages(conversation_id);
CREATE INDEX idx_messages_timestamp ON messages(timestamp);
CREATE INDEX idx_messages_role ON messages(role);

CREATE INDEX idx_daily_analytics_chatbot_id ON daily_analytics(chatbot_id);
CREATE INDEX idx_daily_analytics_date ON daily_analytics(date);

CREATE INDEX idx_intent_patterns_chatbot_id ON intent_patterns(chatbot_id);
CREATE INDEX idx_intent_patterns_intent_name ON intent_patterns(intent_name);

-- Triggers to maintain updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

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
CREATE VIEW chatbot_analytics_summary AS
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