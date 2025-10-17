-- Date Night Planner Seed Data
-- Version: 1.0.0
-- Purpose: Initial activity suggestions and categories

SET search_path TO date_night_planner, public;

-- Insert default activity suggestions
INSERT INTO activity_suggestions (title, description, category, typical_cost_min, typical_cost_max, typical_duration, weather_requirement, popularity_score, tags)
VALUES 
    -- Romantic dinners
    ('Candlelit Dinner', 'Intimate dinner at a fine dining restaurant with romantic ambiance', 'dining', 60, 150, '2 hours', 'indoor', 0.9, ARRAY['romantic', 'dinner', 'fine-dining', 'indoor']),
    ('Rooftop Restaurant', 'Dining with a view at a rooftop venue', 'dining', 50, 120, '2 hours', 'flexible', 0.85, ARRAY['romantic', 'dinner', 'scenic', 'rooftop']),
    ('Wine Tasting', 'Sample wines at a local vineyard or wine bar', 'dining', 40, 80, '2 hours', 'flexible', 0.75, ARRAY['wine', 'tasting', 'sophisticated', 'social']),
    ('Cooking Class', 'Learn to cook a new cuisine together', 'dining', 60, 100, '3 hours', 'indoor', 0.8, ARRAY['interactive', 'cooking', 'learning', 'fun']),
    
    -- Adventure activities
    ('Hiking Trail', 'Explore scenic hiking trails together', 'adventure', 0, 20, '3 hours', 'outdoor', 0.7, ARRAY['outdoor', 'hiking', 'nature', 'fitness']),
    ('Escape Room', 'Solve puzzles together in an escape room', 'adventure', 30, 50, '1 hour', 'indoor', 0.85, ARRAY['puzzle', 'teamwork', 'fun', 'indoor']),
    ('Rock Climbing', 'Indoor or outdoor rock climbing experience', 'adventure', 25, 60, '2 hours', 'flexible', 0.65, ARRAY['climbing', 'fitness', 'adventure', 'challenging']),
    ('Kayaking', 'Paddle together on a lake or river', 'adventure', 40, 80, '2 hours', 'outdoor', 0.6, ARRAY['water', 'outdoor', 'adventure', 'scenic']),
    
    -- Cultural experiences
    ('Art Gallery', 'Browse art exhibitions and discuss favorites', 'cultural', 0, 30, '2 hours', 'indoor', 0.75, ARRAY['art', 'culture', 'indoor', 'sophisticated']),
    ('Live Theater', 'Watch a play or musical performance', 'cultural', 40, 120, '3 hours', 'indoor', 0.8, ARRAY['theater', 'entertainment', 'culture', 'evening']),
    ('Museum Visit', 'Explore history, science, or specialty museums', 'cultural', 10, 30, '2 hours', 'indoor', 0.7, ARRAY['museum', 'learning', 'culture', 'indoor']),
    ('Concert', 'Enjoy live music from local bands or orchestras', 'cultural', 30, 150, '3 hours', 'indoor', 0.85, ARRAY['music', 'live', 'entertainment', 'evening']),
    
    -- Casual dates
    ('Coffee Shop', 'Relax with coffee and conversation', 'casual', 10, 25, '1 hour', 'indoor', 0.9, ARRAY['coffee', 'casual', 'conversation', 'morning']),
    ('Picnic in Park', 'Outdoor picnic with homemade or takeout food', 'casual', 20, 40, '2 hours', 'outdoor', 0.8, ARRAY['picnic', 'outdoor', 'romantic', 'nature']),
    ('Movie Night', 'Watch a film at the cinema or drive-in', 'casual', 15, 30, '2 hours', 'flexible', 0.85, ARRAY['movie', 'entertainment', 'casual', 'evening']),
    ('Board Game Café', 'Play board games over drinks and snacks', 'casual', 20, 40, '2 hours', 'indoor', 0.75, ARRAY['games', 'casual', 'fun', 'social']),
    
    -- Unique experiences
    ('Star Gazing', 'Watch stars at an observatory or dark sky location', 'unique', 0, 30, '2 hours', 'outdoor', 0.65, ARRAY['stars', 'romantic', 'outdoor', 'night']),
    ('Dance Class', 'Learn salsa, swing, or ballroom dancing', 'unique', 30, 60, '1.5 hours', 'indoor', 0.7, ARRAY['dancing', 'active', 'fun', 'social']),
    ('Spa Day', 'Couples massage and relaxation treatments', 'unique', 100, 250, '3 hours', 'indoor', 0.75, ARRAY['spa', 'relaxation', 'luxury', 'wellness']),
    ('Hot Air Balloon', 'Scenic hot air balloon ride', 'unique', 150, 300, '3 hours', 'outdoor', 0.5, ARRAY['balloon', 'scenic', 'adventure', 'special']);

-- Insert sample couple for testing (will be replaced with real data from scenario-authenticator)
INSERT INTO couples (id, relationship_start, shared_preferences)
VALUES 
    ('00000000-0000-0000-0000-000000000001', '2023-01-01', 
     '{"favorite_cuisine": "Italian", "budget_range": "moderate", "preferred_time": "evening"}');

-- Insert sample preferences for test couple
INSERT INTO preferences (couple_id, category, preference_key, preference_value, confidence_score, learned_from)
VALUES 
    ('00000000-0000-0000-0000-000000000001', 'dining', 'cuisine_type', 'Italian', 0.9, 'explicit'),
    ('00000000-0000-0000-0000-000000000001', 'dining', 'ambiance', 'romantic', 0.8, 'explicit'),
    ('00000000-0000-0000-0000-000000000001', 'adventure', 'intensity', 'moderate', 0.7, 'behavior'),
    ('00000000-0000-0000-0000-000000000001', 'cultural', 'interest', 'art', 0.6, 'feedback'),
    ('00000000-0000-0000-0000-000000000001', 'timing', 'preferred_day', 'Saturday', 0.85, 'behavior'),
    ('00000000-0000-0000-0000-000000000001', 'budget', 'range', 'moderate', 0.75, 'explicit');

-- Grant permissions for the application user (adjust as needed)
GRANT ALL PRIVILEGES ON SCHEMA date_night_planner TO postgres;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA date_night_planner TO postgres;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA date_night_planner TO postgres;
GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA date_night_planner TO postgres;