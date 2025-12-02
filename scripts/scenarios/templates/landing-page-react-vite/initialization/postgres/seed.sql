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
(1, 'hero', '{"headline": "Launch Browser Automation Studio", "subheadline": "Ship automation bundles with analytics, downloads, and admin rails on day one.", "cta_text": "Get Started", "cta_url": "/checkout?plan=pro", "background_color": "#0f172a"}', 1, TRUE),
(1, 'video', '{"title": "Tour the landing runtime", "videoUrl": "https://www.youtube.com/watch?v=dQw4w9WgXcQ", "thumbnailUrl": "/assets/fallback/video-thumb.png", "caption": "Weighted variants, analytics, and downloads sync in this 4-minute overview."}', 2, TRUE),
(1, 'features', '{"title": "Everything You Need", "items": [{"title": "Variant routing", "description": "Control traffic weights per persona", "icon": "Zap"}, {"title": "Analytics", "description": "Track CTA, scroll, and download events", "icon": "BarChart"}, {"title": "Download gating", "description": "Gate installers behind active entitlements", "icon": "Shield"}]}', 3, TRUE),
(1, 'testimonials', '{"title": "Loved by operators", "subtitle": "Quotes from internal dogfood launches", "testimonials": [{"name": "Marina Patel", "role": "VP Operations", "company": "Clause", "content": "Downloads + pricing landed perfectly on the first deploy.", "rating": 5}, {"name": "Devon Brooks", "role": "Founder", "company": "Atlas Automations", "content": "The download rail let us ship macOS + Windows builds without custom code.", "rating": 5}] }', 4, TRUE),
(1, 'pricing', '{"title": "Simple Pricing", "plans": [{"name": "Solo", "price": "$49", "features": ["Solo workspace", "5M credits", "Email support"], "cta_text": "Start for $1"}, {"name": "Pro", "price": "$149", "features": ["Team workflows", "Priority support", "Desktop downloads"], "cta_text": "Upgrade", "highlighted": true}, {"name": "Studio", "price": "$349", "features": ["Unlimited automations", "Dedicated architect"], "cta_text": "Talk to sales"}]}', 5, TRUE),
(1, 'downloads', '{"title": "Download Browser Automation Studio", "subtitle": "macOS, Windows, Linux, and store links inherit entitlement gating by default."}', 6, TRUE),
(1, 'faq', '{"title": "Launch guardrails", "subtitle": "Answers teams need before sending paid traffic", "faqs": [{"question": "How do downloads verify entitlements?", "answer": "Each installer request hits /api/v1/downloads with the subscriber email."}, {"question": "Can we restyle sections?", "answer": "Yes, every section respects styling.json tokens."}, {"question": "Do analytics include downloads?", "answer": "Download events are captured per variant alongside CTA clicks."}]}', 7, TRUE),
(1, 'cta', '{"headline": "Ready to launch your bundle?", "subheadline": "Book a walkthrough with the landing team.", "cta_text": "Book demo", "cta_url": "/contact"}', 8, TRUE),
(1, 'footer', '{"company_name": "Vrooli Business Suite", "tagline": "Clause-grade landing runtime with analytics + downloads in lockstep.", "columns": [{"title": "Product", "links": [{"label": "Features", "url": "#features"}, {"label": "Pricing", "url": "#pricing"}]}, {"title": "Company", "links": [{"label": "Docs", "url": "/docs"}, {"label": "Careers", "url": "/careers"}]}], "social_links": {"github": "https://github.com/vrooli", "twitter": "https://twitter.com/vrooli"}}', 9, TRUE)
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
    monthly_included_credits, one_time_bonus_credits, plan_rank, bonus_type, kind, is_variable_amount, display_enabled, metadata, display_weight
) VALUES
((SELECT id FROM prod), 'price_solo_monthly', 'Solo Monthly', 'solo', 'month', 4900, 'usd', TRUE, 'flat_amount', 100, 1, 'solo_monthly_intro', 5, 0, 1, 'none', 'subscription', FALSE, TRUE, '{"features":["Solo workspace","1k monthly credits"]}', 30),
((SELECT id FROM prod), 'price_solo_yearly', 'Solo Yearly', 'solo', 'year', 49900, 'usd', FALSE, NULL, NULL, 0, NULL, 60, 50, 1, 'yearly_bonus', 'subscription', FALSE, TRUE, '{"features":["2 months free equivalent","Bonus credits"]}', 10),
((SELECT id FROM prod), 'price_pro_monthly', 'Pro Monthly', 'pro', 'month', 14900, 'usd', TRUE, 'flat_amount', 100, 1, 'pro_monthly_intro', 25, 0, 2, 'none', 'subscription', FALSE, TRUE, '{"features":["Team workflows","Priority support"]}', 40),
((SELECT id FROM prod), 'price_pro_yearly', 'Pro Yearly', 'pro', 'year', 149900, 'usd', FALSE, NULL, NULL, 0, NULL, 320, 200, 2, 'yearly_bonus', 'subscription', FALSE, TRUE, '{"features":["Advanced automations","Dedicated success"]}', 20),
((SELECT id FROM prod), 'price_studio_monthly', 'Studio Monthly', 'studio', 'month', 34900, 'usd', TRUE, 'flat_amount', 100, 1, 'studio_monthly_intro', 75, 0, 3, 'none', 'subscription', FALSE, TRUE, '{"features":["Studio credits","Multi-seat"]}', 50),
((SELECT id FROM prod), 'price_studio_yearly', 'Studio Yearly', 'studio', 'year', 349900, 'usd', FALSE, NULL, NULL, 0, NULL, 900, 500, 3, 'yearly_bonus', 'subscription', FALSE, TRUE, '{"features":["Enterprise integrations","VIP onboarding"]}', 30),
((SELECT id FROM prod), 'price_credits_topup', 'Credits Top-Up', 'credits', 'one_time', 0, 'usd', FALSE, NULL, NULL, 0, NULL, 0, 0, 0, 'none', 'credits_topup', TRUE, TRUE, '{"description":"Add credits via Stripe checkout"}', 5),
((SELECT id FROM prod), 'price_supporter_contribution', 'Supporter Contribution', 'donation', 'one_time', 0, 'usd', FALSE, NULL, NULL, 0, NULL, 0, 0, 0, 'none', 'supporter_contribution', TRUE, TRUE, '{"grants_credits": false, "grants_entitlements": false, "description":"Support Browser Automation Studio"}', 1)
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
    kind = EXCLUDED.kind,
    is_variable_amount = EXCLUDED.is_variable_amount,
    metadata = EXCLUDED.metadata,
    display_enabled = EXCLUDED.display_enabled,
    display_weight = EXCLUDED.display_weight,
    updated_at = NOW();

