-- Landing Manager Database Schema

-- Admin Users Table (OT-P0-008: ADMIN-AUTH)
-- Stores admin credentials with bcrypt-hashed passwords
CREATE TABLE IF NOT EXISTS admin_users (
    id SERIAL PRIMARY KEY,
    email VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    last_login TIMESTAMP
);

CREATE INDEX idx_admin_users_email ON admin_users(email);

-- A/B Testing Variants Table (OT-P0-014 through OT-P0-018)
-- Stores landing page variants with weights for A/B testing
CREATE TABLE IF NOT EXISTS variants (
    id SERIAL PRIMARY KEY,
    slug VARCHAR(100) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    weight INTEGER DEFAULT 50 CHECK (weight >= 0 AND weight <= 100),
    status VARCHAR(20) DEFAULT 'active' CHECK (status IN ('active', 'archived', 'deleted')),
    header_config JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    archived_at TIMESTAMP
);

CREATE INDEX idx_variants_slug ON variants(slug);
CREATE INDEX idx_variants_status ON variants(status);

-- Variant Axes Table - stores axis selections per variant for persona/JTBD/conversion style targeting
CREATE TABLE IF NOT EXISTS variant_axes (
    variant_id INTEGER REFERENCES variants(id) ON DELETE CASCADE,
    axis_id VARCHAR(100) NOT NULL,
    variant_value VARCHAR(100) NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    PRIMARY KEY (variant_id, axis_id)
);

CREATE INDEX idx_variant_axes_axis ON variant_axes(axis_id);

-- Metrics Events Table (OT-P0-019 through OT-P0-022)
-- Stores analytics events tagged with variant_id
CREATE TABLE IF NOT EXISTS metrics_events (
    id SERIAL PRIMARY KEY,
    variant_id INTEGER REFERENCES variants(id),
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('page_view', 'scroll_depth', 'click', 'form_submit', 'conversion', 'download')),
    event_data JSONB,
    session_id VARCHAR(255),
    visitor_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_metrics_events_variant ON metrics_events(variant_id);
CREATE INDEX idx_metrics_events_type ON metrics_events(event_type);
CREATE INDEX idx_metrics_events_created ON metrics_events(created_at);
CREATE INDEX idx_metrics_events_session ON metrics_events(session_id);

