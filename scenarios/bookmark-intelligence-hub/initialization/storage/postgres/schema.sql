-- Bookmark Intelligence Hub Database Schema
-- Version: 1.0.0
-- Created: 2025-09-06

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Bookmark Profiles table
-- Stores user profiles with their bookmark processing preferences
CREATE TABLE bookmark_profiles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL,  -- Links to scenario-authenticator user management
    name VARCHAR(255) NOT NULL,
    description TEXT,
    platform_configs JSONB DEFAULT '{}', -- Platform-specific configuration
    categorization_rules JSONB DEFAULT '{}', -- User-defined categorization rules
    integration_settings JSONB DEFAULT '{}', -- Settings for cross-scenario integrations
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Bookmark Items table  
-- Stores individual bookmarks and their processed metadata
CREATE TABLE bookmark_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES bookmark_profiles(id) ON DELETE CASCADE,
    platform VARCHAR(50) NOT NULL, -- reddit, twitter, tiktok, etc.
    original_url TEXT NOT NULL,
    title TEXT,
    content_text TEXT,
    author_name VARCHAR(255),
    author_username VARCHAR(255),
    content_metadata JSONB DEFAULT '{}', -- Platform-specific metadata
    category_assigned VARCHAR(100),
    category_confidence FLOAT DEFAULT 0.0, -- 0.0 to 1.0 confidence score
    suggested_actions JSONB DEFAULT '[]', -- Array of suggested actions
    user_feedback JSONB DEFAULT '{}', -- User approval/rejection feedback
    processing_status VARCHAR(50) DEFAULT 'pending', -- pending, processed, failed
    hash_signature VARCHAR(64), -- For deduplication
    processed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Action Items table
