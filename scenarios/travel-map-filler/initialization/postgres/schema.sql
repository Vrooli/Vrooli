-- Travel Map Filler Database Schema

-- Create travels table for storing travel history
CREATE TABLE IF NOT EXISTS travels (
    id BIGINT PRIMARY KEY,
    user_id VARCHAR(255) DEFAULT 'default_user',
    location VARCHAR(500) NOT NULL,
    lat FLOAT NOT NULL,
    lng FLOAT NOT NULL,
    date DATE NOT NULL,
    type VARCHAR(50) NOT NULL,
    notes TEXT,
    country VARCHAR(255),
    city VARCHAR(255),
    continent VARCHAR(100),
    duration_days INTEGER DEFAULT 1,
    rating INTEGER CHECK (rating >= 1 AND rating <= 5),
    photos JSONB DEFAULT '[]',
    tags JSONB DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create achievements table for tracking travel milestones
CREATE TABLE IF NOT EXISTS achievements (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) DEFAULT 'default_user',
    achievement_type VARCHAR(100) NOT NULL,
    achievement_name VARCHAR(255) NOT NULL,
    description TEXT,
    icon VARCHAR(10),
    unlocked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}',
    UNIQUE(user_id, achievement_type)
);

-- Create bucket_list table for future travel plans
CREATE TABLE IF NOT EXISTS bucket_list (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) DEFAULT 'default_user',
    location VARCHAR(500) NOT NULL,
    country VARCHAR(255),
    city VARCHAR(255),
    priority INTEGER DEFAULT 3 CHECK (priority >= 1 AND priority <= 5),
    notes TEXT,
    estimated_date DATE,
    budget_estimate DECIMAL(10, 2),
    tags JSONB DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed BOOLEAN DEFAULT FALSE,
    completed_travel_id BIGINT REFERENCES travels(id)
);

-- Create travel_stats table for aggregated statistics
CREATE TABLE IF NOT EXISTS travel_stats (
    user_id VARCHAR(255) PRIMARY KEY DEFAULT 'default_user',
    total_countries INTEGER DEFAULT 0,
    total_cities INTEGER DEFAULT 0,
    total_continents INTEGER DEFAULT 0,
    total_distance_km FLOAT DEFAULT 0,
    total_days_traveled INTEGER DEFAULT 0,
    favorite_country VARCHAR(255),
    favorite_city VARCHAR(255),
    longest_trip_days INTEGER DEFAULT 0,
    world_coverage_percent FLOAT DEFAULT 0,
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create travel_recommendations table
CREATE TABLE IF NOT EXISTS travel_recommendations (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) DEFAULT 'default_user',
    location VARCHAR(500) NOT NULL,
    country VARCHAR(255),
    reason TEXT,
    score FLOAT,
    tags JSONB DEFAULT '[]',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    dismissed BOOLEAN DEFAULT FALSE
);

-- Create indexes for better query performance
CREATE INDEX IF NOT EXISTS idx_travels_user_date ON travels(user_id, date DESC);
CREATE INDEX IF NOT EXISTS idx_travels_country ON travels(country);
CREATE INDEX IF NOT EXISTS idx_travels_type ON travels(type);
CREATE INDEX IF NOT EXISTS idx_bucket_list_user_priority ON bucket_list(user_id, priority DESC);
CREATE INDEX IF NOT EXISTS idx_achievements_user ON achievements(user_id);

-- Create function to update travel stats
CREATE OR REPLACE FUNCTION update_travel_stats()
RETURNS TRIGGER AS $$
BEGIN
    -- Update aggregated stats for the user
    INSERT INTO travel_stats (
        user_id,
        total_countries,
        total_cities,
        total_continents,
        total_days_traveled
    )
    SELECT 
        NEW.user_id,
        COUNT(DISTINCT country),
        COUNT(DISTINCT city),
        COUNT(DISTINCT continent),
        COALESCE(SUM(duration_days), 0)
    FROM travels
    WHERE user_id = NEW.user_id
    ON CONFLICT (user_id) 
    DO UPDATE SET
        total_countries = EXCLUDED.total_countries,
        total_cities = EXCLUDED.total_cities,
        total_continents = EXCLUDED.total_continents,
        total_days_traveled = EXCLUDED.total_days_traveled,
        last_updated = CURRENT_TIMESTAMP;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger to auto-update stats
CREATE TRIGGER update_stats_on_travel_insert
AFTER INSERT OR UPDATE ON travels
FOR EACH ROW
EXECUTE FUNCTION update_travel_stats();

-- Insert some default achievements definitions
INSERT INTO achievements (user_id, achievement_type, achievement_name, description, icon)
VALUES 
    ('system', 'first_trip', 'First Steps', 'Complete your first trip', '👣'),
    ('system', 'five_countries', 'Explorer', 'Visit 5 different countries', '🧭'),
    ('system', 'ten_countries', 'Adventurer', 'Visit 10 different countries', '🎒'),
    ('system', 'twenty_countries', 'Globetrotter', 'Visit 20 different countries', '🌍'),
    ('system', 'all_continents', 'World Traveler', 'Visit all 7 continents', '🌎'),
    ('system', 'hundred_days', 'Nomad', 'Travel for 100+ days total', '🏕️'),
    ('system', 'ten_cities_year', 'City Hopper', 'Visit 10 cities in one year', '🏙️'),
    ('system', 'bucket_complete', 'Dream Chaser', 'Complete 5 bucket list items', '✨')
ON CONFLICT DO NOTHING;