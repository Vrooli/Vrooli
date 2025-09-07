-- Recommendation Engine Database Schema
-- This schema supports hybrid recommendations using semantic embeddings and behavioral analytics

-- Enable UUID extension
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Users table: tracks users across scenarios
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id VARCHAR(255) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    preferences JSONB DEFAULT '{}',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure unique users per scenario
    UNIQUE(scenario_id, external_id)
);

-- Items table: products/content from any scenario
CREATE TABLE IF NOT EXISTS items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id VARCHAR(255) NOT NULL,
    external_id VARCHAR(255) NOT NULL,
    title VARCHAR(500) NOT NULL,
    description TEXT,
    category VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    embedding_id UUID, -- Reference to Qdrant vector
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Ensure unique items per scenario
    UNIQUE(scenario_id, external_id)
);

-- User interactions table: all user behavior data
CREATE TABLE IF NOT EXISTS user_interactions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    item_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    interaction_type VARCHAR(50) NOT NULL, -- view, like, purchase, share, rate, etc.
    interaction_value FLOAT DEFAULT 1.0,   -- weight/score of interaction
    context JSONB DEFAULT '{}',            -- contextual data (time, device, session, etc.)
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Prevent duplicate interactions within short time window
    CONSTRAINT unique_recent_interaction UNIQUE (user_id, item_id, interaction_type, DATE_TRUNC('minute', timestamp))
);

-- Recommendation events table: track recommendations given to users
CREATE TABLE IF NOT EXISTS recommendation_events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    scenario_id VARCHAR(255) NOT NULL,
    algorithm_used VARCHAR(100) NOT NULL,
    recommended_items JSONB NOT NULL,      -- Array of recommended item IDs with scores
    context JSONB DEFAULT '{}',            -- Request context
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    clicked_items JSONB DEFAULT '[]',      -- Track which recommendations were clicked
    conversion_events JSONB DEFAULT '[]'   -- Track conversions from recommendations
);

-- Algorithm performance table: track A/B test results
CREATE TABLE IF NOT EXISTS algorithm_performance (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    algorithm_name VARCHAR(100) NOT NULL,
    scenario_id VARCHAR(255) NOT NULL,
    metric_name VARCHAR(100) NOT NULL,     -- click_through_rate, conversion_rate, etc.
    metric_value FLOAT NOT NULL,
    sample_size INTEGER NOT NULL,
    test_period_start TIMESTAMP NOT NULL,
    test_period_end TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(algorithm_name, scenario_id, metric_name, test_period_start)
);

-- Scenario configurations table: store scenario-specific recommendation settings
CREATE TABLE IF NOT EXISTS scenario_configs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_id VARCHAR(255) NOT NULL UNIQUE,
    config JSONB NOT NULL DEFAULT '{}',    -- Algorithm weights, thresholds, etc.
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for optimal performance
CREATE INDEX IF NOT EXISTS idx_users_scenario_external ON users(scenario_id, external_id);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users(created_at);

