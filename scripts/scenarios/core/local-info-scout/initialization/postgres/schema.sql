-- Local Info Scout Database Schema

CREATE TABLE IF NOT EXISTS search_history (
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

CREATE TABLE IF NOT EXISTS saved_places (
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

CREATE TABLE IF NOT EXISTS user_preferences (
    id SERIAL PRIMARY KEY,
    user_id VARCHAR(255) UNIQUE NOT NULL,
    default_location VARCHAR(255),
    default_radius DECIMAL(5, 2) DEFAULT 2.0,
    favorite_categories TEXT[],
    hidden_categories TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_search_history_user ON search_history(user_id);
CREATE INDEX idx_search_history_created ON search_history(created_at);
CREATE INDEX idx_saved_places_user ON saved_places(user_id);
CREATE INDEX idx_saved_places_category ON saved_places(category);