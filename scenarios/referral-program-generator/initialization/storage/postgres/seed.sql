-- Referral Program Generator Seed Data
-- Version: 1.0.0

-- Insert sample referral programs for demonstration
INSERT INTO referral_programs (
    id,
    scenario_name,
    commission_rate,
    tracking_code,
    landing_page_url,
    branding_config,
    is_active
) VALUES 
(
    'a0000000-0000-0000-0000-000000000001',
    'sample-saas-app',
    0.25,
    'SAAS001',
    'http://localhost:3000/referral/SAAS001',
    '{"colors": {"primary": "#007bff", "secondary": "#6c757d", "accent": "#28a745"}, "brand_name": "Sample SaaS App", "logo_path": "", "fonts": ["Inter", "sans-serif"]}',
    true
),
(
    'a0000000-0000-0000-0000-000000000002',
    'e-commerce-platform',
    0.15,
    'ECOM002',
    'http://localhost:3000/referral/ECOM002',
    '{"colors": {"primary": "#ff6b6b", "secondary": "#4ecdc4", "accent": "#45b7d1"}, "brand_name": "E-Commerce Platform", "logo_path": "", "fonts": ["Roboto", "sans-serif"]}',
    true
),
(
    'a0000000-0000-0000-0000-000000000003',
    'productivity-tool',
    0.30,
    'PROD003',
    'http://localhost:3000/referral/PROD003',
    '{"colors": {"primary": "#9c88ff", "secondary": "#feca57", "accent": "#ff9ff3"}, "brand_name": "Productivity Tool", "logo_path": "", "fonts": ["Poppins", "sans-serif"]}',
    true
) ON CONFLICT (id) DO NOTHING;

-- Insert sample referral links for demonstration
INSERT INTO referral_links (
    id,
    program_id,
    referrer_id,
    tracking_code,
    clicks,
    conversions,
    total_commission,
    is_active,
    last_click_at,
    last_conversion_at
) VALUES 
(
    'b0000000-0000-0000-0000-000000000001',
    'a0000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000001',
    'REF123ABC',
    45,
    3,
    37.50,
    true,
    NOW() - INTERVAL '2 hours',
    NOW() - INTERVAL '1 day'
),
(
    'b0000000-0000-0000-0000-000000000002',
    'a0000000-0000-0000-0000-000000000001',
    '10000000-0000-0000-0000-000000000002',
    'REF456DEF',
    23,
    1,
    12.50,
    true,
    NOW() - INTERVAL '30 minutes',
    NOW() - INTERVAL '3 days'
),
(
    'b0000000-0000-0000-0000-000000000003',
    'a0000000-0000-0000-0000-000000000002',
    '10000000-0000-0000-0000-000000000001',
    'REF789GHI',
    67,
    8,
    96.00,
    true,
    NOW() - INTERVAL '1 hour',
    NOW() - INTERVAL '2 hours'
) ON CONFLICT (id) DO NOTHING;

-- Insert sample commissions
INSERT INTO commissions (
    id,
    link_id,
    customer_id,
    amount,
    status,
    transaction_date,
    approved_at,
    paid_date,
    payment_method,
    payment_reference,
    notes
) VALUES 
(
    'c0000000-0000-0000-0000-000000000001',
    'b0000000-0000-0000-0000-000000000001',
    '20000000-0000-0000-0000-000000000001',
    12.50,
    'paid',
    NOW() - INTERVAL '7 days',
    NOW() - INTERVAL '5 days',
    NOW() - INTERVAL '1 day',
    'stripe',
    'ch_3Abc123def456',
    'First commission payment'
),
(
    'c0000000-0000-0000-0000-000000000002',
    'b0000000-0000-0000-0000-000000000001',
    '20000000-0000-0000-0000-000000000002',
    25.00,
    'approved',
    NOW() - INTERVAL '3 days',
    NOW() - INTERVAL '1 day',
    NULL,
    NULL,
    NULL,
    'Pending payment processing'
),
(
    'c0000000-0000-0000-0000-000000000003',
    'b0000000-0000-0000-0000-000000000002',
    '20000000-0000-0000-0000-000000000003',
    12.50,
    'pending',
    NOW() - INTERVAL '1 day',
    NULL,
    NULL,
    NULL,
    NULL,
    'Awaiting approval'
),
(
    'c0000000-0000-0000-0000-000000000004',
    'b0000000-0000-0000-0000-000000000003',
    '20000000-0000-0000-0000-000000000004',
    48.00,
    'paid',
    NOW() - INTERVAL '10 days',
    NOW() - INTERVAL '8 days',
    NOW() - INTERVAL '3 days',
    'paypal',
    'PAYID-EXAMPLE123',
    'High-value conversion'
) ON CONFLICT (id) DO NOTHING;

