-- Recommendation Engine Seed Data
-- Sample data for testing and demonstrating recommendation capabilities

-- Insert sample scenarios configuration
INSERT INTO scenario_configs (scenario_id, config) VALUES 
('e-commerce-demo', '{
    "algorithms": {
        "collaborative_filtering": {"enabled": true, "weight": 0.5},
        "content_based": {"enabled": true, "weight": 0.3},
        "popularity_based": {"enabled": true, "weight": 0.2}
    },
    "recommendation_limits": {"default_count": 12, "max_count": 50}
}'),
('content-platform-demo', '{
    "algorithms": {
        "collaborative_filtering": {"enabled": true, "weight": 0.4},
        "content_based": {"enabled": true, "weight": 0.5},
        "popularity_based": {"enabled": true, "weight": 0.1}
    },
    "recommendation_limits": {"default_count": 8, "max_count": 30}
}')
ON CONFLICT (scenario_id) DO UPDATE SET 
    config = EXCLUDED.config,
    updated_at = CURRENT_TIMESTAMP;

-- Insert sample users
INSERT INTO users (scenario_id, external_id, preferences) VALUES 
('e-commerce-demo', 'user-001', '{"preferred_categories": ["electronics", "books"], "price_range": "mid", "brand_loyalty": "low"}'),
('e-commerce-demo', 'user-002', '{"preferred_categories": ["clothing", "accessories"], "price_range": "high", "brand_loyalty": "high"}'),
('e-commerce-demo', 'user-003', '{"preferred_categories": ["home", "garden"], "price_range": "low", "brand_loyalty": "medium"}'),
('content-platform-demo', 'user-101', '{"interests": ["technology", "science"], "content_type": "articles", "reading_time": "medium"}'),
('content-platform-demo', 'user-102', '{"interests": ["cooking", "lifestyle"], "content_type": "videos", "reading_time": "short"}'),
('content-platform-demo', 'user-103', '{"interests": ["business", "finance"], "content_type": "mixed", "reading_time": "long"}')
ON CONFLICT (scenario_id, external_id) DO NOTHING;

-- Insert sample items for e-commerce scenario
INSERT INTO items (scenario_id, external_id, title, description, category, metadata) VALUES 
-- Electronics
('e-commerce-demo', 'item-001', 'Wireless Bluetooth Headphones', 'High-quality wireless headphones with noise cancellation', 'electronics', '{"brand": "TechSound", "price": 89.99, "rating": 4.5, "features": ["noise_cancellation", "wireless", "long_battery"]}'),
('e-commerce-demo', 'item-002', 'Smartphone Android 12', 'Latest Android smartphone with 5G capability', 'electronics', '{"brand": "PhoneCorp", "price": 599.99, "rating": 4.3, "features": ["5g", "android12", "dual_camera"]}'),
('e-commerce-demo', 'item-003', 'Laptop Ultrabook 14inch', 'Lightweight ultrabook for productivity', 'electronics', '{"brand": "CompuTech", "price": 999.99, "rating": 4.7, "features": ["ultralight", "long_battery", "ssd"]}'),

-- Books
('e-commerce-demo', 'item-004', 'The Art of Programming', 'Comprehensive guide to software development', 'books', '{"author": "John Coder", "price": 34.99, "rating": 4.8, "genre": "technical", "pages": 650}'),
('e-commerce-demo', 'item-005', 'Mystery of the Lost Algorithm', 'Thrilling mystery novel for tech enthusiasts', 'books', '{"author": "Jane Writer", "price": 12.99, "rating": 4.2, "genre": "mystery", "pages": 320}'),

-- Clothing
('e-commerce-demo', 'item-006', 'Premium Cotton T-Shirt', 'Comfortable organic cotton t-shirt', 'clothing', '{"brand": "EcoWear", "price": 29.99, "rating": 4.4, "sizes": ["S", "M", "L", "XL"], "material": "organic_cotton"}'),
('e-commerce-demo', 'item-007', 'Designer Jeans Slim Fit', 'Premium denim jeans with modern fit', 'clothing', '{"brand": "DenimLux", "price": 89.99, "rating": 4.6, "sizes": ["28", "30", "32", "34"], "material": "premium_denim"}'),

