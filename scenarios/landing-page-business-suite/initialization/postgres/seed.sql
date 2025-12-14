-- Seed Data for Landing Manager

-- Insert default admin user (OT-P0-008: ADMIN-AUTH)
-- Email: admin@localhost
-- Password: changeme123
-- IMPORTANT: Change this password in production!
INSERT INTO admin_users (email, password_hash) VALUES
('admin@localhost', '$2a$10$nhmpbhFPQUZZwEH.qaYHCeiKBWDvr8z5Z7eM4v62MmNwm.0N.5xeG')
ON CONFLICT (email) DO NOTHING;

-- Insert curated Silent Founder OS variants (OT-P0-014 through OT-P0-018)
INSERT INTO variants (slug, name, description, weight, status) VALUES
('control', 'Silent Founder (Control)', 'Build a business quietly with Vrooli Ascension today.', 100, 'active'),
('silent-founder-entrepreneurship-emotional', 'Silent Founder · Entrepreneurship · Emotional', 'Build a business quietly with Vrooli Ascension today.', 25, 'active'),
('solo-dev-testing-technical', 'Solo Dev · Testing · Technical', 'Replayable e2e tests for solo developers.', 20, 'active'),
('qa-engineer-testing-technical', 'QA Engineer · Testing · Technical', 'Replay evidence and stable regressions.', 15, 'active'),
('automation-engineer-automation-visionary', 'Automation Engineer · Automation · Visionary', 'Visually program the browser; agents on deck.', 20, 'active'),
('agency-marketing-visionary', 'Agency · Marketing · Visionary', 'Deliver client-ready demos and replays in hours.', 20, 'active')
ON CONFLICT (slug) DO UPDATE SET
    name = EXCLUDED.name,
    description = EXCLUDED.description,
    weight = EXCLUDED.weight,
    status = EXCLUDED.status,
    updated_at = NOW();

-- Axis mappings for curated variants (including control)
WITH v AS (SELECT slug, id FROM variants WHERE slug IN (
    'control',
    'silent-founder-entrepreneurship-emotional',
    'solo-dev-testing-technical',
    'qa-engineer-testing-technical',
    'automation-engineer-automation-visionary',
    'agency-marketing-visionary'
))
INSERT INTO variant_axes (variant_id, axis_id, variant_value)
SELECT v.id, axis_map.axis_id, axis_map.variant_value
FROM v
JOIN (
    VALUES
        ('control', 'persona', 'silentFounder'),
        ('control', 'jtbd', 'entrepreneurship'),
        ('control', 'conversionStyle', 'emotional'),
        ('silent-founder-entrepreneurship-emotional', 'persona', 'silentFounder'),
        ('silent-founder-entrepreneurship-emotional', 'jtbd', 'entrepreneurship'),
        ('silent-founder-entrepreneurship-emotional', 'conversionStyle', 'emotional'),
        ('solo-dev-testing-technical', 'persona', 'soloDev'),
        ('solo-dev-testing-technical', 'jtbd', 'testing'),
        ('solo-dev-testing-technical', 'conversionStyle', 'technical'),
        ('qa-engineer-testing-technical', 'persona', 'qaEngineer'),
        ('qa-engineer-testing-technical', 'jtbd', 'testing'),
        ('qa-engineer-testing-technical', 'conversionStyle', 'technical'),
        ('automation-engineer-automation-visionary', 'persona', 'automationEngineer'),
        ('automation-engineer-automation-visionary', 'jtbd', 'automation'),
        ('automation-engineer-automation-visionary', 'conversionStyle', 'visionary'),
        ('agency-marketing-visionary', 'persona', 'agency'),
        ('agency-marketing-visionary', 'jtbd', 'marketing'),
        ('agency-marketing-visionary', 'conversionStyle', 'visionary')
) AS axis_map(slug, axis_id, variant_value) ON v.slug = axis_map.slug
ON CONFLICT (variant_id, axis_id) DO UPDATE
SET variant_value = EXCLUDED.variant_value,
    updated_at = NOW();

