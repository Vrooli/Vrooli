-- Bookmark Intelligence Hub Seed Data
-- Version: 1.0.0
-- Created: 2025-09-06

-- Insert default profile for demo purposes
INSERT INTO bookmark_profiles (
    id,
    user_id,
    name,
    description,
    platform_configs,
    categorization_rules,
    integration_settings
) VALUES (
    uuid_generate_v4(),
    uuid_generate_v4(),
    'Demo Profile',
    'Default profile for demonstration purposes',
    '{
        "reddit": {
            "enabled": true,
            "sync_frequency": "30m",
            "subreddits_to_monitor": ["all"]
        },
        "twitter": {
            "enabled": true,
            "sync_frequency": "1h",
            "include_retweets": false
        },
        "tiktok": {
            "enabled": true,
            "sync_frequency": "2h",
            "content_types": ["favorites", "liked"]
        }
    }',
    '{
        "default_categories": ["Programming", "Recipes", "Fitness", "Travel", "News", "Education", "Entertainment", "Misc"]
    }',
    '{
        "auto_approve_high_confidence": true,
        "confidence_threshold": 0.85,
        "enabled_scenarios": ["recipe-book", "workout-plan-generator", "research-assistant"]
    }'
) ON CONFLICT DO NOTHING;

-- Insert default category rules
INSERT INTO category_rules (profile_id, category_name, rule_type, keywords, patterns, integration_actions, priority) 
SELECT 
    bp.id,
    category_data.category_name,
    category_data.rule_type,
    category_data.keywords,
    category_data.patterns,
    category_data.integration_actions,
    category_data.priority
FROM bookmark_profiles bp,
(VALUES 
    ('Programming', 'keyword', ARRAY['code', 'programming', 'developer', 'coding', 'software', 'algorithm', 'javascript', 'python', 'react', 'api', 'github', 'stackoverflow'], '{"platforms": ["reddit"], "subreddits": ["r/programming", "r/webdev", "r/javascript"]}', '[{"scenario": "code-library", "action": "add_snippet"}]', 10),
    
    ('Recipes', 'keyword', ARRAY['recipe', 'cooking', 'food', 'baking', 'ingredients', 'cook', 'delicious', 'meal', 'kitchen', 'chef'], '{"platforms": ["reddit", "tiktok"], "subreddits": ["r/recipes", "r/cooking", "r/baking"]}', '[{"scenario": "recipe-book", "action": "add_recipe"}]', 9),
    
    ('Fitness', 'keyword', ARRAY['workout', 'fitness', 'exercise', 'gym', 'training', 'muscle', 'cardio', 'strength', 'yoga', 'running'], '{"platforms": ["reddit", "tiktok"], "subreddits": ["r/fitness", "r/bodyweightfitness"]}', '[{"scenario": "workout-plan-generator", "action": "add_exercise"}]', 9),
    
    ('Travel', 'keyword', ARRAY['travel', 'trip', 'vacation', 'destination', 'flight', 'hotel', 'backpack', 'adventure', 'tourism', 'explore'], '{"platforms": ["reddit", "twitter"], "subreddits": ["r/travel", "r/solotravel"]}', '[{"scenario": "travel-planner", "action": "add_destination"}]', 8),
    
    ('Education', 'keyword', ARRAY['learn', 'education', 'tutorial', 'course', 'study', 'knowledge', 'university', 'research', 'academic', 'science'], '{"platforms": ["reddit", "twitter"], "subreddits": ["r/todayilearned", "r/explainlikeimfive"]}', '[{"scenario": "research-assistant", "action": "add_reference"}]', 7),
    
    ('News', 'keyword', ARRAY['news', 'breaking', 'politics', 'world', 'economy', 'business', 'current events', 'headline'], '{"platforms": ["reddit", "twitter"], "subreddits": ["r/news", "r/worldnews"]}', '[{"scenario": "news-aggregator", "action": "add_article"}]', 6),
    
    ('Entertainment', 'keyword', ARRAY['movie', 'tv', 'show', 'music', 'game', 'gaming', 'funny', 'meme', 'entertainment', 'celebrity'], '{"platforms": ["reddit", "tiktok", "twitter"], "subreddits": ["r/movies", "r/gaming", "r/funny"]}', '[]', 5),
    
    ('Misc', 'keyword', ARRAY[], '{}', '[]', 1)
) AS category_data(category_name, rule_type, keywords, patterns, integration_actions, priority)
WHERE bp.name = 'Demo Profile'
ON CONFLICT DO NOTHING;

