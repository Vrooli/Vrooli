-- Elo Swipe Database Schema
-- Universal ranking engine using Elo ratings

-- Create schema if not exists
CREATE SCHEMA IF NOT EXISTS elo_swipe;
SET search_path TO elo_swipe, public;

-- Extension for UUID generation
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Lists table: Container for items to be ranked
CREATE TABLE IF NOT EXISTS lists (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    description TEXT,
    owner_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}',
    settings JSONB DEFAULT '{"k_factor": 32, "initial_rating": 1500}',
    CONSTRAINT unique_list_name_per_owner UNIQUE(name, owner_id)
);

-- Items table: Things being ranked
CREATE TABLE IF NOT EXISTS items (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    list_id UUID NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    content JSONB NOT NULL,
    elo_rating FLOAT DEFAULT 1500,
    comparison_count INTEGER DEFAULT 0,
    wins INTEGER DEFAULT 0,
    losses INTEGER DEFAULT 0,
    confidence_score FLOAT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT unique_item_per_list UNIQUE(list_id, content)
);

-- Comparisons table: History of all comparisons
CREATE TABLE IF NOT EXISTS comparisons (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    list_id UUID NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    winner_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    loser_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    winner_rating_before FLOAT NOT NULL,
    loser_rating_before FLOAT NOT NULL,
    winner_rating_after FLOAT NOT NULL,
    loser_rating_after FLOAT NOT NULL,
    k_factor FLOAT DEFAULT 32,
    user_id VARCHAR(255),
    session_id VARCHAR(255),
    comparison_time_ms INTEGER,
    skipped BOOLEAN DEFAULT FALSE,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT no_self_comparison CHECK (winner_id != loser_id)
);

-- Pairing queue table: Smart pairing suggestions
CREATE TABLE IF NOT EXISTS pairing_queue (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    list_id UUID NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    item_a_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    item_b_id UUID NOT NULL REFERENCES items(id) ON DELETE CASCADE,
    priority FLOAT DEFAULT 0,
    uncertainty_score FLOAT DEFAULT 0,
    suggested_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed BOOLEAN DEFAULT FALSE,
    CONSTRAINT unique_pairing UNIQUE(list_id, item_a_id, item_b_id),
    CONSTRAINT no_duplicate_pairing CHECK (item_a_id < item_b_id)
);

-- Sessions table: Track ranking sessions
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    list_id UUID NOT NULL REFERENCES lists(id) ON DELETE CASCADE,
    user_id VARCHAR(255),
    started_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    ended_at TIMESTAMP,
    comparisons_made INTEGER DEFAULT 0,
    session_metadata JSONB DEFAULT '{}'
);

-- Indexes for performance
CREATE INDEX idx_items_list_id ON items(list_id);
CREATE INDEX idx_items_rating ON items(list_id, elo_rating DESC);
CREATE INDEX idx_comparisons_list_id ON comparisons(list_id);
CREATE INDEX idx_comparisons_timestamp ON comparisons(timestamp DESC);
CREATE INDEX idx_comparisons_participants ON comparisons(winner_id, loser_id);
CREATE INDEX idx_pairing_queue_priority ON pairing_queue(list_id, priority DESC) WHERE NOT completed;
CREATE INDEX idx_sessions_user ON sessions(user_id, started_at DESC);

-- Function to update timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updating timestamps
CREATE TRIGGER update_lists_updated_at BEFORE UPDATE ON lists
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_items_updated_at BEFORE UPDATE ON items
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to calculate Elo rating changes
CREATE OR REPLACE FUNCTION calculate_elo_change(
    winner_rating FLOAT,
    loser_rating FLOAT,
    k_factor FLOAT DEFAULT 32
) RETURNS TABLE(winner_new FLOAT, loser_new FLOAT) AS $$
DECLARE
    expected_winner FLOAT;
    expected_loser FLOAT;
BEGIN
    -- Calculate expected scores
    expected_winner := 1.0 / (1.0 + POWER(10, (loser_rating - winner_rating) / 400.0));
    expected_loser := 1.0 - expected_winner;
    
    -- Calculate new ratings
    winner_new := winner_rating + k_factor * (1.0 - expected_winner);
    loser_new := loser_rating + k_factor * (0.0 - expected_loser);
    
    RETURN NEXT;
END;
$$ LANGUAGE plpgsql;

-- Function to get next optimal comparison
CREATE OR REPLACE FUNCTION get_next_comparison(
    p_list_id UUID
) RETURNS TABLE(
    item_a_id UUID,
    item_b_id UUID,
    priority FLOAT
) AS $$
BEGIN
    -- First try to get from pairing queue
    RETURN QUERY
    SELECT pq.item_a_id, pq.item_b_id, pq.priority
    FROM pairing_queue pq
    WHERE pq.list_id = p_list_id AND NOT pq.completed
    ORDER BY pq.priority DESC
    LIMIT 1;
    
    -- If no queued pairings, find items with lowest comparison count
    IF NOT FOUND THEN
        RETURN QUERY
        WITH item_pairs AS (
            SELECT 
                i1.id as item_a_id,
                i2.id as item_b_id,
                ABS(i1.elo_rating - i2.elo_rating) as rating_diff,
                (i1.comparison_count + i2.comparison_count) as total_comparisons
            FROM items i1
            CROSS JOIN items i2
            WHERE i1.list_id = p_list_id
                AND i2.list_id = p_list_id
                AND i1.id < i2.id
                AND NOT EXISTS (
                    SELECT 1 FROM comparisons c
                    WHERE c.list_id = p_list_id
                        AND ((c.winner_id = i1.id AND c.loser_id = i2.id)
                            OR (c.winner_id = i2.id AND c.loser_id = i1.id))
                )
        )
        SELECT 
            ip.item_a_id,
            ip.item_b_id,
            (1000.0 - ip.rating_diff) / (ip.total_comparisons + 1) as priority
        FROM item_pairs ip
        ORDER BY priority DESC
        LIMIT 1;
    END IF;
END;
$$ LANGUAGE plpgsql;

-- View for current rankings
CREATE OR REPLACE VIEW current_rankings AS
SELECT 
    i.list_id,
    i.id as item_id,
    i.content,
    i.elo_rating,
    i.comparison_count,
    i.wins,
    i.losses,
    i.confidence_score,
    RANK() OVER (PARTITION BY i.list_id ORDER BY i.elo_rating DESC) as rank
FROM items i
ORDER BY i.list_id, rank;

-- View for list statistics
CREATE OR REPLACE VIEW list_statistics AS
SELECT 
    l.id as list_id,
    l.name,
    COUNT(DISTINCT i.id) as item_count,
    COUNT(DISTINCT c.id) as comparison_count,
    AVG(i.comparison_count) as avg_comparisons_per_item,
    MAX(i.elo_rating) as highest_rating,
    MIN(i.elo_rating) as lowest_rating,
    AVG(i.elo_rating) as average_rating,
    STDDEV(i.elo_rating) as rating_stddev
FROM lists l
LEFT JOIN items i ON l.id = i.list_id
LEFT JOIN comparisons c ON l.id = c.list_id
GROUP BY l.id, l.name;

-- Initial data for testing
INSERT INTO lists (name, description, owner_id) 
VALUES ('Sample Priority List', 'Example list for testing Elo Swipe', 'system')
ON CONFLICT DO NOTHING;