-- Home & Garden
('e-commerce-demo', 'item-008', 'Smart Home Hub', 'Central control for all smart home devices', 'home', '{"brand": "SmartTech", "price": 149.99, "rating": 4.5, "features": ["voice_control", "app_control", "device_integration"]}'),
('e-commerce-demo', 'item-009', 'Indoor Plant Care Kit', 'Complete kit for maintaining indoor plants', 'garden', '{"brand": "GreenThumb", "price": 39.99, "rating": 4.3, "features": ["plant_food", "watering_tools", "care_guide"]}}')

ON CONFLICT (scenario_id, external_id) DO NOTHING;

-- Insert sample items for content platform scenario
INSERT INTO items (scenario_id, external_id, title, description, category, metadata) VALUES 
-- Technology content
('content-platform-demo', 'article-001', 'The Future of Artificial Intelligence', 'Deep dive into AI trends and predictions for 2025', 'technology', '{"author": "Dr. AI Expert", "read_time": 8, "difficulty": "intermediate", "topics": ["artificial_intelligence", "machine_learning", "future_tech"]}'),
('content-platform-demo', 'article-002', 'Quantum Computing Explained', 'Beginner-friendly introduction to quantum computing', 'technology', '{"author": "Prof. Quantum", "read_time": 12, "difficulty": "beginner", "topics": ["quantum_computing", "physics", "computing"]}'),

-- Science content
('content-platform-demo', 'article-003', 'Climate Change and Technology Solutions', 'How technology is helping combat climate change', 'science', '{"author": "Dr. Green Tech", "read_time": 10, "difficulty": "intermediate", "topics": ["climate_change", "green_technology", "sustainability"]}'),

-- Cooking content
('content-platform-demo', 'video-001', 'Quick 15-Minute Pasta Recipes', 'Fast and delicious pasta recipes for busy weekdays', 'cooking', '{"author": "Chef Mario", "duration": 15, "difficulty": "easy", "topics": ["pasta", "quick_meals", "italian"]}'),
('content-platform-demo', 'video-002', 'Mastering French Pastry Basics', 'Learn fundamental French pastry techniques', 'cooking', '{"author": "Chef Française", "duration": 45, "difficulty": "advanced", "topics": ["pastry", "french_cuisine", "baking"]}'),

-- Business content
('content-platform-demo', 'article-004', 'Startup Fundraising in 2025', 'Complete guide to raising capital for your startup', 'business', '{"author": "Sarah Entrepreneur", "read_time": 15, "difficulty": "intermediate", "topics": ["fundraising", "startups", "venture_capital"]}'),
('content-platform-demo', 'article-005', 'Remote Work Management Best Practices', 'How to effectively manage remote teams', 'business', '{"author": "Mike Manager", "read_time": 6, "difficulty": "beginner", "topics": ["remote_work", "management", "productivity"]}}')

ON CONFLICT (scenario_id, external_id) DO NOTHING;

-- Insert sample user interactions for e-commerce
INSERT INTO user_interactions (user_id, item_id, interaction_type, interaction_value, context) 
SELECT 
    u.id,
    i.id,
    interactions.interaction_type,
    interactions.value,
    interactions.context::jsonb