-- Insert sample referral events for analytics
INSERT INTO referral_events (
    id,
    link_id,
    event_type,
    customer_id,
    ip_address,
    user_agent,
    referrer_url,
    landing_page,
    conversion_value,
    metadata
) VALUES 
(
    'd0000000-0000-0000-0000-000000000001',
    'b0000000-0000-0000-0000-000000000001',
    'click',
    NULL,
    '192.168.1.100',
    'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
    'https://example.com/blog/post',
    '/',
    NULL,
    '{"source": "blog", "campaign": "launch"}'
),
(
    'd0000000-0000-0000-0000-000000000002',
    'b0000000-0000-0000-0000-000000000001',
    'signup',
    '20000000-0000-0000-0000-000000000001',
    '192.168.1.100',
    'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
    'https://example.com/blog/post',
    '/signup',
    NULL,
    '{"signup_method": "email", "utm_source": "referral"}'
),
(
    'd0000000-0000-0000-0000-000000000003',
    'b0000000-0000-0000-0000-000000000001',
    'purchase',
    '20000000-0000-0000-0000-000000000001',
    '192.168.1.100',
    'Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36',
    'https://example.com/blog/post',
    '/checkout/success',
    50.00,
    '{"order_id": "ORD-12345", "items": [{"name": "Pro Plan", "price": 50.00}]}'
),
(
    'd0000000-0000-0000-0000-000000000004',
    'b0000000-0000-0000-0000-000000000003',
    'click',
    NULL,
    '203.0.113.45',
    'Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15',
    'https://twitter.com/user/status/123',
    '/',
    NULL,
    '{"source": "twitter", "campaign": "social"}'
) ON CONFLICT (id) DO NOTHING;

-- Insert sample campaign
INSERT INTO referral_campaigns (
    id,
    name,
    description,
    start_date,
    end_date,
    is_active
) VALUES 
(
    'e0000000-0000-0000-0000-000000000001',
    'Launch Campaign',
    'Initial launch campaign with increased commission rates',
    NOW() - INTERVAL '30 days',
    NOW() + INTERVAL '60 days',
    true
),
(
    'e0000000-0000-0000-0000-000000000002',
    'Holiday Special',
    'Special holiday promotion with bonus commissions',
    NOW() - INTERVAL '7 days',
    NOW() + INTERVAL '23 days',
    true
) ON CONFLICT (id) DO NOTHING;

-- Link programs to campaigns
INSERT INTO program_campaigns (program_id, campaign_id) VALUES 
('a0000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000001'),
('a0000000-0000-0000-0000-000000000002', 'e0000000-0000-0000-0000-000000000001'),
('a0000000-0000-0000-0000-000000000001', 'e0000000-0000-0000-0000-000000000002')
ON CONFLICT (program_id, campaign_id) DO NOTHING;

-- Insert configuration data
INSERT INTO referral_programs (
    scenario_name,
    commission_rate,
    tracking_code,
    landing_page_url,
    branding_config,
    is_active
) VALUES 
(
    'referral-program-generator',
    0.20,
    'REFGEN001',
    'http://localhost:3000/referral/REFGEN001',
    '{
        "colors": {
            "primary": "#6f42c1",
            "secondary": "#fd7e14",
            "accent": "#20c997"
        },
        "brand_name": "Referral Program Generator",
        "logo_path": "",
        "fonts": ["Inter", "system-ui", "sans-serif"]
    }',
    true
) ON CONFLICT (tracking_code) DO NOTHING;

