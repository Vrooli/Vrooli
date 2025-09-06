-- Social Media Scheduler Seed Data
-- This file provides initial data for development and testing

-- Insert sample user accounts for testing
INSERT INTO users (id, email, password_hash, first_name, last_name, subscription_tier, timezone, preferences, usage_limits) VALUES
(
    '550e8400-e29b-41d4-a716-446655440001',
    'demo@vrooli.com',
    '$2a$10$N9qo8uLOickgx2ZMRZoMye1VDvyUqZHVPPp/3wvS2nOWE.tFpJ/f2', -- password: demo123
    'Demo',
    'User',
    'professional',
    'America/New_York',
    '{"theme": "light", "notifications": true, "auto_optimize": true}',
    '{"posts_per_month": 1000, "platforms": 5, "team_members": 3}'
),
(
    '550e8400-e29b-41d4-a716-446655440002',
    'agency@vrooli.com',
    '$2a$10$N9qo8uLOickgx2ZMRZoMye1VDvyUqZHVPPp/3wvS2nOWE.tFpJ/f2', -- password: demo123
    'Agency',
    'Manager',
    'agency',
    'UTC',
    '{"theme": "dark", "notifications": true, "bulk_operations": true}',
    '{"posts_per_month": 5000, "platforms": 10, "team_members": 15}'
);

-- Insert sample social media accounts (using placeholder tokens for development)
INSERT INTO social_accounts (id, user_id, platform, platform_user_id, username, display_name, access_token, refresh_token, token_expires_at, account_data, is_active) VALUES
(
    '660e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440001',
    'twitter',
    '123456789',
    'vrooli_demo',
    'Vrooli Demo Account',
    'demo_twitter_access_token_encrypted',
    'demo_twitter_refresh_token_encrypted',
    NOW() + INTERVAL '60 days',
    '{"followers_count": 1250, "verified": false, "profile_image_url": "https://example.com/avatar.jpg"}',
    true
),
(
    '660e8400-e29b-41d4-a716-446655440002',
    '550e8400-e29b-41d4-a716-446655440001',
    'linkedin',
    'linkedin_user_123',
    'vrooli-demo',
    'Vrooli Demo Company',
    'demo_linkedin_access_token_encrypted',
    NULL,
    NOW() + INTERVAL '30 days',
    '{"connections_count": 500, "company_page": true, "industry": "Technology"}',
    true
),
(
    '660e8400-e29b-41d4-a716-446655440003',
    '550e8400-e29b-41d4-a716-446655440001',
    'facebook',
    'facebook_page_456',
    'vrooli-page',
    'Vrooli Demo Page',
    'demo_facebook_access_token_encrypted',
    NULL,
    NOW() + INTERVAL '45 days',
    '{"page_likes": 2500, "page_type": "Business", "category": "Software Company"}',
    true
),
(
    '660e8400-e29b-41d4-a716-446655440004',
    '550e8400-e29b-41d4-a716-446655440002',
    'instagram',
    'instagram_business_789',
    'vrooli_agency',
    'Vrooli Marketing Agency',
    'demo_instagram_access_token_encrypted',
    NULL,
    NOW() + INTERVAL '30 days',
    '{"followers_count": 5000, "business_account": true, "profile_picture_url": "https://example.com/insta_avatar.jpg"}',
    true
);

-- Insert sample campaigns
INSERT INTO campaigns (id, user_id, name, description, status, start_date, end_date, brand_guidelines) VALUES
(
    '770e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440001',
    'Summer Product Launch',
    'Launch campaign for our new AI-powered social media scheduler',
    'active',
    '2024-06-01',
    '2024-08-31',
    '{"primary_color": "#4285f4", "secondary_color": "#34a853", "tone": "professional", "hashtags": ["#VrooliLaunch", "#AISocialMedia", "#ProductLaunch"]}'
),
(
    '770e8400-e29b-41d4-a716-446655440002',
    '550e8400-e29b-41d4-a716-446655440001',
    'Holiday Season 2024',
    'Holiday promotional campaign with special offers',
    'active',
    '2024-11-01',
    '2024-12-31',
    '{"primary_color": "#d32f2f", "secondary_color": "#388e3c", "tone": "festive", "hashtags": ["#HolidayOffers", "#VrooliSpecial"]}'
);