-- Shared sections content helpers
-- Silent Founder OS pricing copy reused across variants
DO $$
DECLARE
    pricing_json TEXT := '{
      "title": "Simple, transparent pricing",
      "subtitle": "Vrooli Ascension today. More Silent Founder OS tools added over time.",
      "tiers": [
        {"name":"Free","price":"$0","description":"50 runs/month, builder, replay viewer (watermarked MP4)","features":["Visual workflow builder","Replay viewer with watermark","50 runs/month","No agents or UX metrics"],"cta_text":"Start free","cta_url":"/checkout?plan=free","badge":"Try it now"},
        {"name":"Solo","price":"$29","description":"200 runs/month, MP4 export with watermark","features":["200 runs/month","MP4 export (watermark)","Workflow builder + replays","Email support"],"cta_text":"Upgrade to Solo","cta_url":"/checkout?plan=solo"},
        {"name":"Pro","price":"$79","description":"Unlimited runs, MP4 without watermark, CI hooks","features":["Unlimited runs (fair use)","MP4 exports without watermark","CI integrations + advanced workflow tooling","Early UX metrics access","Limited agent loops"],"cta_text":"Default for Silent Founders","cta_url":"/checkout?plan=pro","highlighted":true,"badge":"Recommended"},
        {"name":"Studio","price":"$199","description":"Agency-ready replays + branding, more agent loops","features":["Multi-seat studio","Custom branding in replays","More agent loop concurrency","Priority support"],"cta_text":"Choose Studio","cta_url":"/checkout?plan=studio"},
        {"name":"Business","price":"$499","description":"For small teams with heavy automation + API needs","features":["Unlimited agent loops","API + webhooks","Reliability & SSO mode prep","Best for teams/clients"],"cta_text":"Talk async","cta_url":"/contact"}
      ]
    }';
    features_json TEXT := '{
      "title": "Vrooli Ascension is live now. The suite keeps growing.",
      "subtitle": "Automate browsers, ship tests, and generate proof-ready replays today. Your subscription includes future Vrooli Business Suite apps at no extra cost.",
      "features": [
        {"title":"Visual workflow builder","description":"Point-and-record or assemble actions to automate admin panels, dashboards, and back-office flows.","icon":"zap"},
        {"title":"Workflows = e2e tests","description":"Run flows in CI, capture failures, and share evidence without rewriting tests.","icon":"shield"},
        {"title":"Replay-as-movie exports","description":"Fake browser frames, smooth cursor animation, zoom/pan highlights, MP4 export with watermark rules by plan.","icon":"sparkles"},
        {"title":"Future UX metrics layer","description":"Coming soon: friction scores, duration, and spatial patterns generated from your workflows.","icon":"layers"},
        {"title":"AI automation (coming soon)","description":"Coming soon: AI that learns your habits from the timeline and builds workflows automatically. Plus agent loops via ecosystem-manager + PRD control tower.","icon":"target"}
      ]
    }';
    faq_json TEXT := '{
      "title":"Answers for quiet founders",
      "subtitle":"What ships today, what is coming, and how we price it.",
      "faqs":[
        {"question":"What do I get today?","answer":"Vrooli Ascension: visual workflow builder, CI-friendly tests, replay viewer, and MP4 exports (watermark rules by plan)."},
        {"question":"What is coming next?","answer":"UX metrics layer (friction, duration, spatial paths) and agent loops that fix flows via ecosystem-manager + PRD control tower."},
        {"question":"Do I have to talk to sales?","answer":"No. No sales calls, no per-seat pricing. Subscribe, download, and grow quietly. Support is async."},
        {"question":"Can I cancel or switch?","answer":"Yes. Plans are flat, cancellable, and your price is honored as the suite expands."}
      ]
    }';
    downloads_json TEXT := '{"title":"Download Vrooli Ascension","subtitle":"macOS, Windows, Linux builds respect entitlements. Install, sign in, and start automating."}';