-- Create some realistic performance data
DO $$
DECLARE
    program_rec RECORD;
    link_rec RECORD;
    i INTEGER;
    click_time TIMESTAMP;
    conversion_prob REAL;
BEGIN
    -- Add more realistic click and conversion data
    FOR program_rec IN SELECT id FROM referral_programs LOOP
        FOR link_rec IN SELECT id FROM referral_links WHERE program_id = program_rec.id LOOP
            -- Generate click events for the last 30 days
            FOR i IN 1..30 LOOP
                click_time := NOW() - (i || ' days')::INTERVAL + (RANDOM() * INTERVAL '23 hours');
                
                -- Generate 1-5 clicks per day with some randomness
                FOR j IN 1..(1 + FLOOR(RANDOM() * 4)) LOOP
                    INSERT INTO referral_events (
                        link_id,
                        event_type,
                        ip_address,
                        user_agent,
                        referrer_url,
                        landing_page,
                        created_at
                    ) VALUES (
                        link_rec.id,
                        'click',
                        ('192.168.1.' || FLOOR(RANDOM() * 255))::INET,
                        'Mozilla/5.0 (compatible; sample-user-agent)',
                        'https://example.com/source',
                        '/',
                        click_time + (j || ' minutes')::INTERVAL
                    );
                END LOOP;
                
                -- 15% chance of conversion each day
                conversion_prob := RANDOM();
                IF conversion_prob < 0.15 THEN
                    INSERT INTO referral_events (
                        link_id,
                        event_type,
                        customer_id,
                        ip_address,
                        user_agent,
                        referrer_url,
                        landing_page,
                        conversion_value,
                        created_at
                    ) VALUES (
                        link_rec.id,
                        'conversion',
                        gen_random_uuid(),
                        ('192.168.1.' || FLOOR(RANDOM() * 255))::INET,
                        'Mozilla/5.0 (compatible; sample-user-agent)',
                        'https://example.com/source',
                        '/checkout/success',
                        20 + (RANDOM() * 180), -- $20-$200 order value
                        click_time + INTERVAL '2 hours'
                    );
                END IF;
            END LOOP;
        END LOOP;
    END LOOP;
END $$;

-- Update referral link statistics based on generated events
UPDATE referral_links SET 
    clicks = (
        SELECT COUNT(*) 
        FROM referral_events 
        WHERE link_id = referral_links.id AND event_type = 'click'
    ),
    conversions = (
        SELECT COUNT(*) 
        FROM referral_events 
        WHERE link_id = referral_links.id AND event_type = 'conversion'
    ),
    total_commission = COALESCE((
        SELECT SUM(amount) 
        FROM commissions 
        WHERE link_id = referral_links.id AND status IN ('approved', 'paid')
    ), 0);

-- Add helpful comments about the seed data
INSERT INTO referral_programs (
    scenario_name,
    commission_rate,
    tracking_code,
    branding_config
) VALUES (
    'demo-scenario',
    0.25,
    'DEMO001',
    '{
        "colors": {"primary": "#007bff", "secondary": "#6c757d"},
        "brand_name": "Demo Scenario",
        "description": "This is sample data for testing the referral program generator"
    }'
) ON CONFLICT (tracking_code) DO NOTHING;

-- Log that seed data has been inserted
DO $$
BEGIN
    RAISE NOTICE 'Referral Program Generator seed data inserted successfully';
    RAISE NOTICE 'Sample programs: %', (SELECT COUNT(*) FROM referral_programs);
    RAISE NOTICE 'Sample links: %', (SELECT COUNT(*) FROM referral_links);
    RAISE NOTICE 'Sample commissions: %', (SELECT COUNT(*) FROM commissions);
    RAISE NOTICE 'Sample events: %', (SELECT COUNT(*) FROM referral_events);
END $$;