FROM users u
CROSS JOIN items i
JOIN (
    VALUES 
    -- User 001 (electronics & books enthusiast)
    ('user-001', 'item-001', 'view', 1.0, '{"session_id": "sess-001", "device": "desktop", "time_of_day": "evening"}'),
    ('user-001', 'item-001', 'purchase', 5.0, '{"session_id": "sess-001", "device": "desktop", "payment_method": "credit_card"}'),
    ('user-001', 'item-002', 'view', 1.0, '{"session_id": "sess-002", "device": "mobile", "time_of_day": "morning"}'),
    ('user-001', 'item-003', 'view', 1.0, '{"session_id": "sess-002", "device": "mobile", "time_of_day": "morning"}'),
    ('user-001', 'item-003', 'like', 2.0, '{"session_id": "sess-002", "device": "mobile"}'),
    ('user-001', 'item-004', 'view', 1.0, '{"session_id": "sess-003", "device": "desktop", "time_of_day": "afternoon"}'),
    ('user-001', 'item-004', 'purchase', 5.0, '{"session_id": "sess-003", "device": "desktop", "payment_method": "paypal"}'),
    
    -- User 002 (fashion enthusiast)
    ('user-002', 'item-006', 'view', 1.0, '{"session_id": "sess-004", "device": "mobile", "time_of_day": "lunch"}'),
    ('user-002', 'item-006', 'like', 2.0, '{"session_id": "sess-004", "device": "mobile"}'),
    ('user-002', 'item-007', 'view', 1.0, '{"session_id": "sess-004", "device": "mobile", "time_of_day": "lunch"}'),
    ('user-002', 'item-007', 'purchase', 5.0, '{"session_id": "sess-004", "device": "mobile", "payment_method": "apple_pay"}'),
    ('user-002', 'item-001', 'view', 1.0, '{"session_id": "sess-005", "device": "desktop", "time_of_day": "evening"}'),
    
    -- User 003 (home & garden enthusiast)
    ('user-003', 'item-008', 'view', 1.0, '{"session_id": "sess-006", "device": "tablet", "time_of_day": "weekend_morning"}'),
    ('user-003', 'item-009', 'view', 1.0, '{"session_id": "sess-006", "device": "tablet", "time_of_day": "weekend_morning"}'),
    ('user-003', 'item-009', 'purchase', 5.0, '{"session_id": "sess-006", "device": "tablet", "payment_method": "debit_card"}'),
    ('user-003', 'item-004', 'view', 1.0, '{"session_id": "sess-007", "device": "desktop", "time_of_day": "evening"}')
) interactions(user_ext_id, item_ext_id, interaction_type, value, context)
WHERE u.external_id = interactions.user_ext_id 
  AND i.external_id = interactions.item_ext_id
  AND u.scenario_id = 'e-commerce-demo'
  AND i.scenario_id = 'e-commerce-demo';

-- Insert sample user interactions for content platform
INSERT INTO user_interactions (user_id, item_id, interaction_type, interaction_value, context)
SELECT 
    u.id,
    i.id,
    interactions.interaction_type,
    interactions.value,
    interactions.context::jsonb
FROM users u
CROSS JOIN items i
JOIN (
    VALUES 
    -- User 101 (tech & science reader)
    ('user-101', 'article-001', 'view', 1.0, '{"session_id": "sess-101", "device": "desktop", "referrer": "google", "read_percentage": 85}'),
    ('user-101', 'article-001', 'share', 3.0, '{"session_id": "sess-101", "device": "desktop", "platform": "twitter"}'),
    ('user-101', 'article-002', 'view', 1.0, '{"session_id": "sess-102", "device": "mobile", "read_percentage": 95}'),
    ('user-101', 'article-002', 'like', 2.0, '{"session_id": "sess-102", "device": "mobile"}'),
    ('user-101', 'article-003', 'view', 1.0, '{"session_id": "sess-103", "device": "desktop", "read_percentage": 70}'),
    
    -- User 102 (cooking enthusiast, prefers videos)
    ('user-102', 'video-001', 'view', 1.0, '{"session_id": "sess-104", "device": "tablet", "watch_percentage": 100}'),
    ('user-102', 'video-001', 'like', 2.0, '{"session_id": "sess-104", "device": "tablet"}'),
    ('user-102', 'video-002', 'view', 1.0, '{"session_id": "sess-105", "device": "tablet", "watch_percentage": 60}'),
    ('user-102', 'article-001', 'view', 1.0, '{"session_id": "sess-106", "device": "mobile", "read_percentage": 30}'),
    
    -- User 103 (business content consumer)
    ('user-103', 'article-004', 'view', 1.0, '{"session_id": "sess-107", "device": "desktop", "read_percentage": 100}'),
    ('user-103', 'article-004', 'share', 3.0, '{"session_id": "sess-107", "device": "desktop", "platform": "linkedin"}'),
    ('user-103', 'article-005', 'view', 1.0, '{"session_id": "sess-108", "device": "mobile", "read_percentage": 90}'),
    ('user-103', 'article-005', 'like', 2.0, '{"session_id": "sess-108", "device": "mobile"}'),
    ('user-103', 'article-001', 'view', 1.0, '{"session_id": "sess-109", "device": "desktop", "read_percentage": 50}')
) interactions(user_ext_id, item_ext_id, interaction_type, value, context)
WHERE u.external_id = interactions.user_ext_id 
  AND i.external_id = interactions.item_ext_id
  AND u.scenario_id = 'content-platform-demo'
  AND i.scenario_id = 'content-platform-demo';

