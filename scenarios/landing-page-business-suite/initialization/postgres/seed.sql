-- Seed Data for Landing Manager
--
-- NOTE: Variant, section, and branding configuration is now stored in JSON files
-- (config/variants/*.json and config/branding.json) and loaded into memory at startup.
-- This seed file only seeds runtime/dynamic data.

-- Insert default admin user (OT-P0-008: ADMIN-AUTH)
-- Email: admin@localhost
-- Password: changeme123
-- IMPORTANT: Change this password in production!
INSERT INTO admin_users (email, password_hash) VALUES
('admin@localhost', '$2a$10$nhmpbhFPQUZZwEH.qaYHCeiKBWDvr8z5Z7eM4v62MmNwm.0N.5xeG')
ON CONFLICT (email) DO NOTHING;

-- Bundle products configuration (defines bundle structure; prices come from Stripe import)
-- NOTE: Prices are no longer seeded here. Use the admin UI to import plans from Stripe.
-- Plans are stored in .vrooli/plans.json after import.
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

-- NOTE: Site branding is now stored in tracked JSON file (config/branding.json)
-- and loaded into memory at startup via ConfigStore.