-- Stores suggested actions for bookmarks and their approval status
CREATE TABLE action_items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    bookmark_item_id UUID NOT NULL REFERENCES bookmark_items(id) ON DELETE CASCADE,
    action_type VARCHAR(100) NOT NULL, -- add_to_recipe_book, schedule_workout, etc.
    target_scenario VARCHAR(100), -- Target scenario for the action
    action_data JSONB DEFAULT '{}', -- Data needed to execute the action
    approval_status VARCHAR(50) DEFAULT 'pending', -- pending, approved, rejected, executed
    confidence_score FLOAT DEFAULT 0.0, -- How confident we are in this suggestion
    executed_at TIMESTAMP WITH TIME ZONE,
    execution_result JSONB DEFAULT '{}', -- Result of action execution
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Category Rules table
-- Stores user-defined and learned categorization rules
CREATE TABLE category_rules (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES bookmark_profiles(id) ON DELETE CASCADE,
    category_name VARCHAR(100) NOT NULL,
    rule_type VARCHAR(50) NOT NULL, -- keyword, pattern, ml_model, user_defined
    keywords TEXT[], -- Array of keywords for matching
    patterns JSONB DEFAULT '{}', -- Regex patterns and other matching rules
    integration_actions JSONB DEFAULT '[]', -- Default actions for this category
    priority INTEGER DEFAULT 0, -- Higher priority rules checked first
    success_rate FLOAT DEFAULT 0.0, -- Track accuracy of this rule
    usage_count INTEGER DEFAULT 0, -- How many times this rule has been used
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Platform Integrations table
-- Stores platform-specific integration configurations and status
CREATE TABLE platform_integrations (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES bookmark_profiles(id) ON DELETE CASCADE,
    platform_name VARCHAR(50) NOT NULL, -- reddit, twitter, tiktok
    integration_type VARCHAR(50) NOT NULL, -- huginn, browserless, api
    configuration JSONB DEFAULT '{}', -- Platform-specific config
    authentication_data JSONB DEFAULT '{}', -- Encrypted auth tokens/cookies
    status VARCHAR(50) DEFAULT 'inactive', -- active, inactive, error, rate_limited
    last_sync_at TIMESTAMP WITH TIME ZONE,
    sync_frequency INTERVAL DEFAULT '1 hour',
    error_count INTEGER DEFAULT 0,
    last_error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Processing Statistics table
-- Tracks processing performance and accuracy metrics
CREATE TABLE processing_stats (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID NOT NULL REFERENCES bookmark_profiles(id) ON DELETE CASCADE,
    date_period DATE NOT NULL, -- Daily stats
    platform VARCHAR(50),
    category VARCHAR(100),
    total_processed INTEGER DEFAULT 0,
    correctly_categorized INTEGER DEFAULT 0,
    user_corrections INTEGER DEFAULT 0,
    actions_suggested INTEGER DEFAULT 0,
    actions_approved INTEGER DEFAULT 0,
    actions_rejected INTEGER DEFAULT 0,
    processing_time_ms INTEGER DEFAULT 0, -- Average processing time
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Scenario Integration Log table
-- Logs interactions with other scenarios for debugging and analytics
CREATE TABLE scenario_integration_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    profile_id UUID REFERENCES bookmark_profiles(id) ON DELETE SET NULL,
    bookmark_item_id UUID REFERENCES bookmark_items(id) ON DELETE SET NULL,
    action_item_id UUID REFERENCES action_items(id) ON DELETE SET NULL,
    target_scenario VARCHAR(100) NOT NULL,
    integration_type VARCHAR(50) NOT NULL, -- api_call, event_publish, cli_command
    request_data JSONB DEFAULT '{}',
    response_data JSONB DEFAULT '{}',
    success BOOLEAN DEFAULT false,
    error_message TEXT,
    processing_time_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Create indexes for performance
CREATE INDEX idx_bookmark_items_profile_id ON bookmark_items(profile_id);
CREATE INDEX idx_bookmark_items_platform ON bookmark_items(platform);
CREATE INDEX idx_bookmark_items_category ON bookmark_items(category_assigned);
CREATE INDEX idx_bookmark_items_created_at ON bookmark_items(created_at DESC);
CREATE INDEX idx_bookmark_items_hash ON bookmark_items(hash_signature);
CREATE INDEX idx_bookmark_items_status ON bookmark_items(processing_status);

CREATE INDEX idx_action_items_bookmark_id ON action_items(bookmark_item_id);
CREATE INDEX idx_action_items_status ON action_items(approval_status);
CREATE INDEX idx_action_items_scenario ON action_items(target_scenario);
CREATE INDEX idx_action_items_created_at ON action_items(created_at DESC);

CREATE INDEX idx_category_rules_profile_id ON category_rules(profile_id);
CREATE INDEX idx_category_rules_category ON category_rules(category_name);
CREATE INDEX idx_category_rules_priority ON category_rules(priority DESC);
CREATE INDEX idx_category_rules_active ON category_rules(is_active);

CREATE INDEX idx_platform_integrations_profile_id ON platform_integrations(profile_id);
CREATE INDEX idx_platform_integrations_platform ON platform_integrations(platform_name);
CREATE INDEX idx_platform_integrations_status ON platform_integrations(status);

CREATE INDEX idx_processing_stats_profile_date ON processing_stats(profile_id, date_period DESC);
CREATE INDEX idx_processing_stats_platform ON processing_stats(platform);
CREATE INDEX idx_processing_stats_category ON processing_stats(category);

CREATE INDEX idx_integration_log_profile_id ON scenario_integration_log(profile_id);
CREATE INDEX idx_integration_log_scenario ON scenario_integration_log(target_scenario);
CREATE INDEX idx_integration_log_created_at ON scenario_integration_log(created_at DESC);

-- Create triggers for updated_at timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_bookmark_profiles_updated_at BEFORE UPDATE ON bookmark_profiles FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_bookmark_items_updated_at BEFORE UPDATE ON bookmark_items FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_action_items_updated_at BEFORE UPDATE ON action_items FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_category_rules_updated_at BEFORE UPDATE ON category_rules FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_platform_integrations_updated_at BEFORE UPDATE ON platform_integrations FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create views for common queries
CREATE VIEW bookmark_summary AS
SELECT 
    bi.profile_id,
    bi.platform,
    bi.category_assigned,
    COUNT(*) as bookmark_count,
    COUNT(CASE WHEN bi.processing_status = 'processed' THEN 1 END) as processed_count,
    COUNT(CASE WHEN ai.approval_status = 'pending' THEN 1 END) as pending_actions,
    AVG(bi.category_confidence) as avg_confidence
FROM bookmark_items bi
LEFT JOIN action_items ai ON bi.id = ai.bookmark_item_id
GROUP BY bi.profile_id, bi.platform, bi.category_assigned;

CREATE VIEW profile_stats AS
SELECT 
    bp.id as profile_id,
    bp.name as profile_name,
    COUNT(DISTINCT bi.id) as total_bookmarks,
    COUNT(DISTINCT bi.category_assigned) as categories_count,
    COUNT(DISTINCT CASE WHEN ai.approval_status = 'pending' THEN ai.id END) as pending_actions,
    COUNT(DISTINCT pi.platform_name) as connected_platforms,
    COALESCE(AVG(bi.category_confidence), 0) as accuracy_rate,
    MAX(bi.created_at) as last_bookmark_at
FROM bookmark_profiles bp
LEFT JOIN bookmark_items bi ON bp.id = bi.profile_id
LEFT JOIN action_items ai ON bi.id = ai.bookmark_item_id
LEFT JOIN platform_integrations pi ON bp.id = pi.profile_id AND pi.status = 'active'
WHERE bp.is_active = true
GROUP BY bp.id, bp.name;