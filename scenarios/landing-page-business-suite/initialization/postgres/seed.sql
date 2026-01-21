-- Seed Data for Landing Manager
--
-- NOTE: Variant, section, and branding configuration is now stored in JSON files
-- (.vrooli/variants/*.json and .vrooli/branding.json) and loaded into memory at startup.
-- This seed file only seeds runtime/dynamic data.

-- Insert default admin user (OT-P0-008: ADMIN-AUTH)
-- Email: admin@localhost
-- Password: changeme123
-- IMPORTANT: Change this password in production!
INSERT INTO admin_users (email, password_hash) VALUES
('admin@localhost', '$2a$10$nhmpbhFPQUZZwEH.qaYHCeiKBWDvr8z5Z7eM4v62MmNwm.0N.5xeG')
ON CONFLICT (email) DO NOTHING;

-- Bundle products & prices (business suite)
INSERT INTO bundle_products (bundle_key, bundle_name, stripe_product_id, credits_per_usd, display_credits_multiplier, display_credits_label, environment, metadata)
VALUES
('business_suite', 'Vrooli Business Suite (Silent Founder OS)', 'prod_business_suite', 1000000, 0.001, 'credits', 'production', '{"description":"Vrooli Ascension today; expanding suite tomorrow"}')
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
((SELECT id FROM prod), 'price_free_monthly', 'Free Monthly', 'free', 'month', 0, 'usd', FALSE, NULL, NULL, 0, NULL, 50, 0, 0, 'none', 'subscription', FALSE, TRUE, '{"features":["50 runs/month","Replay viewer (watermark)"],"badge":"Start free"}', 5),
((SELECT id FROM prod), 'price_solo_monthly', 'Solo Monthly', 'solo', 'month', 2900, 'usd', FALSE, NULL, NULL, 0, NULL, 200, 0, 1, 'none', 'subscription', FALSE, TRUE, '{"features":["200 runs/month","MP4 export (watermark)","Async support"],"cta_label":"Upgrade to Solo"}', 20),
((SELECT id FROM prod), 'price_pro_monthly', 'Pro Monthly', 'pro', 'month', 7900, 'usd', FALSE, NULL, NULL, 0, NULL, 1000000, 0, 2, 'none', 'subscription', FALSE, TRUE, '{"features":["Unlimited runs (fair use)","MP4 without watermark","CI hooks","Limited agent loops","Early UX metrics access"],"badge":"Recommended","highlight":true}', 40),
((SELECT id FROM prod), 'price_studio_monthly', 'Studio Monthly', 'studio', 'month', 19900, 'usd', FALSE, NULL, NULL, 0, NULL, 2000000, 0, 3, 'none', 'subscription', FALSE, TRUE, '{"features":["Custom branding in replays","More agent loop concurrency","Multi-seat","Priority support"]}', 25),
((SELECT id FROM prod), 'price_business_monthly', 'Business Monthly', 'business', 'month', 49900, 'usd', FALSE, NULL, NULL, 0, NULL, 4000000, 0, 4, 'none', 'subscription', FALSE, TRUE, '{"features":["Unlimited agent loops","API + webhooks","Reliability options"]}', 10),
((SELECT id FROM prod), 'price_solo_yearly', 'Solo Yearly', 'solo', 'year', 29000, 'usd', FALSE, NULL, NULL, 0, NULL, 200, 0, 1, 'yearly_bonus', 'subscription', FALSE, TRUE, '{"features":["2 months free equivalent","MP4 export (watermark)"]}', 10),
((SELECT id FROM prod), 'price_pro_yearly', 'Pro Yearly', 'pro', 'year', 79000, 'usd', FALSE, NULL, NULL, 0, NULL, 1000000, 0, 2, 'yearly_bonus', 'subscription', FALSE, TRUE, '{"features":["MP4 without watermark","CI hooks","Limited agent loops"]}', 20),
((SELECT id FROM prod), 'price_studio_yearly', 'Studio Yearly', 'studio', 'year', 199000, 'usd', FALSE, NULL, NULL, 0, NULL, 2000000, 0, 3, 'yearly_bonus', 'subscription', FALSE, TRUE, '{"features":["Custom branding in replays","More agent loop concurrency","Multi-seat studio"]}', 30),
((SELECT id FROM prod), 'price_business_yearly', 'Business Yearly', 'business', 'year', 499000, 'usd', FALSE, NULL, NULL, 0, NULL, 4000000, 0, 4, 'yearly_bonus', 'subscription', FALSE, TRUE, '{"features":["Unlimited agent loops","API + webhooks","Reliability + SSO prep"]}', 5),
((SELECT id FROM prod), 'price_credits_topup', 'Credits Top-Up', 'credits', 'one_time', 0, 'usd', FALSE, NULL, NULL, 0, NULL, 0, 0, 0, 'none', 'credits_topup', TRUE, TRUE, '{"description":"Add credits via Stripe checkout"}', 5),
((SELECT id FROM prod), 'price_supporter_contribution', 'Supporter Contribution', 'donation', 'one_time', 0, 'usd', FALSE, NULL, NULL, 0, NULL, 0, 0, 0, 'none', 'supporter_contribution', TRUE, TRUE, '{"grants_credits": false, "grants_entitlements": false, "description":"Support Vrooli Ascension"}', 1)
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
INSERT INTO download_apps (bundle_key, app_key, name, tagline, description, icon_url, screenshot_url, install_overview, install_steps, storefronts, metadata, display_order)
VALUES
('business_suite', 'automation_studio', 'Vrooli Ascension', 'Desktop app for browser automation and cinematic workflow replays.', 'Automate repetitive browser tasks, record workflows passively, and export studio-quality video replays of your work.', NULL, NULL, 'Pick your platform below. The app verifies your subscription after install.', '["Run the installer","Follow the setup wizard","Sign in with your subscription email"]', '[{"store":"app_store","label":"macOS App Store","url":"https://apps.apple.com/app/id000000","badge":"Download on the App Store"}]', '{"bundle":"business_suite"}', 1)
ON CONFLICT (bundle_key, app_key) DO UPDATE SET
    name = EXCLUDED.name,
    tagline = EXCLUDED.tagline,
    description = EXCLUDED.description,
    icon_url = EXCLUDED.icon_url,
    screenshot_url = EXCLUDED.screenshot_url,
    install_overview = EXCLUDED.install_overview,
    install_steps = EXCLUDED.install_steps,
    storefronts = EXCLUDED.storefronts,
    metadata = EXCLUDED.metadata,
    display_order = EXCLUDED.display_order,
    updated_at = NOW();

INSERT INTO download_assets (bundle_key, app_key, platform, variant_key, artifact_url, release_version, release_notes, checksum, requires_entitlement, metadata, display_order)
VALUES
-- Windows variants
('business_suite', 'automation_studio', 'windows', 'installer', 'https://downloads.vrooli.local/business-suite/win/VrooliAscensionSetup.exe', '1.0.0', NULL, 'sha256-win-exe', TRUE, '{"size_mb":210}', 1),
('business_suite', 'automation_studio', 'windows', 'portable', 'https://downloads.vrooli.local/business-suite/win/VrooliAscension-portable.zip', '1.0.0', NULL, 'sha256-win-zip', TRUE, '{"size_mb":195}', 2),
-- macOS variants
('business_suite', 'automation_studio', 'mac', 'dmg', 'https://downloads.vrooli.local/business-suite/mac/VrooliAscension.dmg', '1.0.0', NULL, 'sha256-mac-dmg', TRUE, '{"size_mb":190}', 1),
('business_suite', 'automation_studio', 'mac', 'pkg', 'https://downloads.vrooli.local/business-suite/mac/VrooliAscension.pkg', '1.0.0', NULL, 'sha256-mac-pkg', TRUE, '{"size_mb":195}', 2),
-- Linux variants
('business_suite', 'automation_studio', 'linux', 'appimage', 'https://downloads.vrooli.local/business-suite/linux/VrooliAscension.AppImage', '1.0.0', NULL, 'sha256-linux-appimage', TRUE, '{"size_mb":205}', 1),
('business_suite', 'automation_studio', 'linux', 'deb', 'https://downloads.vrooli.local/business-suite/linux/vrooli-ascension_1.0.0_amd64.deb', '1.0.0', NULL, 'sha256-linux-deb', TRUE, '{"size_mb":200}', 2)
ON CONFLICT (bundle_key, app_key, platform, variant_key) DO UPDATE SET
    artifact_url = EXCLUDED.artifact_url,
    release_version = EXCLUDED.release_version,
    release_notes = EXCLUDED.release_notes,
    checksum = EXCLUDED.checksum,
    requires_entitlement = EXCLUDED.requires_entitlement,
    metadata = EXCLUDED.metadata,
    display_order = EXCLUDED.display_order,
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
ON CONFLICT (id) DO NOTHING;

-- NOTE: Site branding is now stored in JSON file (.vrooli/branding.json)
-- and loaded into memory at startup via ConfigStore.