-- Download apps + platform installers
INSERT INTO download_apps (bundle_key, app_key, name, tagline, description, install_overview, install_steps, storefronts, metadata, display_order)
VALUES
('business_suite', 'automation_studio', 'Browser Automation Studio', 'Full desktop automation workbench', 'Ships the flagship Browser Automation Studio desktop client with entitlement-backed installers and telemetry baked in.', 'Download the installer for your OS and sign in with the email tied to your active subscription.', '["Download the installer for your OS","Launch the setup wizard","Sign in with your subscription email to unlock downloads"]', '[{"store":"app_store","label":"macOS App Store","url":"https://apps.apple.com/app/id000000","badge":"Download on the App Store"}]', '{"bundle":"business_suite"}', 1),
('business_suite', 'command_center', 'Vrooli Command Center', 'Mobile companion for monitoring bots', 'Native companion apps for leaders who need to approve automations, monitor download health, and trigger bundles while on the go.', 'Install from your preferred app store or download the desktop utilities when you need offline orchestration.', '["Install from the App Store or Google Play","Enable notifications for entitlement updates","Link to your workspace to sync download telemetry"]', '[{"store":"app_store","label":"Apple App Store","url":"https://apps.apple.com/app/id000111","badge":"Download on the App Store"},{"store":"play_store","label":"Google Play","url":"https://play.google.com/store/apps/details?id=vrooli.command","badge":"Get it on Google Play"}]', '{"bundle":"business_suite"}', 2)
ON CONFLICT (bundle_key, app_key) DO UPDATE SET
    name = EXCLUDED.name,
    tagline = EXCLUDED.tagline,
    description = EXCLUDED.description,
    install_overview = EXCLUDED.install_overview,
    install_steps = EXCLUDED.install_steps,
    storefronts = EXCLUDED.storefronts,
    metadata = EXCLUDED.metadata,
    display_order = EXCLUDED.display_order,
    updated_at = NOW();

INSERT INTO download_assets (bundle_key, app_key, platform, artifact_url, release_version, release_notes, checksum, requires_entitlement, metadata)
VALUES
('business_suite', 'automation_studio', 'windows', 'https://downloads.vrooli.local/business-suite/win/VrooliBusinessStudioSetup.exe', '1.2.0', 'Includes fresh launch console + webdriver upgrades.', 'sha256-win-placeholder', TRUE, '{"size_mb":230}'),
('business_suite', 'automation_studio', 'mac', 'https://downloads.vrooli.local/business-suite/mac/VrooliBusinessStudio.dmg', '1.2.0', 'Signed universal build for Apple Silicon + Intel.', 'sha256-mac-placeholder', TRUE, '{"size_mb":215}'),
('business_suite', 'automation_studio', 'linux', 'https://downloads.vrooli.local/business-suite/linux/vrooli-business-studio.tar.gz', '1.2.0', 'AppImage bundle tested on Ubuntu/Debian.', 'sha256-linux-placeholder', TRUE, '{"size_mb":225}'),
('business_suite', 'command_center', 'windows', 'https://downloads.vrooli.local/command-center/win/VrooliCommandCenter.exe', '0.9.5', 'Preview build with entitlement notifications.', 'sha256-win-cc', FALSE, '{"size_mb":120,"channel":"beta"}'),
('business_suite', 'command_center', 'mac', 'https://downloads.vrooli.local/command-center/mac/VrooliCommandCenter.dmg', '0.9.5', 'Signed build with push notification helpers.', 'sha256-mac-cc', FALSE, '{"size_mb":115,"channel":"beta"}'),
('business_suite', 'command_center', 'linux', 'https://downloads.vrooli.local/command-center/linux/VrooliCommandCenter.tar.gz', '0.9.5', 'Preview release for Debian/Ubuntu.', 'sha256-linux-cc', FALSE, '{"size_mb":118,"channel":"beta"}')
ON CONFLICT (bundle_key, app_key, platform) DO UPDATE SET
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

-- Seed payment settings row so admin UI can update it without manual SQL
INSERT INTO payment_settings (id, publishable_key, secret_key, webhook_secret, dashboard_url)
VALUES (1, NULL, NULL, NULL, NULL)
ON CONFLICT (id) DO UPDATE SET
    publishable_key = EXCLUDED.publishable_key,
    secret_key = EXCLUDED.secret_key,
    webhook_secret = EXCLUDED.webhook_secret,
    dashboard_url = EXCLUDED.dashboard_url,
    updated_at = NOW();