-- Insert sample bookmark items for demonstration
WITH demo_profile AS (
    SELECT id FROM bookmark_profiles WHERE name = 'Demo Profile' LIMIT 1
)
INSERT INTO bookmark_items (
    profile_id,
    platform,
    original_url,
    title,
    content_text,
    author_name,
    author_username,
    content_metadata,
    category_assigned,
    category_confidence,
    processing_status,
    hash_signature,
    processed_at
) 
SELECT 
    dp.id,
    sample_data.platform,
    sample_data.original_url,
    sample_data.title,
    sample_data.content_text,
    sample_data.author_name,
    sample_data.author_username,
    sample_data.content_metadata,
    sample_data.category_assigned,
    sample_data.category_confidence,
    'processed',
    md5(sample_data.original_url),
    NOW() - (sample_data.days_ago * interval '1 day')
FROM demo_profile dp,
(VALUES 
    ('reddit', 'https://reddit.com/r/programming/comments/example1', 'Best practices for API design', 'Great discussion about REST API design principles and common pitfalls to avoid.', 'CodeMaster', 'codemaster_dev', '{"subreddit": "r/programming", "score": 1247, "comments": 89}', 'Programming', 0.95, 1),
    
    ('twitter', 'https://x.com/user/status/example1', 'Tweet about JavaScript frameworks', 'Interesting comparison between React, Vue, and Angular performance in 2025', 'WebDev Pro', 'webdev_pro', '{"likes": 342, "retweets": 89, "replies": 23}', 'Programming', 0.88, 2),
    
    ('reddit', 'https://reddit.com/r/recipes/comments/example2', 'Amazing chocolate chip cookies', 'This recipe has been in my family for generations. The secret is browning the butter first!', 'GrandmasKitchen', 'grandmas_kitchen', '{"subreddit": "r/recipes", "score": 2341, "comments": 156}', 'Recipes', 0.92, 3),
    
    ('tiktok', 'https://tiktok.com/@fitness_guru/video/example1', 'TikTok by FitnessGuru', '10-minute morning workout that will change your life! No equipment needed.', 'Fitness Guru', 'fitness_guru', '{"views": 1200000, "likes": 89000, "shares": 12000, "duration": "00:00:45"}', 'Fitness', 0.87, 4),
    
    ('reddit', 'https://reddit.com/r/travel/comments/example3', 'Hidden gems in Japan', 'Just returned from a 3-week solo trip to Japan. Here are some incredible places most tourists never see.', 'SoloTraveler', 'solo_traveler_99', '{"subreddit": "r/travel", "score": 3456, "comments": 234}', 'Travel', 0.94, 5),
    
    ('twitter', 'https://x.com/tech_news/status/example2', 'Breaking: New AI breakthrough', 'Researchers at MIT have developed a new AI model that can understand context 40% better than previous models.', 'Tech News Daily', 'tech_news_daily', '{"likes": 1567, "retweets": 445, "replies": 123}', 'News', 0.91, 1)
) AS sample_data(platform, original_url, title, content_text, author_name, author_username, content_metadata, category_assigned, category_confidence, days_ago)
ON CONFLICT DO NOTHING;

-- Insert sample action items
WITH demo_bookmarks AS (
    SELECT bi.id, bi.category_assigned
    FROM bookmark_items bi
    JOIN bookmark_profiles bp ON bi.profile_id = bp.id
    WHERE bp.name = 'Demo Profile'
)
INSERT INTO action_items (
    bookmark_item_id,
    action_type,
    target_scenario,
    action_data,
    confidence_score,
    approval_status
)
SELECT 
    db.id,
    CASE 
        WHEN db.category_assigned = 'Recipes' THEN 'add_to_recipe_book'
        WHEN db.category_assigned = 'Fitness' THEN 'schedule_workout'
        WHEN db.category_assigned = 'Programming' THEN 'add_to_code_library'
        WHEN db.category_assigned = 'Travel' THEN 'add_to_travel_list'
        ELSE 'add_to_research'
    END,
    CASE 
        WHEN db.category_assigned = 'Recipes' THEN 'recipe-book'
        WHEN db.category_assigned = 'Fitness' THEN 'workout-plan-generator'
        WHEN db.category_assigned = 'Programming' THEN 'code-library'
        WHEN db.category_assigned = 'Travel' THEN 'travel-planner'
        ELSE 'research-assistant'
    END,
    CASE 
        WHEN db.category_assigned = 'Recipes' THEN '{"category": "desserts", "difficulty": "easy", "prep_time": "30min"}'
        WHEN db.category_assigned = 'Fitness' THEN '{"workout_type": "cardio", "duration": "10min", "equipment": "none"}'
        WHEN db.category_assigned = 'Programming' THEN '{"language": "javascript", "topic": "best_practices"}'
        WHEN db.category_assigned = 'Travel' THEN '{"destination": "Japan", "type": "hidden_gems"}'
        ELSE '{"topic": "general", "source": "social_media"}'
    END::jsonb,
    0.85,
    CASE WHEN random() > 0.7 THEN 'approved' ELSE 'pending' END
