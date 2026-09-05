
-- Local Info Scout Database Schema

CREATE TABLE IF NOT EXISTS lis_search_history (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255),
    query TEXT NOT NULL,
    lat DECIMAL(10, 8),
    lon DECIMAL(11, 8),
    radius DECIMAL(5, 2),
    category VARCHAR(100),
    results_count INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS lis_saved_places (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255),
    place_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    address TEXT,
    category VARCHAR(100),
    lat DECIMAL(10, 8),
    lon DECIMAL(11, 8),
    notes TEXT,
    rating DECIMAL(2, 1),
    saved_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(user_id, place_id)
);

CREATE TABLE IF NOT EXISTS lis_user_preferences (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) UNIQUE NOT NULL,
    default_location VARCHAR(255),
    default_radius DECIMAL(5, 2) DEFAULT 2.0,
    favorite_categories TEXT[],
    hidden_categories TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_lis_search_history_user ON lis_search_history(user_id);
CREATE INDEX IF NOT EXISTS idx_lis_search_history_created ON lis_search_history(created_at);
CREATE INDEX IF NOT EXISTS idx_lis_saved_places_user ON lis_saved_places(user_id);
CREATE INDEX IF NOT EXISTS idx_lis_saved_places_category ON lis_saved_places(category);

CREATE TABLE IF NOT EXISTS lis_places (
    id VARCHAR(50) PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    address VARCHAR(500),
    category VARCHAR(50),
    lat DOUBLE PRECISION,
    lon DOUBLE PRECISION,
    rating DECIMAL(2,1),
    price_level INTEGER,
    open_now BOOLEAN,
    description TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_lis_places_category ON lis_places(category);
CREATE INDEX IF NOT EXISTS idx_lis_places_location ON lis_places(lat, lon);
CREATE INDEX IF NOT EXISTS idx_lis_places_rating ON lis_places(rating);

CREATE TABLE IF NOT EXISTS lis_search_logs (
    id SERIAL PRIMARY KEY,
    query TEXT,
    lat DOUBLE PRECISION,
    lon DOUBLE PRECISION,
    radius DOUBLE PRECISION,
    category VARCHAR(50),
    results_count INTEGER,
    cache_hit BOOLEAN,
    search_time_ms INTEGER,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_lis_search_logs_created ON lis_search_logs(created_at);