-- Insert sample scheduled posts
INSERT INTO scheduled_posts (id, user_id, campaign_id, title, content, platform_variants, platforms, scheduled_at, timezone, status, analytics_data, metadata) VALUES
(
    '880e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440001',
    '770e8400-e29b-41d4-a716-446655440001',
    'Introducing Our New Social Media Scheduler',
    'We''re excited to announce the launch of our AI-powered social media scheduler! 🚀 Streamline your content strategy with intelligent platform optimization and seamless scheduling. Perfect for businesses ready to scale their social media presence. #VrooliLaunch #AISocialMedia #ProductLaunch',
    '{
        "twitter": "🚀 Excited to announce our new AI-powered social media scheduler! Streamline your content strategy with intelligent platform optimization. Perfect for scaling your social media presence. #VrooliLaunch #AISocialMedia #ProductLaunch",
        "linkedin": "We''re thrilled to introduce our latest innovation: an AI-powered social media scheduler designed for businesses serious about scaling their social media presence.\n\nKey features:\n• Intelligent platform optimization\n• Seamless multi-platform scheduling\n• Advanced analytics and insights\n• Team collaboration tools\n\nPerfect for marketing teams, agencies, and growing businesses. Learn more about how we''re revolutionizing social media management.\n\n#VrooliLaunch #AISocialMedia #ProductLaunch #MarketingTechnology",
        "facebook": "🎉 Big announcement! We''ve just launched our AI-powered social media scheduler, and we can''t wait for you to try it!\n\nThis isn''t just another scheduling tool - it''s your intelligent social media assistant that:\n✅ Optimizes content for each platform automatically\n✅ Suggests the best posting times for your audience\n✅ Maintains your brand consistency across all channels\n✅ Provides deep analytics to improve your strategy\n\nWhether you''re a small business owner, marketing manager, or agency professional, this tool will transform how you manage your social media presence.\n\n#VrooliLaunch #AISocialMedia #ProductLaunch"
    }',
    ARRAY['twitter', 'linkedin', 'facebook'],
    '2024-07-15 14:00:00',
    'America/New_York',
    'posted',
    '{"total_engagement": 245, "total_reach": 8500, "total_impressions": 12000}',
    '{"generated_by": "campaign-content-studio", "optimized_by": "ollama", "brand_checked": true}'
),
(
    '880e8400-e29b-41d4-a716-446655440002',
    '550e8400-e29b-41d4-a716-446655440001',
    '770e8400-e29b-41d4-a716-446655440001',
    'Tips for Effective Social Media Scheduling',
    'Master the art of social media scheduling with these proven strategies! 📅 Timing is everything when it comes to maximizing engagement. Our AI analyzes your audience behavior to suggest optimal posting times. What''s your biggest social media challenge? Share in the comments! #SocialMediaTips #VrooliLaunch #MarketingStrategy',
    '{
        "twitter": "📅 Master social media scheduling with these proven strategies! Timing = everything for max engagement. Our AI analyzes your audience to suggest optimal posting times. What''s your biggest social media challenge? 🤔 #SocialMediaTips #VrooliLaunch",
        "linkedin": "Master the Art of Social Media Scheduling\n\nEffective social media scheduling isn''t just about consistency - it''s about strategic timing that maximizes your reach and engagement.\n\nHere are the key factors to consider:\n\n🎯 Audience Analysis: Understanding when your specific audience is most active\n📊 Platform Optimization: Each platform has unique peak times and behaviors\n🔄 Content Variety: Balancing promotional, educational, and engaging content\n⏰ Consistency: Building audience expectations through regular posting\n\nOur AI-powered scheduler takes the guesswork out of timing by analyzing your audience behavior patterns and suggesting optimal posting windows.\n\nWhat''s your biggest challenge with social media scheduling? I''d love to hear your thoughts in the comments.\n\n#SocialMediaTips #VrooliLaunch #MarketingStrategy #DigitalMarketing",
        "instagram": "Master the art of social media scheduling! 📅✨\n\nTiming is everything when it comes to maximizing engagement. Our AI analyzes your audience behavior to suggest the perfect posting times for YOUR specific followers.\n\n💡 Pro tips:\n• Post when your audience is most active\n• Maintain consistent posting schedule\n• Mix up your content types\n• Engage with comments quickly\n\nWhat''s your biggest social media challenge? Tell us in the comments! 👇\n\n#SocialMediaTips #VrooliLaunch #MarketingStrategy #InstagramTips #SocialMediaScheduling"
    }',
    ARRAY['twitter', 'linkedin', 'instagram'],
    '2024-07-20 11:30:00',
    'America/New_York',
    'scheduled',
    '{}',
    '{"content_type": "educational", "target_audience": "marketers", "engagement_goal": "comments"}'
);

