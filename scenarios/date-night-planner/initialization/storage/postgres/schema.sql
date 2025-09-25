-- Date Night Planner Database Schema
-- Version: 1.0.0
-- Purpose: Store couples, date plans, preferences, and memories

-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS date_night_planner;

-- Use the schema
SET search_path TO date_night_planner, public;

-- UUID extension for generating unique IDs
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Couples table: represents relationship entities
CREATE TABLE IF NOT EXISTS couples (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    auth_user_id_1 UUID, -- Links to scenario-authenticator users
    auth_user_id_2 UUID, -- Links to scenario-authenticator users
    relationship_start DATE,
    shared_preferences JSONB DEFAULT '{}',
    privacy_settings JSONB DEFAULT '{
        "surprise_mode": false,
        "share_memories": true,
        "allow_recommendations": true
    }',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Date Plans table: stores all planned and completed dates
CREATE TABLE IF NOT EXISTS date_plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    couple_id UUID NOT NULL REFERENCES couples(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    planned_date TIMESTAMP WITH TIME ZONE,
    activities JSONB DEFAULT '[]', -- Array of activity objects
    estimated_cost DECIMAL(10, 2),
    estimated_duration INTERVAL,
    weather_backup JSONB,
    status VARCHAR(20) DEFAULT 'planned' CHECK (status IN ('planned', 'completed', 'cancelled')),
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    memory_notes TEXT,
    photos TEXT[], -- URLs to stored photos
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Preferences table: learned and explicit couple preferences
CREATE TABLE IF NOT EXISTS preferences (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    couple_id UUID NOT NULL REFERENCES couples(id) ON DELETE CASCADE,
    category VARCHAR(100) NOT NULL, -- 'cuisine', 'activity_type', 'ambiance', etc.
    preference_key VARCHAR(255) NOT NULL,
    preference_value TEXT NOT NULL,
    confidence_score DECIMAL(3, 2) DEFAULT 0.5 CHECK (confidence_score >= 0 AND confidence_score <= 1),
    learned_from VARCHAR(50) DEFAULT 'explicit' CHECK (learned_from IN ('explicit', 'feedback', 'behavior', 'imported')),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Date Memories table: stores photos and memories from dates
CREATE TABLE IF NOT EXISTS date_memories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    date_plan_id UUID NOT NULL REFERENCES date_plans(id) ON DELETE CASCADE,
    memory_type VARCHAR(50) DEFAULT 'photo' CHECK (memory_type IN ('photo', 'note', 'video', 'receipt')),
    content_url TEXT,
    caption TEXT,
    is_favorite BOOLEAN DEFAULT false,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Venue Cache table: stores discovered venues from local-info-scout
CREATE TABLE IF NOT EXISTS venue_cache (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    venue_name VARCHAR(255) NOT NULL,
    venue_type VARCHAR(100),
    location JSONB NOT NULL, -- {lat, lng, address, city, etc.}
    details JSONB, -- Opening hours, price range, amenities, etc.
    rating DECIMAL(2, 1),
    last_updated TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    expires_at TIMESTAMP WITH TIME ZONE DEFAULT NOW() + INTERVAL '30 days'
);

-- Activity Suggestions table: stores generated suggestions for reuse
CREATE TABLE IF NOT EXISTS activity_suggestions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(100),
    typical_cost_min DECIMAL(10, 2),
    typical_cost_max DECIMAL(10, 2),
    typical_duration INTERVAL,
    weather_requirement VARCHAR(50) CHECK (weather_requirement IN ('indoor', 'outdoor', 'flexible')),
    popularity_score DECIMAL(3, 2) DEFAULT 0.5,
    tags TEXT[],
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Couple Activity History: tracks what couples have done
CREATE TABLE IF NOT EXISTS couple_activity_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    couple_id UUID NOT NULL REFERENCES couples(id) ON DELETE CASCADE,
    activity_suggestion_id UUID REFERENCES activity_suggestions(id),
    date_plan_id UUID REFERENCES date_plans(id),
    completed_at TIMESTAMP WITH TIME ZONE,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    would_repeat BOOLEAN
);

-- Integration Logs: track integration with other scenarios
CREATE TABLE IF NOT EXISTS integration_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    scenario_name VARCHAR(100) NOT NULL,
    action VARCHAR(100) NOT NULL,
    couple_id UUID REFERENCES couples(id),
    request_data JSONB,
    response_data JSONB,
    status VARCHAR(20) DEFAULT 'success' CHECK (status IN ('success', 'failure', 'fallback')),
    error_message TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_couples_auth_users ON couples(auth_user_id_1, auth_user_id_2);
CREATE INDEX IF NOT EXISTS idx_date_plans_couple_id ON date_plans(couple_id);
CREATE INDEX IF NOT EXISTS idx_date_plans_status ON date_plans(status);
CREATE INDEX IF NOT EXISTS idx_date_plans_planned_date ON date_plans(planned_date);
CREATE INDEX IF NOT EXISTS idx_preferences_couple_id ON preferences(couple_id);
CREATE INDEX IF NOT EXISTS idx_preferences_category ON preferences(category);
CREATE INDEX IF NOT EXISTS idx_venue_cache_expires ON venue_cache(expires_at);
CREATE INDEX IF NOT EXISTS idx_activity_suggestions_category ON activity_suggestions(category);
CREATE INDEX IF NOT EXISTS idx_couple_activity_history_couple ON couple_activity_history(couple_id);
CREATE INDEX IF NOT EXISTS idx_integration_logs_scenario ON integration_logs(scenario_name, created_at);

-- Functions
-- Update timestamp function
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for updated_at
CREATE TRIGGER update_couples_updated_at BEFORE UPDATE ON couples
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_date_plans_updated_at BEFORE UPDATE ON date_plans
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER update_preferences_updated_at BEFORE UPDATE ON preferences
    FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- Function to increase preference confidence based on successful dates
CREATE OR REPLACE FUNCTION increase_preference_confidence(
    p_couple_id UUID,
    p_category VARCHAR,
    p_key VARCHAR,
    p_increase DECIMAL DEFAULT 0.1
)
RETURNS VOID AS $$
BEGIN
    UPDATE preferences 
    SET confidence_score = LEAST(confidence_score + p_increase, 1.0)
    WHERE couple_id = p_couple_id 
      AND category = p_category 
      AND preference_key = p_key;
END;
$$ LANGUAGE plpgsql;

-- Function to get top date suggestions for a couple
CREATE OR REPLACE FUNCTION get_date_suggestions(
    p_couple_id UUID,
    p_limit INTEGER DEFAULT 5
)
RETURNS TABLE (
    suggestion_id UUID,
    title VARCHAR,
    description TEXT,
    estimated_cost DECIMAL,
    confidence_score DECIMAL
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        s.id,
        s.title,
        s.description,
        (s.typical_cost_min + s.typical_cost_max) / 2 as estimated_cost,
        COALESCE(
            (SELECT AVG(p.confidence_score) 
             FROM preferences p 
             WHERE p.couple_id = p_couple_id 
               AND p.category = s.category),
            0.5
        ) as confidence_score
    FROM activity_suggestions s
    WHERE s.id NOT IN (
        SELECT activity_suggestion_id 
        FROM couple_activity_history 
        WHERE couple_id = p_couple_id 
          AND completed_at > NOW() - INTERVAL '30 days'
    )
    ORDER BY confidence_score DESC, s.popularity_score DESC
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- Add comments for documentation
COMMENT ON TABLE couples IS 'Stores couple relationships and shared preferences';
COMMENT ON TABLE date_plans IS 'Stores planned and completed date activities';
COMMENT ON TABLE preferences IS 'Stores learned and explicit couple preferences';
COMMENT ON TABLE date_memories IS 'Stores photos and memories from completed dates';
COMMENT ON TABLE venue_cache IS 'Caches venue information from local-info-scout integration';
COMMENT ON TABLE activity_suggestions IS 'Stores reusable activity suggestions';
COMMENT ON TABLE couple_activity_history IS 'Tracks what activities couples have done';
COMMENT ON TABLE integration_logs IS 'Logs interactions with other scenarios for debugging';