FROM demo_bookmarks db
WHERE db.category_assigned IN ('Recipes', 'Fitness', 'Programming', 'Travel', 'News')
ON CONFLICT DO NOTHING;

-- Insert platform integration records
WITH demo_profile AS (
    SELECT id FROM bookmark_profiles WHERE name = 'Demo Profile' LIMIT 1
)
INSERT INTO platform_integrations (
    profile_id,
    platform_name,
    integration_type,
    configuration,
    status,
    last_sync_at,
    sync_frequency
)
SELECT 
    dp.id,
    platform_data.platform_name,
    platform_data.integration_type,
    platform_data.configuration,
    platform_data.status,
    NOW() - (platform_data.hours_ago * interval '1 hour'),
    platform_data.sync_frequency::interval
FROM demo_profile dp,
(VALUES 
    ('reddit', 'huginn', '{"agent_name": "reddit-saved-posts-monitor", "subreddits": ["all"], "rate_limit": "1000/hour"}', 'active', 2, '30 minutes'),
    ('twitter', 'huginn', '{"agent_name": "twitter-bookmarks-monitor", "fallback": "browserless", "rate_limit": "300/15min"}', 'active', 1, '1 hour'),
    ('tiktok', 'browserless', '{"scraper_type": "favorites", "fallback_enabled": true, "rate_limit": "100/hour"}', 'inactive', 24, '2 hours')
) AS platform_data(platform_name, integration_type, configuration, status, hours_ago, sync_frequency)
ON CONFLICT DO NOTHING;

-- Insert processing statistics
WITH demo_profile AS (
    SELECT id FROM bookmark_profiles WHERE name = 'Demo Profile' LIMIT 1
)
INSERT INTO processing_stats (
    profile_id,
    date_period,
    platform,
    category,
    total_processed,
    correctly_categorized,
    user_corrections,
    actions_suggested,
    actions_approved,
    actions_rejected
)
SELECT 
    dp.id,
    CURRENT_DATE - (stats_data.days_ago * interval '1 day')::date,
    stats_data.platform,
    stats_data.category,
    stats_data.total_processed,
    stats_data.correctly_categorized,
    stats_data.user_corrections,
    stats_data.actions_suggested,
    stats_data.actions_approved,
    stats_data.actions_rejected
FROM demo_profile dp,
(VALUES 
    ('reddit', 'Programming', 25, 23, 2, 15, 12, 3, 0),
    ('reddit', 'Recipes', 18, 17, 1, 12, 10, 2, 1),
    ('twitter', 'News', 32, 29, 3, 8, 5, 3, 2),
    ('tiktok', 'Fitness', 15, 13, 2, 10, 7, 3, 3),
    ('reddit', 'Travel', 12, 11, 1, 8, 6, 2, 7)
) AS stats_data(platform, category, total_processed, correctly_categorized, user_corrections, actions_suggested, actions_approved, actions_rejected, days_ago)
ON CONFLICT DO NOTHING;

-- Create a function to generate realistic demo data over time
CREATE OR REPLACE FUNCTION generate_historical_stats(profile_uuid UUID, days_back INTEGER DEFAULT 30)
RETURNS VOID AS $$
DECLARE
    day_offset INTEGER;
    platforms TEXT[] := ARRAY['reddit', 'twitter', 'tiktok'];
    categories TEXT[] := ARRAY['Programming', 'Recipes', 'Fitness', 'Travel', 'News', 'Education'];
    platform TEXT;
    category TEXT;
BEGIN
    FOR day_offset IN 1..days_back LOOP
        FOREACH platform IN ARRAY platforms LOOP
            FOREACH category IN ARRAY categories LOOP
                -- Generate random but realistic stats
                INSERT INTO processing_stats (
                    profile_id,
                    date_period,
                    platform,
                    category,
                    total_processed,
                    correctly_categorized,
                    user_corrections,
                    actions_suggested,
                    actions_approved,
                    actions_rejected
                ) VALUES (
                    profile_uuid,
                    CURRENT_DATE - (day_offset * interval '1 day')::date,
                    platform,
                    category,
                    FLOOR(random() * 50 + 5)::INTEGER, -- 5-55 processed
                    FLOOR(random() * 45 + 5)::INTEGER, -- 5-50 correct
                    FLOOR(random() * 5)::INTEGER, -- 0-5 corrections
                    FLOOR(random() * 20 + 1)::INTEGER, -- 1-21 suggested
                    FLOOR(random() * 15 + 1)::INTEGER, -- 1-16 approved
                    FLOOR(random() * 8)::INTEGER -- 0-8 rejected
                );
            END LOOP;
        END LOOP;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Generate historical stats for demo profile (uncomment to enable)
-- SELECT generate_historical_stats(
--     (SELECT id FROM bookmark_profiles WHERE name = 'Demo Profile' LIMIT 1),
--     7 -- Last 7 days
-- );