BEGIN
    -- Reset sections for control and curated variants so seeds overwrite stale content
    DELETE FROM content_sections WHERE variant_id IN (
        SELECT id FROM variants WHERE slug IN (
            'control',
            'silent-founder-entrepreneurship-emotional',
            'solo-dev-testing-technical',
            'qa-engineer-testing-technical',
            'automation-engineer-automation-visionary',
            'agency-marketing-visionary'
        )
    );

    -- Control variant = Silent Founder OS
    INSERT INTO content_sections (variant_id, section_type, content, "order", enabled) VALUES
    ((SELECT id FROM variants WHERE slug='control'), 'hero', '{"title":"Record once. Automate forever","subtitle":"And turn every recording into a polished, professional demo video","cta_text":"Start free","cta_url":"/checkout?plan=pro","secondary_cta_text":"Watch video","secondary_cta_url":"#video-2"}', 1, TRUE),
    ((SELECT id FROM variants WHERE slug='control'), 'video', '{"title":"Watch Vrooli Ascension build and replay a flow","videoUrl":"https://www.youtube.com/watch?v=ysz5S6PUM-U","thumbnailUrl":"/assets/fallback/video-thumb.png","caption":"Visual workflow builder → e2e test → replay-as-movie export. Available today; Silent Founder OS keeps adding tools."}', 2, TRUE),
    ((SELECT id FROM variants WHERE slug='control'), 'features', features_json::json, 3, TRUE),
    ((SELECT id FROM variants WHERE slug='control'), 'pricing', pricing_json::json, 4, TRUE),
    ((SELECT id FROM variants WHERE slug='control'), 'faq', faq_json::json, 5, TRUE),
    ((SELECT id FROM variants WHERE slug='control'), 'cta', '{"title":"See Vrooli Ascension in action","subtitle":"Start free, export a replay, and know more tools are coming to the same subscription.","cta_text":"Get started quietly","cta_url":"/checkout?plan=pro"}', 6, TRUE),
    ((SELECT id FROM variants WHERE slug='control'), 'downloads', downloads_json::json, 7, TRUE)
    ON CONFLICT DO NOTHING;

    -- Silent Founder (entrepreneurship · emotional)
    INSERT INTO content_sections (variant_id, section_type, content, "order", enabled) VALUES
    ((SELECT id FROM variants WHERE slug='silent-founder-entrepreneurship-emotional'), 'hero', '{"title":"Record once. Automate forever","subtitle":"And turn every recording into a polished, professional demo video","cta_text":"Start free","cta_url":"/checkout?plan=pro","secondary_cta_text":"Watch video","secondary_cta_url":"#video-2"}', 1, TRUE),
    ((SELECT id FROM variants WHERE slug='silent-founder-entrepreneurship-emotional'), 'video', '{"title":"Watch Vrooli Ascension build and replay a flow","videoUrl":"https://www.youtube.com/watch?v=ysz5S6PUM-U","thumbnailUrl":"/assets/fallback/video-thumb.png","caption":"Visual workflow builder → e2e test → replay-as-movie export. Available today; Silent Founder OS keeps adding tools."}', 2, TRUE),
    ((SELECT id FROM variants WHERE slug='silent-founder-entrepreneurship-emotional'), 'features', features_json::json, 3, TRUE),
    ((SELECT id FROM variants WHERE slug='silent-founder-entrepreneurship-emotional'), 'pricing', pricing_json::json, 4, TRUE),
    ((SELECT id FROM variants WHERE slug='silent-founder-entrepreneurship-emotional'), 'faq', faq_json::json, 5, TRUE),
    ((SELECT id FROM variants WHERE slug='silent-founder-entrepreneurship-emotional'), 'cta', '{"title":"See Vrooli Ascension in action","subtitle":"Start free, export a replay, and know more tools are coming to the same subscription.","cta_text":"Get started quietly","cta_url":"/checkout?plan=pro"}', 6, TRUE),
    ((SELECT id FROM variants WHERE slug='silent-founder-entrepreneurship-emotional'), 'downloads', downloads_json::json, 7, TRUE)
    ON CONFLICT DO NOTHING;

    -- Solo Dev (testing · technical)
    INSERT INTO content_sections (variant_id, section_type, content, "order", enabled) VALUES
    ((SELECT id FROM variants WHERE slug='solo-dev-testing-technical'), 'hero', '{"title":"End-to-end tests that replay like a movie.","subtitle":"Your workflows are your tests. Build in Vrooli Ascension, run in CI, and share MP4 evidence without watermark on Pro.","cta_text":"Build a test","cta_url":"/checkout?plan=pro","secondary_cta_text":"Watch a failure replay","secondary_cta_url":"#video-2"}', 1, TRUE),
    ((SELECT id FROM variants WHERE slug='solo-dev-testing-technical'), 'video', '{"title":"See a CI-friendly replay","videoUrl":"https://www.youtube.com/watch?v=ysz5S6PUM-U","thumbnailUrl":"/assets/fallback/video-thumb.png","caption":"Trigger workflows from CI, capture failures, export the replay for teammates and docs."}', 2, TRUE),
    ((SELECT id FROM variants WHERE slug='solo-dev-testing-technical'), 'features', features_json::json, 3, TRUE),
    ((SELECT id FROM variants WHERE slug='solo-dev-testing-technical'), 'pricing', pricing_json::json, 4, TRUE),
    ((SELECT id FROM variants WHERE slug='solo-dev-testing-technical'), 'faq', faq_json::json, 5, TRUE),
    ((SELECT id FROM variants WHERE slug='solo-dev-testing-technical'), 'cta', '{"title":"Ship tests and demos from one workflow","subtitle":"Upgrade to remove watermarks, unlock unlimited runs, and stay ready for the next Silent Founder OS tools.","cta_text":"Choose Pro","cta_url":"/checkout?plan=pro"}', 6, TRUE),
    ((SELECT id FROM variants WHERE slug='solo-dev-testing-technical'), 'downloads', downloads_json::json, 7, TRUE)
    ON CONFLICT DO NOTHING;

    -- QA Engineer (testing · technical)
    INSERT INTO content_sections (variant_id, section_type, content, "order", enabled) VALUES
    ((SELECT id FROM variants WHERE slug='qa-engineer-testing-technical'), 'hero', '{"title":"Give QA replay evidence for every regression.","subtitle":"Vrooli Ascension records flows as living tests. Failures replay in a fake browser frame so devs and PMs see exactly what broke.","cta_text":"Start QA-ready","cta_url":"/checkout?plan=pro","secondary_cta_text":"Preview replay evidence","secondary_cta_url":"#video-2"}', 1, TRUE),
    ((SELECT id FROM variants WHERE slug='qa-engineer-testing-technical'), 'video', '{"title":"Debug with visual proof","videoUrl":"https://www.youtube.com/watch?v=ysz5S6PUM-U","thumbnailUrl":"/assets/fallback/video-thumb.png","caption":"Replay cursor paths, zooms, and highlights for any failing step. Export as MP4 for tickets."}', 2, TRUE),
    ((SELECT id FROM variants WHERE slug='qa-engineer-testing-technical'), 'features', features_json::json, 3, TRUE),
    ((SELECT id FROM variants WHERE slug='qa-engineer-testing-technical'), 'pricing', pricing_json::json, 4, TRUE),
    ((SELECT id FROM variants WHERE slug='qa-engineer-testing-technical'), 'faq', faq_json::json, 5, TRUE),
    ((SELECT id FROM variants WHERE slug='qa-engineer-testing-technical'), 'cta', '{"title":"Stop chasing repro steps","subtitle":"Upgrade for watermark-free MP4 exports, CI hooks, and early UX metrics.","cta_text":"Choose Pro","cta_url":"/checkout?plan=pro"}', 6, TRUE),
    ((SELECT id FROM variants WHERE slug='qa-engineer-testing-technical'), 'downloads', downloads_json::json, 7, TRUE)
    ON CONFLICT DO NOTHING;

    -- Automation Engineer (automation · visionary)
    INSERT INTO content_sections (variant_id, section_type, content, "order", enabled) VALUES
    ((SELECT id FROM variants WHERE slug='automation-engineer-automation-visionary'), 'hero', '{"title":"Visually program the browser. Agents take it from there.","subtitle":"Replace brittle scripts with a visual builder, schedule runs, and get ready for agent loops that fix issues automatically.","cta_text":"Automate a flow","cta_url":"/checkout?plan=pro","secondary_cta_text":"See the builder","secondary_cta_url":"#video-2"}', 1, TRUE),
    ((SELECT id FROM variants WHERE slug='automation-engineer-automation-visionary'), 'video', '{"title":"From workflow to automation loop","videoUrl":"https://www.youtube.com/watch?v=ysz5S6PUM-U","thumbnailUrl":"/assets/fallback/video-thumb.png","caption":"Capture a workflow, replay it, and visualize where agent loops will plug in next."}', 2, TRUE),
    ((SELECT id FROM variants WHERE slug='automation-engineer-automation-visionary'), 'features', features_json::json, 3, TRUE),
    ((SELECT id FROM variants WHERE slug='automation-engineer-automation-visionary'), 'pricing', pricing_json::json, 4, TRUE),
    ((SELECT id FROM variants WHERE slug='automation-engineer-automation-visionary'), 'faq', faq_json::json, 5, TRUE),
    ((SELECT id FROM variants WHERE slug='automation-engineer-automation-visionary'), 'cta', '{"title":"Ship back-office automations with proof","subtitle":"Stay on one subscription while Silent Founder OS adds agent loops and metrics.","cta_text":"Start with Pro","cta_url":"/checkout?plan=pro"}', 6, TRUE),
    ((SELECT id FROM variants WHERE slug='automation-engineer-automation-visionary'), 'downloads', downloads_json::json, 7, TRUE)
    ON CONFLICT DO NOTHING;

    -- Agency (marketing · visionary)
    INSERT INTO content_sections (variant_id, section_type, content, "order", enabled) VALUES
    ((SELECT id FROM variants WHERE slug='agency-marketing-visionary'), 'hero', '{"title":"Deliver client-ready demos in hours.","subtitle":"Record flows, export MP4s with fake browser frames, and ship branded replays without custom video work.","cta_text":"Start Studio","cta_url":"/checkout?plan=studio","secondary_cta_text":"See a client-ready replay","secondary_cta_url":"#video-2"}', 1, TRUE),
    ((SELECT id FROM variants WHERE slug='agency-marketing-visionary'), 'video', '{"title":"Cinematic browser replays","videoUrl":"https://www.youtube.com/watch?v=ysz5S6PUM-U","thumbnailUrl":"/assets/fallback/video-thumb.png","caption":"Cursor animation, zooms, and highlights packaged as MP4 for marketing and tutorials."}', 2, TRUE),
    ((SELECT id FROM variants WHERE slug='agency-marketing-visionary'), 'features', features_json::json, 3, TRUE),
    ((SELECT id FROM variants WHERE slug='agency-marketing-visionary'), 'pricing', pricing_json::json, 4, TRUE),
    ((SELECT id FROM variants WHERE slug='agency-marketing-visionary'), 'faq', faq_json::json, 5, TRUE),
    ((SELECT id FROM variants WHERE slug='agency-marketing-visionary'), 'cta', '{"title":"Make replays your deliverable","subtitle":"Studio and Business plans unlock branding, agent loops, and API/webhooks for clients.","cta_text":"Choose Studio","cta_url":"/checkout?plan=studio"}', 6, TRUE),
    ((SELECT id FROM variants WHERE slug='agency-marketing-visionary'), 'downloads', downloads_json::json, 7, TRUE)
    ON CONFLICT DO NOTHING;