CREATE INDEX IF NOT EXISTS idx_items_scenario_external ON items(scenario_id, external_id);
CREATE INDEX IF NOT EXISTS idx_items_scenario_category ON items(scenario_id, category);
CREATE INDEX IF NOT EXISTS idx_items_created_at ON items(created_at);
CREATE INDEX IF NOT EXISTS idx_items_title_trgm ON items USING gin(title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_items_description_trgm ON items USING gin(description gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_interactions_user ON user_interactions(user_id);
CREATE INDEX IF NOT EXISTS idx_interactions_item ON user_interactions(item_id);
CREATE INDEX IF NOT EXISTS idx_interactions_type ON user_interactions(interaction_type);
CREATE INDEX IF NOT EXISTS idx_interactions_timestamp ON user_interactions(timestamp);
CREATE INDEX IF NOT EXISTS idx_interactions_user_timestamp ON user_interactions(user_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_interactions_item_timestamp ON user_interactions(item_id, timestamp DESC);

CREATE INDEX IF NOT EXISTS idx_recommendations_user ON recommendation_events(user_id);
CREATE INDEX IF NOT EXISTS idx_recommendations_scenario ON recommendation_events(scenario_id);
CREATE INDEX IF NOT EXISTS idx_recommendations_timestamp ON recommendation_events(timestamp);
CREATE INDEX IF NOT EXISTS idx_recommendations_algorithm ON recommendation_events(algorithm_used);

CREATE INDEX IF NOT EXISTS idx_algorithm_performance_name ON algorithm_performance(algorithm_name);
CREATE INDEX IF NOT EXISTS idx_algorithm_performance_scenario ON algorithm_performance(scenario_id);
CREATE INDEX IF NOT EXISTS idx_algorithm_performance_metric ON algorithm_performance(metric_name);

-- Create functions for automatic timestamp updates
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Create triggers for updated_at columns
DROP TRIGGER IF EXISTS update_users_updated_at ON users;
CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_items_updated_at ON items;
CREATE TRIGGER update_items_updated_at 
    BEFORE UPDATE ON items 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS update_scenario_configs_updated_at ON scenario_configs;
CREATE TRIGGER update_scenario_configs_updated_at 
    BEFORE UPDATE ON scenario_configs 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Create views for common queries
CREATE OR REPLACE VIEW user_interaction_summary AS
SELECT 
    u.scenario_id,
    u.external_id as user_external_id,
    COUNT(DISTINCT ui.item_id) as unique_items_interacted,
    COUNT(ui.id) as total_interactions,
    AVG(ui.interaction_value) as avg_interaction_value,
    MAX(ui.timestamp) as last_interaction_time,
    MIN(ui.timestamp) as first_interaction_time
FROM users u
LEFT JOIN user_interactions ui ON u.id = ui.user_id
GROUP BY u.id, u.scenario_id, u.external_id;

CREATE OR REPLACE VIEW item_popularity_summary AS
SELECT 
    i.scenario_id,
    i.external_id as item_external_id,
    i.title,
    i.category,
    COUNT(DISTINCT ui.user_id) as unique_users_interacted,
    COUNT(ui.id) as total_interactions,
    AVG(ui.interaction_value) as avg_interaction_value,
    MAX(ui.timestamp) as last_interaction_time
FROM items i
LEFT JOIN user_interactions ui ON i.id = ui.item_id
GROUP BY i.id, i.scenario_id, i.external_id, i.title, i.category;

CREATE OR REPLACE VIEW scenario_activity_summary AS
SELECT 
    scenario_id,
    COUNT(DISTINCT u.id) as total_users,
    COUNT(DISTINCT i.id) as total_items,
    COUNT(ui.id) as total_interactions,
    COUNT(DISTINCT DATE(ui.timestamp)) as active_days,
    MAX(ui.timestamp) as last_activity_time
FROM (
    SELECT DISTINCT scenario_id FROM users
    UNION 
    SELECT DISTINCT scenario_id FROM items
) scenarios
LEFT JOIN users u ON scenarios.scenario_id = u.scenario_id
LEFT JOIN items i ON scenarios.scenario_id = i.scenario_id
LEFT JOIN user_interactions ui ON u.id = ui.user_id OR i.id = ui.item_id
GROUP BY scenario_id;

-- Insert default configuration for scenarios
INSERT INTO scenario_configs (scenario_id, config) 
VALUES ('default', '{
    "algorithms": {
        "collaborative_filtering": {
            "enabled": true,
            "weight": 0.4,
            "min_interactions": 5
        },
        "content_based": {
            "enabled": true,
            "weight": 0.4,
            "similarity_threshold": 0.7
        },
        "popularity_based": {
            "enabled": true,
            "weight": 0.2,
            "time_decay_days": 30
        }
    },
    "recommendation_limits": {
        "default_count": 10,
        "max_count": 50
    },
    "performance_tracking": {
        "click_through_rate": true,
        "conversion_rate": true,
        "diversity_score": true
    }
}') ON CONFLICT (scenario_id) DO NOTHING;

-- Create materialized view for fast recommendation lookups (refreshed periodically)
CREATE MATERIALIZED VIEW IF NOT EXISTS popular_items_by_category AS
SELECT 
    i.scenario_id,
    i.category,
    i.id as item_id,
    i.external_id,
    i.title,
    COUNT(ui.id) as interaction_count,
    AVG(ui.interaction_value) as avg_score,
    ROW_NUMBER() OVER (PARTITION BY i.scenario_id, i.category ORDER BY COUNT(ui.id) DESC, AVG(ui.interaction_value) DESC) as popularity_rank
FROM items i
LEFT JOIN user_interactions ui ON i.id = ui.item_id
    AND ui.timestamp > CURRENT_TIMESTAMP - INTERVAL '30 days'
GROUP BY i.scenario_id, i.category, i.id, i.external_id, i.title;

-- Create unique index on materialized view for fast lookups
CREATE UNIQUE INDEX IF NOT EXISTS idx_popular_items_unique 
ON popular_items_by_category(scenario_id, category, item_id);

-- Refresh materialized view (should be called periodically via cron or application)
REFRESH MATERIALIZED VIEW popular_items_by_category;