-- Insert corresponding platform posts for the posted scheduled post
INSERT INTO platform_posts (id, scheduled_post_id, social_account_id, platform, platform_post_id, optimized_content, status, posted_at, engagement_data) VALUES
(
    '990e8400-e29b-41d4-a716-446655440001',
    '880e8400-e29b-41d4-a716-446655440001',
    '660e8400-e29b-41d4-a716-446655440001',
    'twitter',
    'tweet_1234567890',
    '🚀 Excited to announce our new AI-powered social media scheduler! Streamline your content strategy with intelligent platform optimization. Perfect for scaling your social media presence. #VrooliLaunch #AISocialMedia #ProductLaunch',
    'posted',
    '2024-07-15 14:00:00',
    '{"likes": 45, "retweets": 12, "replies": 8, "impressions": 2500}'
),
(
    '990e8400-e29b-41d4-a716-446655440002',
    '880e8400-e29b-41d4-a716-446655440001',
    '660e8400-e29b-41d4-a716-446655440002',
    'linkedin',
    'linkedin_post_abcd123',
    'We''re thrilled to introduce our latest innovation: an AI-powered social media scheduler designed for businesses serious about scaling their social media presence.\n\nKey features:\n• Intelligent platform optimization\n• Seamless multi-platform scheduling\n• Advanced analytics and insights\n• Team collaboration tools\n\nPerfect for marketing teams, agencies, and growing businesses. Learn more about how we''re revolutionizing social media management.\n\n#VrooliLaunch #AISocialMedia #ProductLaunch #MarketingTechnology',
    'posted',
    '2024-07-15 14:01:00',
    '{"likes": 89, "comments": 15, "shares": 23, "impressions": 4200}'
),
(
    '990e8400-e29b-41d4-a716-446655440003',
    '880e8400-e29b-41d4-a716-446655440001',
    '660e8400-e29b-41d4-a716-446655440003',
    'facebook',
    'facebook_post_xyz789',
    '🎉 Big announcement! We''ve just launched our AI-powered social media scheduler, and we can''t wait for you to try it!\n\nThis isn''t just another scheduling tool - it''s your intelligent social media assistant that:\n✅ Optimizes content for each platform automatically\n✅ Suggests the best posting times for your audience\n✅ Maintains your brand consistency across all channels\n✅ Provides deep analytics to improve your strategy\n\nWhether you''re a small business owner, marketing manager, or agency professional, this tool will transform how you manage your social media presence.\n\n#VrooliLaunch #AISocialMedia #ProductLaunch',
    'posted',
    '2024-07-15 14:02:00',
    '{"likes": 156, "comments": 32, "shares": 28, "reach": 3800}'
);

-- Insert sample analytics data
INSERT INTO post_analytics (platform_post_id, metrics, engagement_rate, reach, impressions, clicks) VALUES
(
    '990e8400-e29b-41d4-a716-446655440001',
    '{"likes": 45, "retweets": 12, "replies": 8, "impressions": 2500, "profile_clicks": 15, "url_clicks": 22}',
    0.0260, -- 65 engagements / 2500 impressions
    2200,
    2500,
    37
),
(
    '990e8400-e29b-41d4-a716-446655440002',
    '{"likes": 89, "comments": 15, "shares": 23, "impressions": 4200, "profile_clicks": 28, "url_clicks": 45}',
    0.0302, -- 127 engagements / 4200 impressions
    3800,
    4200,
    73
),
(
    '990e8400-e29b-41d4-a716-446655440003',
    '{"likes": 156, "comments": 32, "shares": 28, "reach": 3800, "link_clicks": 67, "page_likes": 8}',
    0.0568, -- 216 engagements / 3800 reach
    3800,
    5300,
    75
);

