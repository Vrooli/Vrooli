-- Seed Data for Landing Manager

-- Insert default admin user (OT-P0-008: ADMIN-AUTH)
-- Email: admin@localhost
-- Password: changeme123
-- IMPORTANT: Change this password in production!
INSERT INTO admin_users (email, password_hash) VALUES
('admin@localhost', '$2a$10$nhmpbhFPQUZZwEH.qaYHCeiKBWDvr8z5Z7eM4v62MmNwm.0N.5xeG')
ON CONFLICT (email) DO NOTHING;

-- Insert default A/B testing variants (OT-P0-014 through OT-P0-018)
INSERT INTO variants (slug, name, description, weight, status) VALUES
('control', 'Control (Original)', 'Original landing page design', 50, 'active'),
('variant-a', 'Variant A', 'Experimental variant A', 50, 'active')
ON CONFLICT (slug) DO NOTHING;

-- Default axis mappings for starter variants
WITH control AS (
    SELECT id FROM variants WHERE slug = 'control'
)
INSERT INTO variant_axes (variant_id, axis_id, variant_value)
VALUES
((SELECT id FROM control), 'persona', 'ops_leader'),
((SELECT id FROM control), 'jtbd', 'launch_bundle'),
((SELECT id FROM control), 'conversionStyle', 'demo_led')
ON CONFLICT (variant_id, axis_id) DO UPDATE
SET variant_value = EXCLUDED.variant_value,
    updated_at = NOW();

WITH variant_a AS (
    SELECT id FROM variants WHERE slug = 'variant-a'
)
INSERT INTO variant_axes (variant_id, axis_id, variant_value)
VALUES
((SELECT id FROM variant_a), 'persona', 'automation_freelancer'),
((SELECT id FROM variant_a), 'jtbd', 'scale_services'),
((SELECT id FROM variant_a), 'conversionStyle', 'self_serve')
ON CONFLICT (variant_id, axis_id) DO UPDATE
SET variant_value = EXCLUDED.variant_value,
    updated_at = NOW();

-- Insert default content sections for control variant (OT-P0-012, OT-P0-013)
INSERT INTO content_sections (variant_id, section_type, content, "order", enabled) VALUES
(1, 'hero', '{"headline": "Build Landing Pages in Minutes", "subheadline": "Create beautiful, conversion-optimized landing pages with A/B testing and analytics built-in", "cta_text": "Get Started Free", "cta_url": "/signup", "background_color": "#0f172a"}', 1, TRUE),
(1, 'features', '{"title": "Everything You Need", "items": [{"title": "A/B Testing", "description": "Test variants and optimize conversions", "icon": "Zap"}, {"title": "Analytics", "description": "Track visitor behavior and metrics", "icon": "BarChart"}, {"title": "Stripe Integration", "description": "Accept payments instantly", "icon": "CreditCard"}]}', 2, TRUE),
(1, 'pricing', '{"title": "Simple Pricing", "plans": [{"name": "Starter", "price": "$29", "features": ["5 landing pages", "Basic analytics", "Email support"], "cta_text": "Start Free Trial"}, {"name": "Pro", "price": "$99", "features": ["Unlimited pages", "Advanced analytics", "Priority support", "Custom domains"], "cta_text": "Get Started", "highlighted": true}]}', 3, TRUE),
(1, 'cta', '{"headline": "Ready to Launch Your Landing Page?", "subheadline": "Join thousands of marketers using Landing Manager", "cta_text": "Start Building Now", "cta_url": "/signup"}', 4, TRUE)
ON CONFLICT DO NOTHING;

-- Bundle products & prices (business suite)
INSERT INTO bundle_products (bundle_key, bundle_name, stripe_product_id, credits_per_usd, display_credits_multiplier, display_credits_label, environment, metadata)
VALUES
('business_suite', 'Vrooli Business Suite', 'prod_business_suite', 1000000, 0.001, 'credits', 'production', '{"description":"Browser Automation Studio bundle"}')
ON CONFLICT (bundle_key) DO UPDATE SET
    bundle_name = EXCLUDED.bundle_name,
    stripe_product_id = EXCLUDED.stripe_product_id,
    credits_per_usd = EXCLUDED.credits_per_usd,
    display_credits_multiplier = EXCLUDED.display_credits_multiplier,
    display_credits_label = EXCLUDED.display_credits_label,
    metadata = EXCLUDED.metadata,
    updated_at = NOW();

