-- Retro Game Launcher Database Schema
-- Store games, remixes, and player data

-- User accounts
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    subscription_tier VARCHAR(20) DEFAULT 'free' CHECK (subscription_tier IN ('free', 'premium', 'pro')),
    games_created_this_month INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_login TIMESTAMP
);

-- Game library
CREATE TABLE games (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    prompt TEXT NOT NULL,
    description TEXT,
    code TEXT NOT NULL,
    engine VARCHAR(50) DEFAULT 'javascript' CHECK (engine IN ('javascript', 'pico8', 'tic80')),
    author_id UUID REFERENCES users(id),
    parent_game_id UUID REFERENCES games(id),
    is_remix BOOLEAN DEFAULT false,
    thumbnail_url TEXT,
    play_count INTEGER DEFAULT 0,
    remix_count INTEGER DEFAULT 0,
    rating DECIMAL(3,2),
    tags TEXT[],
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    published BOOLEAN DEFAULT true
);

-- Game assets
CREATE TABLE game_assets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID REFERENCES games(id) ON DELETE CASCADE,
    asset_type VARCHAR(50) CHECK (asset_type IN ('sprite', 'sound', 'music', 'tilemap', 'font')),
    name VARCHAR(255) NOT NULL,
    data TEXT,
    url TEXT,
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- High scores
CREATE TABLE high_scores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID REFERENCES games(id) ON DELETE CASCADE,
    user_id UUID REFERENCES users(id),
    score INTEGER NOT NULL,
    play_time_seconds INTEGER,
    achieved_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB
);

-- Game sessions
CREATE TABLE game_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    game_id UUID REFERENCES games(id),
    user_id UUID REFERENCES users(id),
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    final_score INTEGER,
    progress_data JSONB
);

-- Prompt templates and examples
CREATE TABLE prompt_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    category VARCHAR(100) NOT NULL,
    template TEXT NOT NULL,
    example_prompt TEXT,
    example_game_id UUID REFERENCES games(id),
    usage_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- User favorites
CREATE TABLE favorites (
    user_id UUID REFERENCES users(id),
    game_id UUID REFERENCES games(id),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, game_id)
);

-- Generation history
CREATE TABLE generation_history (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id),
    prompt TEXT NOT NULL,
    game_id UUID REFERENCES games(id),
    success BOOLEAN,
    error_message TEXT,
    generation_time_ms INTEGER,
    model_used VARCHAR(100),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes
CREATE INDEX idx_games_author ON games(author_id);
CREATE INDEX idx_games_parent ON games(parent_game_id);
CREATE INDEX idx_games_play_count ON games(play_count DESC);
CREATE INDEX idx_games_created ON games(created_at DESC);
CREATE INDEX idx_games_tags ON games USING GIN(tags);
CREATE INDEX idx_high_scores_game ON high_scores(game_id, score DESC);
CREATE INDEX idx_sessions_user ON game_sessions(user_id);
CREATE INDEX idx_generation_user ON generation_history(user_id);