-- Checkout Sessions Table (OT-P0-025, OT-P0-026: STRIPE-CONFIG, STRIPE-ROUTES)
-- Stores Stripe checkout session metadata
CREATE TABLE IF NOT EXISTS checkout_sessions (
    id SERIAL PRIMARY KEY,
    session_id VARCHAR(255) UNIQUE NOT NULL,
    customer_email VARCHAR(255),
    price_id VARCHAR(255),
    subscription_id VARCHAR(255),
    status VARCHAR(50) NOT NULL,
    session_type VARCHAR(50) NOT NULL DEFAULT 'subscription',
    amount_cents INTEGER,
    schedule_id VARCHAR(255),
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_checkout_sessions_session_id ON checkout_sessions(session_id);
CREATE INDEX idx_checkout_sessions_status ON checkout_sessions(status);
CREATE INDEX idx_checkout_sessions_type ON checkout_sessions(session_type);

-- Subscriptions Table (OT-P0-028, OT-P0-029, OT-P0-030: SUB-VERIFY, SUB-CACHE, SUB-CANCEL)
-- Stores Stripe subscription status for verification and caching
CREATE TABLE IF NOT EXISTS subscriptions (
    id SERIAL PRIMARY KEY,
    subscription_id VARCHAR(255) UNIQUE NOT NULL,
    customer_id VARCHAR(255),
    customer_email VARCHAR(255),
    status VARCHAR(50) NOT NULL CHECK (status IN ('active', 'trialing', 'past_due', 'canceled', 'unpaid')),
    canceled_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_subscriptions_subscription_id ON subscriptions(subscription_id);
CREATE INDEX idx_subscriptions_customer_email ON subscriptions(customer_email);
CREATE INDEX idx_subscriptions_customer_id ON subscriptions(customer_id);
CREATE INDEX idx_subscriptions_status ON subscriptions(status);
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS plan_tier VARCHAR(50);
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS price_id VARCHAR(255);
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS bundle_key VARCHAR(100);

CREATE TABLE IF NOT EXISTS subscription_schedules (
    id SERIAL PRIMARY KEY,
    schedule_id VARCHAR(255) UNIQUE NOT NULL,
    subscription_id VARCHAR(255),
    price_id VARCHAR(255) NOT NULL,
    billing_interval VARCHAR(20) NOT NULL CHECK (billing_interval IN ('month','year','one_time')),
    intro_enabled BOOLEAN DEFAULT FALSE,
    intro_amount_cents INTEGER,
    intro_periods INTEGER DEFAULT 0,
    normal_amount_cents INTEGER,
    next_billing_at TIMESTAMP,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_subscription_schedules_schedule_id ON subscription_schedules(schedule_id);
CREATE INDEX idx_subscription_schedules_subscription_id ON subscription_schedules(subscription_id);

-- Content Sections Table (OT-P0-012, OT-P0-013: CUSTOM-SPLIT, CUSTOM-LIVE)
-- Stores customizable landing page sections for live preview editing
CREATE TABLE IF NOT EXISTS content_sections (
    id SERIAL PRIMARY KEY,
    variant_id INTEGER REFERENCES variants(id) ON DELETE CASCADE,
    section_type VARCHAR(50) NOT NULL CHECK (section_type IN ('hero', 'features', 'pricing', 'cta', 'testimonials', 'faq', 'footer', 'video', 'downloads')),
    content JSONB NOT NULL,
    "order" INTEGER DEFAULT 0,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_content_sections_variant ON content_sections(variant_id);
CREATE INDEX idx_content_sections_type ON content_sections(section_type);
CREATE INDEX idx_content_sections_order ON content_sections("order");

-- Bundle products (Stripe metadata)
CREATE TABLE IF NOT EXISTS bundle_products (
    id SERIAL PRIMARY KEY,
    bundle_key VARCHAR(100) UNIQUE NOT NULL,
    bundle_name VARCHAR(255) NOT NULL,
    stripe_product_id VARCHAR(255) UNIQUE NOT NULL,
    credits_per_usd BIGINT NOT NULL,
    display_credits_multiplier NUMERIC(12,6) DEFAULT 1.0,
    display_credits_label VARCHAR(50) DEFAULT 'credits',
    environment VARCHAR(50) DEFAULT 'production',
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_bundle_products_env ON bundle_products(environment);

-- Bundle prices (Stripe price metadata)
CREATE TABLE IF NOT EXISTS bundle_prices (
    id SERIAL PRIMARY KEY,
    product_id INTEGER REFERENCES bundle_products(id) ON DELETE CASCADE,
    stripe_price_id VARCHAR(255) UNIQUE NOT NULL,
    plan_name VARCHAR(100) NOT NULL,
    plan_tier VARCHAR(50) NOT NULL CHECK (plan_tier IN ('free','solo','pro','studio','business','credits','donation')),
    billing_interval VARCHAR(20) NOT NULL CHECK (billing_interval IN ('month','year','one_time')),
    amount_cents INTEGER NOT NULL,
    currency VARCHAR(10) DEFAULT 'usd',
    intro_enabled BOOLEAN DEFAULT FALSE,
    intro_type VARCHAR(50),
    intro_amount_cents INTEGER,
    intro_periods INTEGER DEFAULT 0,
    intro_price_lookup_key VARCHAR(255),
    monthly_included_credits INTEGER DEFAULT 0,
    one_time_bonus_credits INTEGER DEFAULT 0,
    plan_rank INTEGER DEFAULT 0,
    bonus_type VARCHAR(50),
    kind VARCHAR(50) DEFAULT 'subscription',
    is_variable_amount BOOLEAN DEFAULT FALSE,
    display_enabled BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}'::jsonb,
    display_weight INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_bundle_prices_tier ON bundle_prices(plan_tier);
CREATE INDEX idx_bundle_prices_interval ON bundle_prices(billing_interval);

CREATE TABLE IF NOT EXISTS download_apps (
    id SERIAL PRIMARY KEY,
    bundle_key VARCHAR(100) NOT NULL,
    app_key VARCHAR(100) NOT NULL,
    name VARCHAR(255) NOT NULL,
    tagline TEXT,
    description TEXT,
    icon_url TEXT,
    screenshot_url TEXT,
    install_overview TEXT,
    install_steps JSONB DEFAULT '[]'::jsonb,
    storefronts JSONB DEFAULT '[]'::jsonb,
    metadata JSONB DEFAULT '{}'::jsonb,
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE (bundle_key, app_key)
);

CREATE TABLE IF NOT EXISTS download_assets (
    id SERIAL PRIMARY KEY,
    bundle_key VARCHAR(100) NOT NULL,
    app_key VARCHAR(100) NOT NULL,
    platform VARCHAR(50) NOT NULL CHECK (platform IN ('windows','mac','linux')),
    variant_key VARCHAR(50) NOT NULL DEFAULT 'default',
    artifact_url TEXT NOT NULL,
    release_version VARCHAR(50) NOT NULL,
    release_notes TEXT,
    checksum VARCHAR(255),
    requires_entitlement BOOLEAN DEFAULT TRUE,
    metadata JSONB DEFAULT '{}'::jsonb,
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    CONSTRAINT fk_download_app FOREIGN KEY (bundle_key, app_key)
        REFERENCES download_apps(bundle_key, app_key) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_download_apps_bundle ON download_apps(bundle_key);

-- Payment + Stripe configuration (admin-managed)
CREATE TABLE IF NOT EXISTS payment_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    publishable_key TEXT,
    secret_key TEXT,
    webhook_secret TEXT,
    dashboard_url TEXT,
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_download_assets_bundle_app_platform_variant ON download_assets(bundle_key, app_key, platform, variant_key);

-- Credit wallets and transactions
CREATE TABLE IF NOT EXISTS credit_wallets (
    id SERIAL PRIMARY KEY,
    customer_email VARCHAR(255) UNIQUE NOT NULL,
    balance_credits BIGINT DEFAULT 0,
    bonus_credits BIGINT DEFAULT 0,
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS credit_transactions (
    id SERIAL PRIMARY KEY,
    customer_email VARCHAR(255) NOT NULL,
    amount_credits BIGINT NOT NULL,
    transaction_type VARCHAR(50) NOT NULL,
    source VARCHAR(100),
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_credit_transactions_customer ON credit_transactions(customer_email);

-- Site Branding Table (singleton pattern for site-wide branding)
-- Stores logos, favicons, default SEO, theme colors, and technical settings
CREATE TABLE IF NOT EXISTS site_branding (
    id INTEGER PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    site_name TEXT NOT NULL DEFAULT 'My Landing',
    tagline TEXT,

    -- Logo assets
    logo_url TEXT,
    logo_icon_url TEXT,
    favicon_url TEXT,
    apple_touch_icon_url TEXT,

    -- Default SEO (fallback when variant doesn't specify)
    default_title TEXT,
    default_description TEXT,
    default_og_image_url TEXT,

    -- Theme overrides (extends styling.json)
    theme_primary_color TEXT,
    theme_background_color TEXT,

    -- Technical settings
    canonical_base_url TEXT,
    google_site_verification TEXT,
    robots_txt TEXT,

    -- Support settings
    support_chat_url TEXT,

    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

-- Ensure only one row exists (singleton)
CREATE UNIQUE INDEX IF NOT EXISTS site_branding_singleton ON site_branding ((1));

-- Uploaded Assets Table
-- Stores metadata for uploaded files (logos, favicons, og images, etc.)
CREATE TABLE IF NOT EXISTS assets (
    id SERIAL PRIMARY KEY,
    filename TEXT NOT NULL,
    original_filename TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    size_bytes BIGINT NOT NULL,
    storage_path TEXT NOT NULL,
    thumbnail_path TEXT,
    alt_text TEXT,
    category TEXT DEFAULT 'general' CHECK (category IN ('logo', 'favicon', 'og_image', 'general')),
    uploaded_by TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_assets_category ON assets(category);
CREATE INDEX idx_assets_created ON assets(created_at);

-- Add SEO config column to variants table for per-variant SEO
ALTER TABLE variants ADD COLUMN IF NOT EXISTS seo_config JSONB DEFAULT '{}'::jsonb;

-- Add support chat URL to site_branding for existing databases
ALTER TABLE site_branding ADD COLUMN IF NOT EXISTS support_chat_url TEXT;

-- Add support email to site_branding for existing databases
ALTER TABLE site_branding ADD COLUMN IF NOT EXISTS support_email TEXT;

-- SMTP configuration for email notifications
ALTER TABLE site_branding ADD COLUMN IF NOT EXISTS smtp_host TEXT;
ALTER TABLE site_branding ADD COLUMN IF NOT EXISTS smtp_port INTEGER DEFAULT 587;
ALTER TABLE site_branding ADD COLUMN IF NOT EXISTS smtp_username TEXT;
ALTER TABLE site_branding ADD COLUMN IF NOT EXISTS smtp_password TEXT;
ALTER TABLE site_branding ADD COLUMN IF NOT EXISTS smtp_from TEXT;

-- Feedback Requests Table
-- Stores user feedback, bug reports, feature requests, and refund requests
CREATE TABLE IF NOT EXISTS feedback_requests (
    id SERIAL PRIMARY KEY,
    type VARCHAR(50) NOT NULL CHECK (type IN ('refund', 'bug', 'feature', 'general')),
    email VARCHAR(255) NOT NULL,
    subject VARCHAR(500) NOT NULL,
    message TEXT NOT NULL,
    order_id VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'in_progress', 'resolved', 'rejected')),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_feedback_requests_type ON feedback_requests(type);
CREATE INDEX idx_feedback_requests_status ON feedback_requests(status);
CREATE INDEX idx_feedback_requests_email ON feedback_requests(email);
CREATE INDEX idx_feedback_requests_created ON feedback_requests(created_at);