-- Insert sample media files
INSERT INTO media_files (id, user_id, filename, original_filename, file_type, file_size, dimensions, minio_path, public_url, platform_variants) VALUES
(
    'aa0e8400-e29b-41d4-a716-446655440001',
    '550e8400-e29b-41d4-a716-446655440001',
    'product_launch_hero_1920x1080.jpg',
    'product-launch-hero.jpg',
    'image/jpeg',
    1248576,
    '{"width": 1920, "height": 1080}',
    'social-media-assets/550e8400-e29b-41d4-a716-446655440001/product_launch_hero_1920x1080.jpg',
    'https://assets.vrooli.com/social-media-assets/550e8400-e29b-41d4-a716-446655440001/product_launch_hero_1920x1080.jpg',
    '{
        "twitter": {"url": "https://assets.vrooli.com/social-media-assets/550e8400-e29b-41d4-a716-446655440001/product_launch_hero_1200x675.jpg", "dimensions": {"width": 1200, "height": 675}},
        "instagram": {"url": "https://assets.vrooli.com/social-media-assets/550e8400-e29b-41d4-a716-446655440001/product_launch_hero_1080x1080.jpg", "dimensions": {"width": 1080, "height": 1080}},
        "linkedin": {"url": "https://assets.vrooli.com/social-media-assets/550e8400-e29b-41d4-a716-446655440001/product_launch_hero_1200x627.jpg", "dimensions": {"width": 1200, "height": 627}},
        "facebook": {"url": "https://assets.vrooli.com/social-media-assets/550e8400-e29b-41d4-a716-446655440001/product_launch_hero_1200x630.jpg", "dimensions": {"width": 1200, "height": 630}}
    }'
);

-- Insert sample activity logs
INSERT INTO activity_logs (user_id, action, resource_type, resource_id, details, ip_address) VALUES
(
    '550e8400-e29b-41d4-a716-446655440001',
    'post_scheduled',
    'post',
    '880e8400-e29b-41d4-a716-446655440001',
    '{"platforms": ["twitter", "linkedin", "facebook"], "scheduled_for": "2024-07-15T14:00:00Z"}',
    '192.168.1.100'
),
(
    '550e8400-e29b-41d4-a716-446655440001',
    'post_published',
    'post',
    '880e8400-e29b-41d4-a716-446655440001',
    '{"platforms_posted": ["twitter", "linkedin", "facebook"], "posted_at": "2024-07-15T14:00:00Z"}',
    '192.168.1.100'
),
(
    '550e8400-e29b-41d4-a716-446655440001',
    'account_connected',
    'social_account',
    '660e8400-e29b-41d4-a716-446655440001',
    '{"platform": "twitter", "username": "vrooli_demo"}',
    '192.168.1.100'
);

-- Insert development configuration data
INSERT INTO system_jobs (job_type, job_data, status, priority, scheduled_for) VALUES
('sync_platform_metrics', '{"platform": "all", "lookback_days": 7}', 'pending', 2, NOW() + INTERVAL '1 hour'),
('optimize_posting_times', '{"user_id": "550e8400-e29b-41d4-a716-446655440001", "analysis_period_days": 30}', 'pending', 3, NOW() + INTERVAL '6 hours'),
('cleanup_old_analytics', '{"retention_days": 90}', 'pending', 1, NOW() + INTERVAL '1 day');

-- Add comments for seed data documentation
COMMENT ON TABLE users IS 'Seed data includes demo and agency test accounts';
COMMENT ON TABLE social_accounts IS 'Seed data includes connected accounts for testing all major platforms';
COMMENT ON TABLE campaigns IS 'Seed data includes active campaigns for development testing';
COMMENT ON TABLE scheduled_posts IS 'Seed data includes both posted and scheduled posts with platform variants';

-- Update schema migrations
INSERT INTO schema_migrations (version) VALUES ('002_seed_data');

-- Display seed data summary
DO $$
DECLARE
    user_count INTEGER;
    account_count INTEGER;
    post_count INTEGER;
    campaign_count INTEGER;
BEGIN
    SELECT COUNT(*) INTO user_count FROM users;
    SELECT COUNT(*) INTO account_count FROM social_accounts;
    SELECT COUNT(*) INTO post_count FROM scheduled_posts;
    SELECT COUNT(*) INTO campaign_count FROM campaigns;
    
    RAISE NOTICE '🌱 Seed data inserted successfully:';
    RAISE NOTICE '   👤 Users: %', user_count;
    RAISE NOTICE '   📱 Social Accounts: %', account_count;
    RAISE NOTICE '   📝 Scheduled Posts: %', post_count;
    RAISE NOTICE '   📊 Campaigns: %', campaign_count;
    RAISE NOTICE '   🎯 Ready for development and testing!';
END $$;