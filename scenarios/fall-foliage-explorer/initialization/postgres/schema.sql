-- Fall Foliage Explorer Database Schema

-- Regions table for geographic areas
CREATE TABLE IF NOT EXISTS regions (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    state VARCHAR(100),
    country VARCHAR(100) DEFAULT 'USA',
    latitude DECIMAL(10, 8) NOT NULL,
    longitude DECIMAL(11, 8) NOT NULL,
    elevation_meters INTEGER,
    typical_peak_week INTEGER, -- Week number (1-52) when foliage typically peaks
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Foliage observations table
CREATE TABLE IF NOT EXISTS foliage_observations (
    id SERIAL PRIMARY KEY,
    region_id INTEGER REFERENCES regions(id) ON DELETE CASCADE,
    observation_date DATE NOT NULL,
    foliage_percentage INTEGER CHECK (foliage_percentage >= 0 AND foliage_percentage <= 100),
    color_intensity INTEGER CHECK (color_intensity >= 0 AND color_intensity <= 10),
    peak_status VARCHAR(50), -- 'not_started', 'progressing', 'near_peak', 'peak', 'past_peak'
    source VARCHAR(100), -- 'official', 'user_report', 'predicted'
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(region_id, observation_date, source)
);

-- Weather data for predictions
CREATE TABLE IF NOT EXISTS weather_data (
    id SERIAL PRIMARY KEY,
    region_id INTEGER REFERENCES regions(id) ON DELETE CASCADE,
    date DATE NOT NULL,
    temperature_high_c DECIMAL(5, 2),
    temperature_low_c DECIMAL(5, 2),
    precipitation_mm DECIMAL(6, 2),
    humidity_percent INTEGER,
    wind_speed_kmh DECIMAL(5, 2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(region_id, date)
);

-- Predictions table
CREATE TABLE IF NOT EXISTS foliage_predictions (
    id SERIAL PRIMARY KEY,
    region_id INTEGER REFERENCES regions(id) ON DELETE CASCADE,
    prediction_date DATE NOT NULL,
    predicted_peak_date DATE,
    confidence_score DECIMAL(3, 2) CHECK (confidence_score >= 0 AND confidence_score <= 1),
    model_version VARCHAR(50),
    factors JSONB, -- Store factors that influenced the prediction
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User reports table
CREATE TABLE IF NOT EXISTS user_reports (
    id SERIAL PRIMARY KEY,
    region_id INTEGER REFERENCES regions(id) ON DELETE CASCADE,
    report_date DATE NOT NULL,
    foliage_status VARCHAR(50),
    photo_url TEXT,
    description TEXT,
    upvotes INTEGER DEFAULT 0,
    verified BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Trip plans table
CREATE TABLE IF NOT EXISTS trip_plans (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255),
    start_date DATE,
    end_date DATE,
    regions JSONB, -- Array of region IDs with visit dates
    notes TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Indexes for performance
CREATE INDEX idx_observations_region_date ON foliage_observations(region_id, observation_date);
CREATE INDEX idx_weather_region_date ON weather_data(region_id, date);
CREATE INDEX idx_predictions_region ON foliage_predictions(region_id);
CREATE INDEX idx_user_reports_region ON user_reports(region_id);

-- Insert some initial regions (popular foliage destinations)
INSERT INTO regions (name, state, latitude, longitude, elevation_meters, typical_peak_week) VALUES
('White Mountains', 'New Hampshire', 44.2700, -71.3034, 1917, 40),
('Green Mountains', 'Vermont', 43.9207, -72.8986, 1340, 40),
('Adirondacks', 'New York', 44.1127, -74.0524, 1629, 40),
('Great Smoky Mountains', 'Tennessee', 35.6532, -83.5070, 2025, 42),
('Blue Ridge Parkway', 'Virginia', 37.5615, -79.3553, 1200, 42),
('Berkshires', 'Massachusetts', 42.3604, -73.2290, 1064, 41),
('Pocono Mountains', 'Pennsylvania', 41.1247, -75.3821, 748, 41),
('Shenandoah Valley', 'Virginia', 38.4833, -78.8500, 1234, 42),
('Finger Lakes', 'New York', 42.6500, -76.8833, 382, 41),
('Upper Peninsula', 'Michigan', 46.5000, -87.5000, 603, 39)
ON CONFLICT DO NOTHING;