END $$;

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
INSERT INTO download_apps (bundle_key, app_key, name, tagline, description, install_overview, install_steps, storefronts, metadata, display_order)
VALUES
('business_suite', 'automation_studio', 'Vrooli Ascension', 'Silent Founder OS · Day-one value', 'Desktop suite for visual browser automation, tests, and cinematic replays.', 'Download the installer for your OS and sign in with the email tied to your active subscription.', '["Download the installer for your OS","Launch the setup wizard","Sign in with your subscription email to unlock downloads"]', '[{"store":"app_store","label":"macOS App Store","url":"https://apps.apple.com/app/id000000","badge":"Download on the App Store"}]', '{"bundle":"business_suite"}', 1)
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
('business_suite', 'automation_studio', 'windows', 'https://downloads.vrooli.local/business-suite/win/VrooliAscensionSetup.exe', '1.2.0', 'Includes fresh launch console + webdriver upgrades.', 'sha256-win-placeholder', TRUE, '{"size_mb":230}'),
('business_suite', 'automation_studio', 'mac', 'https://downloads.vrooli.local/business-suite/mac/VrooliAscension.dmg', '1.2.0', 'Signed universal build for Apple Silicon + Intel.', 'sha256-mac-placeholder', TRUE, '{"size_mb":215}'),
('business_suite', 'automation_studio', 'linux', 'https://downloads.vrooli.local/business-suite/linux/vrooli-ascension.tar.gz', '1.2.0', 'AppImage bundle tested on Ubuntu/Debian.', 'sha256-linux-placeholder', TRUE, '{"size_mb":225}')
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
ON CONFLICT (id) DO NOTHING;

-- Seed default site branding (singleton row)
INSERT INTO site_branding (id, site_name, tagline, robots_txt)
VALUES (
    1,
    'Silent Founder OS',
    'Build and market your business quietly with Vrooli Ascension.',
    E'User-agent: *\nAllow: /'
)
ON CONFLICT (id) DO UPDATE SET
    site_name = EXCLUDED.site_name,
    tagline = EXCLUDED.tagline,
    robots_txt = EXCLUDED.robots_txt,
    updated_at = NOW();
