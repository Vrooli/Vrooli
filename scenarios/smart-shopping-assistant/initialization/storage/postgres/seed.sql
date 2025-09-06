-- Smart Shopping Assistant Seed Data
-- Version: 1.0.0

-- Insert demo shopping profile
INSERT INTO shopping_profiles (id, name, preferences, budget_limits, settings)
VALUES 
    ('11111111-1111-1111-1111-111111111111', 
     'Demo Profile', 
     '{"categories": ["electronics", "home", "books"], "brands": ["Apple", "Sony", "Samsung"], "avoid": ["leather"]}',
     '{"monthly": 1000, "per_purchase": 500, "categories": {"electronics": 2000}}',
     '{"notifications": true, "price_drop_threshold": 10, "auto_track": true}')
ON CONFLICT (id) DO NOTHING;

-- Insert sample products
INSERT INTO products (id, external_id, name, category, subcategory, description, brand, model)
VALUES 
    ('22222222-2222-2222-2222-222222222222',
     'B08N5WRWNW',
     'Echo Dot (4th Gen)',
     'Electronics',
     'Smart Home',
     'Smart speaker with Alexa',
     'Amazon',
     'Echo Dot 4'),
    ('33333333-3333-3333-3333-333333333333',
     'B09B8XNZM8',
     'AirPods Pro (2nd generation)',
     'Electronics',
     'Audio',
     'Active Noise Cancelling earbuds',
     'Apple',
     'MQD83AM/A'),
    ('44444444-4444-4444-4444-444444444444',
     'B07FZ8S74R',
     'Kindle Paperwhite',
     'Electronics',
     'E-Readers',
     'Waterproof e-reader with 8GB storage',
     'Amazon',
     'Paperwhite 10th Gen')
ON CONFLICT (external_id) DO NOTHING;

-- Insert sample price history
INSERT INTO price_history (product_id, retailer, price, original_price, availability, url)
VALUES 
    ('22222222-2222-2222-2222-222222222222', 'Amazon', 39.99, 49.99, 'In Stock', 'https://amazon.com/echo-dot'),
    ('22222222-2222-2222-2222-222222222222', 'BestBuy', 44.99, 49.99, 'In Stock', 'https://bestbuy.com/echo-dot'),
    ('33333333-3333-3333-3333-333333333333', 'Apple', 249.00, 249.00, 'In Stock', 'https://apple.com/airpods-pro'),
    ('33333333-3333-3333-3333-333333333333', 'Amazon', 234.99, 249.00, 'In Stock', 'https://amazon.com/airpods-pro'),
    ('44444444-4444-4444-4444-444444444444', 'Amazon', 139.99, 139.99, 'In Stock', 'https://amazon.com/kindle-paperwhite');

-- Insert sample price alerts
INSERT INTO price_alerts (profile_id, product_id, target_price, alert_type, is_active)
VALUES 
    ('11111111-1111-1111-1111-111111111111', '22222222-2222-2222-2222-222222222222', 35.00, 'below_target', true),
    ('11111111-1111-1111-1111-111111111111', '33333333-3333-3333-3333-333333333333', 200.00, 'below_target', true);

-- Insert sample affiliate links
INSERT INTO affiliate_links (product_id, retailer, affiliate_network, original_url, affiliate_url, commission_rate, is_active)
VALUES 
    ('22222222-2222-2222-2222-222222222222', 
     'Amazon', 
     'Amazon Associates',
     'https://amazon.com/echo-dot',
     'https://amazon.com/echo-dot?tag=smartshop-20',
     4.0,
     true),
    ('33333333-3333-3333-3333-333333333333',
     'Apple',
     'Apple Affiliate',
     'https://apple.com/airpods-pro',
     'https://apple.com/airpods-pro?at=smartshop',
     2.5,
     true);

-- Insert sample purchase pattern
INSERT INTO purchase_patterns (profile_id, product_category, pattern_type, frequency_days, average_quantity, average_spend, confidence_score)
VALUES 
    ('11111111-1111-1111-1111-111111111111', 'Electronics', 'seasonal', 90, 1.5, 250.00, 0.75);

-- Insert sample reviews summary
INSERT INTO reviews_summary (product_id, source, average_rating, total_reviews, pros, cons, sentiment_score)
VALUES 
    ('22222222-2222-2222-2222-222222222222',
     'Amazon',
     4.5,
     50000,
     '["Great sound quality", "Easy setup", "Compact design"]',
     '["Limited bass", "Requires power outlet"]',
     0.85),
    ('33333333-3333-3333-3333-333333333333',
     'Apple',
     4.7,
     25000,
     '["Excellent noise cancellation", "Great battery life", "Comfortable fit"]',
     '["Expensive", "Can fall out during exercise"]',
     0.90);