-- Insert some sample recommendation events for testing
INSERT INTO recommendation_events (user_id, scenario_id, algorithm_used, recommended_items, context, clicked_items) 
SELECT 
    u.id,
    'e-commerce-demo',
    'collaborative_filtering',
    '[{"item_id": "item-002", "score": 0.85}, {"item_id": "item-003", "score": 0.78}]'::jsonb,
    '{"request_context": "homepage", "device": "desktop"}'::jsonb,
    '["item-003"]'::jsonb
FROM users u 
WHERE u.external_id = 'user-001' AND u.scenario_id = 'e-commerce-demo';

INSERT INTO recommendation_events (user_id, scenario_id, algorithm_used, recommended_items, context, clicked_items) 
SELECT 
    u.id,
    'content-platform-demo',
    'content_based',
    '[{"item_id": "article-003", "score": 0.92}, {"item_id": "article-002", "score": 0.71}]'::jsonb,
    '{"request_context": "article_page", "device": "mobile"}'::jsonb,
    '["article-003"]'::jsonb
FROM users u 
WHERE u.external_id = 'user-101' AND u.scenario_id = 'content-platform-demo';

-- Insert sample algorithm performance data
INSERT INTO algorithm_performance (algorithm_name, scenario_id, metric_name, metric_value, sample_size, test_period_start, test_period_end) VALUES 
('collaborative_filtering', 'e-commerce-demo', 'click_through_rate', 0.12, 1000, CURRENT_TIMESTAMP - INTERVAL '7 days', CURRENT_TIMESTAMP - INTERVAL '1 day'),
('content_based', 'e-commerce-demo', 'click_through_rate', 0.08, 1000, CURRENT_TIMESTAMP - INTERVAL '7 days', CURRENT_TIMESTAMP - INTERVAL '1 day'),
('popularity_based', 'e-commerce-demo', 'click_through_rate', 0.05, 1000, CURRENT_TIMESTAMP - INTERVAL '7 days', CURRENT_TIMESTAMP - INTERVAL '1 day'),
('collaborative_filtering', 'content-platform-demo', 'click_through_rate', 0.18, 500, CURRENT_TIMESTAMP - INTERVAL '7 days', CURRENT_TIMESTAMP - INTERVAL '1 day'),
('content_based', 'content-platform-demo', 'click_through_rate', 0.22, 500, CURRENT_TIMESTAMP - INTERVAL '7 days', CURRENT_TIMESTAMP - INTERVAL '1 day'),
('popularity_based', 'content-platform-demo', 'click_through_rate', 0.09, 500, CURRENT_TIMESTAMP - INTERVAL '7 days', CURRENT_TIMESTAMP - INTERVAL '1 day');

-- Refresh materialized view with new data
REFRESH MATERIALIZED VIEW popular_items_by_category;

-- Show summary of seed data
SELECT 'Seed data inserted successfully!' as status;
SELECT scenario_id, COUNT(*) as user_count FROM users GROUP BY scenario_id;
SELECT scenario_id, COUNT(*) as item_count FROM items GROUP BY scenario_id;
SELECT COUNT(*) as interaction_count FROM user_interactions;
SELECT COUNT(*) as recommendation_event_count FROM recommendation_events;