WITH prod AS (
    SELECT id FROM bundle_products WHERE bundle_key = 'business_suite'
)
INSERT INTO bundle_prices (
    product_id, stripe_price_id, plan_name, plan_tier, billing_interval, amount_cents, currency,
    intro_enabled, intro_type, intro_amount_cents, intro_periods, intro_price_lookup_key,
    monthly_included_credits, one_time_bonus_credits, plan_rank, bonus_type, metadata, display_weight
) VALUES
((SELECT id FROM prod), 'price_solo_monthly', 'Solo Monthly', 'solo', 'month', 4900, 'usd', TRUE, 'flat_amount', 100, 1, 'solo_monthly_intro', 5, 0, 1, 'none', '{"features":["Solo workspace","1k monthly credits"]}', 30),
((SELECT id FROM prod), 'price_solo_yearly', 'Solo Yearly', 'solo', 'year', 49900, 'usd', FALSE, NULL, NULL, 0, NULL, 60, 50, 1, 'yearly_bonus', '{"features":["2 months free equivalent","Bonus credits"]}', 10),
((SELECT id FROM prod), 'price_pro_monthly', 'Pro Monthly', 'pro', 'month', 14900, 'usd', TRUE, 'flat_amount', 100, 1, 'pro_monthly_intro', 25, 0, 2, 'none', '{"features":["Team workflows","Priority support"]}', 40),
((SELECT id FROM prod), 'price_pro_yearly', 'Pro Yearly', 'pro', 'year', 149900, 'usd', FALSE, NULL, NULL, 0, NULL, 320, 200, 2, 'yearly_bonus', '{"features":["Advanced automations","Dedicated success"]}', 20),
((SELECT id FROM prod), 'price_studio_monthly', 'Studio Monthly', 'studio', 'month', 34900, 'usd', TRUE, 'flat_amount', 100, 1, 'studio_monthly_intro', 75, 0, 3, 'none', '{"features":["Studio credits","Multi-seat"]}', 50),
((SELECT id FROM prod), 'price_studio_yearly', 'Studio Yearly', 'studio', 'year', 349900, 'usd', FALSE, NULL, NULL, 0, NULL, 900, 500, 3, 'yearly_bonus', '{"features":["Enterprise integrations","VIP onboarding"]}', 30)
ON CONFLICT (stripe_price_id) DO UPDATE SET
    plan_name = EXCLUDED.plan_name,
    plan_tier = EXCLUDED.plan_tier,
    billing_interval = EXCLUDED.billing_interval,
    amount_cents = EXCLUDED.amount_cents,
    intro_enabled = EXCLUDED.intro_enabled,
    intro_type = EXCLUDED.intro_type,
    intro_amount_cents = EXCLUDED.intro_amount_cents,
    intro_periods = EXCLUDED.intro_periods,
    intro_price_lookup_key = EXCLUDED.intro_price_lookup_key,
    monthly_included_credits = EXCLUDED.monthly_included_credits,
    one_time_bonus_credits = EXCLUDED.one_time_bonus_credits,
    plan_rank = EXCLUDED.plan_rank,
    bonus_type = EXCLUDED.bonus_type,
    metadata = EXCLUDED.metadata,
    display_weight = EXCLUDED.display_weight,
    updated_at = NOW();

-- Download assets for Browser Automation Studio
INSERT INTO download_assets (bundle_key, platform, artifact_url, release_version, release_notes, checksum, requires_entitlement, metadata)
VALUES
('business_suite', 'windows', 'https://downloads.vrooli.local/business-suite/win/VrooliBusinessSuiteSetup.exe', '1.0.0', 'Initial GA release with Browser Automation Studio.', 'sha256-win-placeholder', TRUE, '{"size_mb":210}'),
('business_suite', 'mac', 'https://downloads.vrooli.local/business-suite/mac/VrooliBusinessSuite.dmg', '1.0.0', 'Universal build for Apple Silicon and Intel.', 'sha256-mac-placeholder', TRUE, '{"size_mb":190}'),
('business_suite', 'linux', 'https://downloads.vrooli.local/business-suite/linux/vrooli-business-suite.tar.gz', '1.0.0', 'AppImage bundle tested on Ubuntu/Debian.', 'sha256-linux-placeholder', TRUE, '{"size_mb":205}')
ON CONFLICT (bundle_key, platform) DO UPDATE SET
    artifact_url = EXCLUDED.artifact_url,
    release_version = EXCLUDED.release_version,
    release_notes = EXCLUDED.release_notes,
    checksum = EXCLUDED.checksum,
    requires_entitlement = EXCLUDED.requires_entitlement,
    metadata = EXCLUDED.metadata,
    updated_at = NOW();

-- Seed credit wallets for demo accounts
INSERT INTO credit_wallets (customer_email, balance_credits, bonus_credits, updated_at)
VALUES
('solo@demo.vrooli', 5000000, 0, NOW()),
('pro@demo.vrooli', 25000000, 1000000, NOW()),
('studio@demo.vrooli', 75000000, 5000000, NOW())
ON CONFLICT (customer_email) DO UPDATE SET
    balance_credits = EXCLUDED.balance_credits,
    bonus_credits = EXCLUDED.bonus_credits,
    updated_at = NOW();
