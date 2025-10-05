-- Typing Test Database Schema

-- Players table
CREATE TABLE IF NOT EXISTS players (
    id SERIAL PRIMARY KEY,
    name VARCHAR(50) NOT NULL UNIQUE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_played TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    total_games INTEGER DEFAULT 0,
    best_wpm INTEGER DEFAULT 0,
    best_accuracy DECIMAL(5,2) DEFAULT 0,
    total_score BIGINT DEFAULT 0
);

-- Create index for player lookups
CREATE INDEX IF NOT EXISTS idx_players_name ON players(name);

-- Scores table
CREATE TABLE IF NOT EXISTS scores (
    id SERIAL PRIMARY KEY,
    player_id INTEGER REFERENCES players(id) ON DELETE CASCADE,
    name VARCHAR(50) NOT NULL,
    score INTEGER NOT NULL,
    wpm INTEGER NOT NULL,
    accuracy DECIMAL(5,2) NOT NULL,
    max_combo INTEGER DEFAULT 0,
    difficulty VARCHAR(20) NOT NULL,
    mode VARCHAR(20) NOT NULL,
    total_chars INTEGER DEFAULT 0,
    error_count INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for leaderboard queries
CREATE INDEX IF NOT EXISTS idx_scores_created_at ON scores(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_scores_score ON scores(score DESC);
CREATE INDEX IF NOT EXISTS idx_scores_wpm ON scores(wpm DESC);
CREATE INDEX IF NOT EXISTS idx_scores_difficulty ON scores(difficulty);
CREATE INDEX IF NOT EXISTS idx_scores_mode ON scores(mode);

-- Challenges table for AI-generated typing challenges
CREATE TABLE IF NOT EXISTS challenges (
    id SERIAL PRIMARY KEY,
    text TEXT NOT NULL,
    difficulty VARCHAR(20) NOT NULL,
    category VARCHAR(50),
    source VARCHAR(100),
    word_count INTEGER,
    character_count INTEGER,
    avg_word_length DECIMAL(4,2),
    complexity_score DECIMAL(3,2),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    used_count INTEGER DEFAULT 0,
    avg_completion_time INTEGER,
    avg_accuracy DECIMAL(5,2)
);

-- Create indexes for challenge queries
CREATE INDEX IF NOT EXISTS idx_challenges_difficulty ON challenges(difficulty);
CREATE INDEX IF NOT EXISTS idx_challenges_category ON challenges(category);
CREATE INDEX IF NOT EXISTS idx_challenges_used_count ON challenges(used_count);

-- Typing patterns table for AI analysis
CREATE TABLE IF NOT EXISTS typing_patterns (
    id SERIAL PRIMARY KEY,
    player_id INTEGER REFERENCES players(id) ON DELETE CASCADE,
    common_mistakes JSONB,
    slow_keys JSONB,
    fast_keys JSONB,
    avg_pause_time DECIMAL(6,3),
    consistency_score DECIMAL(3,2),
    improvement_rate DECIMAL(5,2),
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Game sessions table for detailed tracking
CREATE TABLE IF NOT EXISTS game_sessions (
    id SERIAL PRIMARY KEY,
    player_id INTEGER REFERENCES players(id) ON DELETE CASCADE,
    challenge_id INTEGER REFERENCES challenges(id) ON DELETE SET NULL,
    session_data JSONB,
    keystroke_data JSONB,
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    duration_seconds INTEGER,
    pauses_count INTEGER DEFAULT 0
);

-- Daily stats aggregation table
CREATE TABLE IF NOT EXISTS daily_stats (
    id SERIAL PRIMARY KEY,
    date DATE NOT NULL UNIQUE,
    total_games INTEGER DEFAULT 0,
    unique_players INTEGER DEFAULT 0,
    avg_wpm DECIMAL(5,2),
    avg_accuracy DECIMAL(5,2),
    top_score INTEGER,
    top_player VARCHAR(50),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create index for daily stats
CREATE INDEX IF NOT EXISTS idx_daily_stats_date ON daily_stats(date DESC);

-- Achievements table
CREATE TABLE IF NOT EXISTS achievements (
    id SERIAL PRIMARY KEY,
    player_id INTEGER REFERENCES players(id) ON DELETE CASCADE,
    achievement_type VARCHAR(50) NOT NULL,
    achievement_name VARCHAR(100) NOT NULL,
    description TEXT,
    unlocked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create composite index for achievement lookups
CREATE INDEX IF NOT EXISTS idx_achievements_player_type ON achievements(player_id, achievement_type);

-- View for current leaderboard
CREATE OR REPLACE VIEW leaderboard_current AS
SELECT 
    ROW_NUMBER() OVER (ORDER BY s.score DESC) as rank,
    s.name,
    s.score,
    s.wpm,
    s.accuracy,
    s.max_combo,
    s.difficulty,
    s.mode,
    s.created_at
FROM scores s
ORDER BY s.score DESC
LIMIT 100;

-- View for daily leaderboard
CREATE OR REPLACE VIEW leaderboard_daily AS
SELECT 
    ROW_NUMBER() OVER (ORDER BY s.score DESC) as rank,
    s.name,
    s.score,
    s.wpm,
    s.accuracy,
    s.max_combo,
    s.difficulty,
    s.mode,
    s.created_at
FROM scores s
WHERE DATE(s.created_at) = CURRENT_DATE
ORDER BY s.score DESC
LIMIT 100;

-- View for weekly leaderboard
CREATE OR REPLACE VIEW leaderboard_weekly AS
SELECT 
    ROW_NUMBER() OVER (ORDER BY s.score DESC) as rank,
    s.name,
    s.score,
    s.wpm,
    s.accuracy,
    s.max_combo,
    s.difficulty,
    s.mode,
    s.created_at
FROM scores s
WHERE s.created_at >= CURRENT_DATE - INTERVAL '7 days'
ORDER BY s.score DESC
LIMIT 100;

-- Function to update player stats after new score
CREATE OR REPLACE FUNCTION update_player_stats()
RETURNS TRIGGER AS $$
BEGIN
    -- Update or insert player record
    INSERT INTO players (name, last_played, total_games, best_wpm, best_accuracy, total_score)
    VALUES (NEW.name, NOW(), 1, NEW.wpm, NEW.accuracy, NEW.score)
    ON CONFLICT (name) DO UPDATE SET
        last_played = NOW(),
        total_games = players.total_games + 1,
        best_wpm = GREATEST(players.best_wpm, NEW.wpm),
        best_accuracy = GREATEST(players.best_accuracy, NEW.accuracy),
        total_score = players.total_score + NEW.score;
    
    -- Set player_id in the scores record
    NEW.player_id = (SELECT id FROM players WHERE name = NEW.name LIMIT 1);
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update player stats
CREATE TRIGGER update_player_stats_trigger
BEFORE INSERT ON scores
FOR EACH ROW
EXECUTE FUNCTION update_player_stats();

-- Function to update daily stats
CREATE OR REPLACE FUNCTION update_daily_stats()
RETURNS TRIGGER AS $$
BEGIN
    INSERT INTO daily_stats (date, total_games, unique_players, avg_wpm, avg_accuracy, top_score, top_player)
    VALUES (
        CURRENT_DATE,
        1,
        1,
        NEW.wpm,
        NEW.accuracy,
        NEW.score,
        NEW.name
    )
    ON CONFLICT (date) DO UPDATE SET
        total_games = daily_stats.total_games + 1,
        unique_players = (
            SELECT COUNT(DISTINCT name) 
            FROM scores 
            WHERE DATE(created_at) = CURRENT_DATE
        ),
        avg_wpm = (
            SELECT AVG(wpm) 
            FROM scores 
            WHERE DATE(created_at) = CURRENT_DATE
        ),
        avg_accuracy = (
            SELECT AVG(accuracy) 
            FROM scores 
            WHERE DATE(created_at) = CURRENT_DATE
        ),
        top_score = GREATEST(daily_stats.top_score, NEW.score),
        top_player = CASE 
            WHEN NEW.score > daily_stats.top_score THEN NEW.name 
            ELSE daily_stats.top_player 
        END;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Trigger to update daily stats
CREATE TRIGGER update_daily_stats_trigger
AFTER INSERT ON scores
FOR EACH ROW
EXECUTE FUNCTION update_daily_stats();

-- Insert some initial challenges
INSERT INTO challenges (text, difficulty, category, word_count, character_count, avg_word_length, complexity_score) VALUES
('The quick brown fox jumps over the lazy dog.', 'easy', 'pangram', 9, 44, 4.9, 0.3),
('Practice makes perfect when learning to type.', 'easy', 'motivational', 7, 43, 6.1, 0.3),
('Technology advances rapidly in the modern world.', 'easy', 'technology', 7, 46, 6.6, 0.4),
('The implementation of sophisticated algorithms requires careful planning.', 'medium', 'technical', 8, 71, 8.9, 0.7),
('Quantum computing revolutionizes data processing capabilities.', 'medium', 'technology', 6, 59, 9.8, 0.8),
('The juxtaposition of contemporary architectural paradigms exemplifies aesthetic evolution.', 'hard', 'academic', 9, 87, 9.7, 0.9),
('Cryptographic protocols ensure information security in distributed systems.', 'hard', 'technical', 8, 73, 9.1, 0.9);