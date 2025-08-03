-- E-commerce Platform Sample Data
-- Initial seed data for testing and development

-- Insert Categories
INSERT INTO categories (id, name, slug, description, sort_order, is_active) VALUES
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Electronics', 'electronics', 'Electronic devices and accessories', 1, true),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'Clothing', 'clothing', 'Fashion and apparel', 2, true),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a13', 'Home & Garden', 'home-garden', 'Home improvement and garden supplies', 3, true),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14', 'Books', 'books', 'Books and publications', 4, true),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a15', 'Sports & Outdoors', 'sports-outdoors', 'Sporting goods and outdoor equipment', 5, true),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a16', 'Toys & Games', 'toys-games', 'Toys, games, and hobbies', 6, true),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a17', 'Food & Beverages', 'food-beverages', 'Groceries and drinks', 7, true),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a18', 'Health & Beauty', 'health-beauty', 'Personal care and wellness', 8, true),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a19', 'Automotive', 'automotive', 'Auto parts and accessories', 9, true),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a20', 'Office Supplies', 'office-supplies', 'Business and office products', 10, true);

-- Insert Subcategories
INSERT INTO categories (parent_id, name, slug, description, sort_order, is_active) VALUES
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Smartphones', 'smartphones', 'Mobile phones and accessories', 1, true),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Laptops', 'laptops', 'Notebook computers', 2, true),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'Tablets', 'tablets', 'Tablet computers', 3, true),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'Men''s Clothing', 'mens-clothing', 'Clothing for men', 1, true),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'Women''s Clothing', 'womens-clothing', 'Clothing for women', 2, true),
    ('a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'Shoes', 'shoes', 'Footwear', 3, true);

-- Insert Test Users
INSERT INTO users (email, username, password_hash, first_name, last_name, role, email_verified) VALUES
    ('admin@example.com', 'admin', '$2a$10$K.0HwpsoPDGaB/atFBmmXOGTw4ceeg33.WrxJgccpkRJLYybA8Jl2', 'Admin', 'User', 'admin', true),
    ('john.doe@example.com', 'johndoe', '$2a$10$K.0HwpsoPDGaB/atFBmmXOGTw4ceeg33.WrxJgccpkRJLYybA8Jl2', 'John', 'Doe', 'customer', true),
    ('jane.smith@example.com', 'janesmith', '$2a$10$K.0HwpsoPDGaB/atFBmmXOGTw4ceeg33.WrxJgccpkRJLYybA8Jl2', 'Jane', 'Smith', 'customer', true),
    ('vendor@example.com', 'vendor1', '$2a$10$K.0HwpsoPDGaB/atFBmmXOGTw4ceeg33.WrxJgccpkRJLYybA8Jl2', 'Vendor', 'One', 'vendor', true);

-- Insert Sample Products
INSERT INTO products (sku, name, slug, description, short_description, category_id, brand, status, featured, metadata) VALUES
    ('PHONE-001', 'Premium Smartphone Pro Max', 'premium-smartphone-pro-max', 
     'Experience the ultimate in mobile technology with our flagship smartphone featuring cutting-edge performance, stunning display, and professional-grade cameras.',
     'Flagship smartphone with advanced features', 
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'TechBrand', 'active', true, 
     '{"specifications": {"screen": "6.7 inch OLED", "camera": "108MP", "battery": "5000mAh"}}'::jsonb),
    
    ('LAPTOP-001', 'UltraBook Pro 15', 'ultrabook-pro-15',
     'Powerful and portable laptop designed for professionals. Features latest generation processor, dedicated graphics, and all-day battery life.',
     'High-performance laptop for professionals',
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'CompuTech', 'active', true,
     '{"specifications": {"processor": "Intel i7", "ram": "16GB", "storage": "512GB SSD"}}'::jsonb),
    
    ('SHIRT-001', 'Classic Cotton T-Shirt', 'classic-cotton-tshirt',
     '100% organic cotton t-shirt with a comfortable fit. Perfect for everyday wear.',
     'Comfortable cotton t-shirt',
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a12', 'FashionCo', 'active', false,
     '{"material": "100% Organic Cotton", "care": "Machine washable"}'::jsonb),
    
    ('BOOK-001', 'The Art of Programming', 'art-of-programming',
     'Comprehensive guide to modern programming practices and principles. Essential reading for developers.',
     'Programming guide for developers',
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a14', 'TechPublish', 'active', false,
     '{"pages": 450, "isbn": "978-1234567890", "language": "English"}'::jsonb),
    
    ('WATCH-001', 'Smart Fitness Watch', 'smart-fitness-watch',
     'Track your fitness goals with this advanced smartwatch featuring heart rate monitoring, GPS, and workout tracking.',
     'Fitness tracking smartwatch',
     'a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11', 'FitTech', 'active', true,
     '{"features": ["Heart Rate Monitor", "GPS", "Water Resistant", "7-day Battery"]}'::jsonb);

-- Insert Product Variants
INSERT INTO product_variants (product_id, sku, name, price, compare_price, cost, quantity, attributes) 
SELECT 
    p.id,
    p.sku || '-BLK-128',
    '128GB - Black',
    999.99,
    1199.99,
    650.00,
    50,
    '{"color": "Black", "storage": "128GB"}'::jsonb
FROM products p WHERE p.sku = 'PHONE-001';

INSERT INTO product_variants (product_id, sku, name, price, compare_price, cost, quantity, attributes)
SELECT 
    p.id,
    p.sku || '-SLV-256',
    '256GB - Silver',
    1199.99,
    1399.99,
    750.00,
    30,
    '{"color": "Silver", "storage": "256GB"}'::jsonb
FROM products p WHERE p.sku = 'PHONE-001';

INSERT INTO product_variants (product_id, sku, name, price, compare_price, cost, quantity, attributes)
SELECT 
    p.id,
    p.sku || '-15-I7',
    '15" Intel i7 16GB/512GB',
    1499.99,
    1799.99,
    1100.00,
    25,
    '{"screen_size": "15 inch", "processor": "Intel i7", "ram": "16GB", "storage": "512GB"}'::jsonb
FROM products p WHERE p.sku = 'LAPTOP-001';

INSERT INTO product_variants (product_id, sku, name, price, cost, quantity, attributes)
SELECT 
    p.id,
    p.sku || '-' || size || '-' || color,
    size || ' - ' || color,
    24.99,
    12.00,
    100,
    json_build_object('size', size, 'color', color)::jsonb
FROM products p
CROSS JOIN (VALUES ('S'), ('M'), ('L'), ('XL')) AS sizes(size)
CROSS JOIN (VALUES ('White'), ('Black'), ('Navy'), ('Gray')) AS colors(color)
WHERE p.sku = 'SHIRT-001';

-- Insert Sample Coupons
INSERT INTO coupons (code, description, discount_type, discount_value, minimum_amount, usage_limit, valid_until) VALUES
    ('WELCOME10', 'Welcome discount for new customers', 'percentage', 10, 50.00, 1000, CURRENT_TIMESTAMP + INTERVAL '30 days'),
    ('SAVE20', 'Save $20 on orders over $100', 'fixed', 20, 100.00, 500, CURRENT_TIMESTAMP + INTERVAL '15 days'),
    ('FREESHIP', 'Free shipping on all orders', 'free_shipping', 0, 0, NULL, CURRENT_TIMESTAMP + INTERVAL '7 days'),
    ('FLASH50', 'Flash sale - 50% off', 'percentage', 50, 200.00, 100, CURRENT_TIMESTAMP + INTERVAL '1 day');

-- Insert Sample Reviews
INSERT INTO reviews (product_id, user_id, rating, title, comment, verified_purchase, status)
SELECT 
    p.id,
    u.id,
    5,
    'Excellent product!',
    'This product exceeded my expectations. Great quality and fast shipping. Highly recommended!',
    true,
    'approved'
FROM products p, users u
WHERE p.sku = 'PHONE-001' AND u.email = 'john.doe@example.com';

INSERT INTO reviews (product_id, user_id, rating, title, comment, verified_purchase, status)
SELECT 
    p.id,
    u.id,
    4,
    'Good value for money',
    'Solid product with good features. Minor issues with battery life but overall satisfied.',
    true,
    'approved'
FROM products p, users u
WHERE p.sku = 'PHONE-001' AND u.email = 'jane.smith@example.com';

-- Insert Sample Orders
WITH sample_order AS (
    INSERT INTO orders (
        order_number, 
        user_id, 
        email, 
        status, 
        payment_status, 
        fulfillment_status,
        currency, 
        subtotal, 
        tax_amount, 
        shipping_amount, 
        total,
        created_at
    )
    SELECT 
        'ORD-2024-001001',
        u.id,
        u.email,
        'delivered',
        'paid',
        'fulfilled',
        'USD',
        999.99,
        80.00,
        10.00,
        1089.99,
        CURRENT_TIMESTAMP - INTERVAL '5 days'
    FROM users u WHERE u.email = 'john.doe@example.com'
    RETURNING id
)
INSERT INTO order_items (order_id, product_id, variant_id, product_name, product_sku, quantity, price, total)
SELECT 
    so.id,
    pv.product_id,
    pv.id,
    p.name,
    pv.sku,
    1,
    999.99,
    999.99
FROM sample_order so, product_variants pv
JOIN products p ON p.id = pv.product_id
WHERE pv.sku = 'PHONE-001-BLK-128';

-- Insert more sample orders with different statuses
WITH sample_order AS (
    INSERT INTO orders (
        order_number, 
        user_id, 
        email, 
        status, 
        payment_status, 
        fulfillment_status,
        currency, 
        subtotal, 
        tax_amount, 
        shipping_amount, 
        total,
        created_at
    )
    SELECT 
        'ORD-2024-001002',
        u.id,
        u.email,
        'processing',
        'paid',
        'unfulfilled',
        'USD',
        1524.98,
        122.00,
        0.00,
        1646.98,
        CURRENT_TIMESTAMP - INTERVAL '1 day'
    FROM users u WHERE u.email = 'jane.smith@example.com'
    RETURNING id
)
INSERT INTO order_items (order_id, product_id, variant_id, product_name, product_sku, quantity, price, total)
SELECT 
    so.id,
    pv.product_id,
    pv.id,
    p.name,
    pv.sku,
    1,
    1499.99,
    1499.99
FROM sample_order so, product_variants pv
JOIN products p ON p.id = pv.product_id
WHERE pv.sku = 'LAPTOP-001-15-I7'
UNION ALL
SELECT 
    so.id,
    pv.product_id,
    pv.id,
    p.name,
    pv.sku,
    1,
    24.99,
    24.99
FROM sample_order so, product_variants pv
JOIN products p ON p.id = pv.product_id
WHERE pv.sku = 'SHIRT-001-M-Black';

-- Insert Sample Cart (Abandoned Cart)
WITH sample_cart AS (
    INSERT INTO carts (user_id, status, currency, created_at, updated_at)
    SELECT 
        u.id,
        'abandoned',
        'USD',
        CURRENT_TIMESTAMP - INTERVAL '2 days',
        CURRENT_TIMESTAMP - INTERVAL '2 days'
    FROM users u WHERE u.email = 'john.doe@example.com'
    RETURNING id
)
INSERT INTO cart_items (cart_id, product_id, variant_id, quantity, price)
SELECT 
    sc.id,
    pv.product_id,
    pv.id,
    2,
    24.99
FROM sample_cart sc, product_variants pv
WHERE pv.sku = 'SHIRT-001-L-White';

-- Insert Sample Wishlists
INSERT INTO wishlists (user_id, product_id)
SELECT u.id, p.id
FROM users u, products p
WHERE u.email = 'jane.smith@example.com' 
AND p.sku IN ('PHONE-001', 'WATCH-001');

-- Update product search vectors (already handled by trigger, but let's ensure)
UPDATE products SET updated_at = CURRENT_TIMESTAMP;

-- Add some analytics events
INSERT INTO analytics_events (user_id, session_id, event_type, event_data, created_at)
SELECT 
    u.id,
    'session_' || generate_series,
    CASE 
        WHEN generate_series % 3 = 0 THEN 'page_view'
        WHEN generate_series % 3 = 1 THEN 'product_view'
        ELSE 'add_to_cart'
    END,
    json_build_object(
        'page', CASE 
            WHEN generate_series % 3 = 0 THEN '/home'
            WHEN generate_series % 3 = 1 THEN '/product/phone'
            ELSE '/cart'
        END,
        'duration', (random() * 300)::int
    )::jsonb,
    CURRENT_TIMESTAMP - (generate_series || ' hours')::interval
FROM users u, generate_series(1, 20)
WHERE u.email = 'john.doe@example.com';

-- Display summary
SELECT 'Data seeding completed!' AS status;
SELECT 'Categories' AS table_name, COUNT(*) AS count FROM categories
UNION ALL
SELECT 'Products', COUNT(*) FROM products
UNION ALL
SELECT 'Product Variants', COUNT(*) FROM product_variants
UNION ALL
SELECT 'Users', COUNT(*) FROM users
UNION ALL
SELECT 'Orders', COUNT(*) FROM orders
UNION ALL
SELECT 'Reviews', COUNT(*) FROM reviews
UNION ALL
SELECT 'Coupons', COUNT(*) FROM coupons;