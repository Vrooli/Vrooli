-- Landing Manager Database Schema
--
-- NOTE: Variant, section, and branding configuration is now stored in JSON files
-- (.vrooli/variants/*.json and .vrooli/branding.json) and loaded into memory at startup.
-- This schema only contains tables for runtime/dynamic data.

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

-- Metrics Events Table (OT-P0-019 through OT-P0-022)
-- Stores analytics events (variant_slug is stored as text, not FK)
CREATE TABLE IF NOT EXISTS metrics_events (
    id SERIAL PRIMARY KEY,
    variant_slug VARCHAR(100),
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('page_view', 'scroll_depth', 'click', 'form_submit', 'conversion', 'download')),
    event_data JSONB,
    session_id VARCHAR(255),
    visitor_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_metrics_events_variant ON metrics_events(variant_slug);
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

-- NOTE: Content sections are now stored in JSON files (.vrooli/variants/*.json)
-- and loaded into memory at startup via ConfigStore.

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

-- NOTE: Site branding is now stored in JSON file (.vrooli/branding.json)
-- and loaded into memory at startup via ConfigStore.

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

-- Subscription Tier Limits Table
-- Defines credit limits per subscription tier (cost-based or app-specific)
CREATE TABLE IF NOT EXISTS subscription_tier_limits (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tier_id VARCHAR(50) NOT NULL,           -- 'free', 'solo', 'pro', 'studio', 'business'
    limit_type VARCHAR(20) NOT NULL,        -- 'cost_based' or 'app_specific'
    limit_key VARCHAR(100) NOT NULL,        -- 'ai_credits', 'workflow_exports', etc.
    limit_value BIGINT NOT NULL,            -- In base units (-1 = unlimited)
    cost_multiplier BIGINT DEFAULT 1000000, -- cents x multiplier for cost_based
    app_bundle_key VARCHAR(100),            -- NULL for cost_based, app key for app_specific
    reset_period VARCHAR(20) DEFAULT 'monthly',
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(tier_id, limit_type, limit_key, app_bundle_key)
);

CREATE INDEX idx_subscription_tier_limits_tier ON subscription_tier_limits(tier_id);
CREATE INDEX idx_subscription_tier_limits_type ON subscription_tier_limits(limit_type);
CREATE INDEX idx_subscription_tier_limits_app ON subscription_tier_limits(app_bundle_key);

-- Usage Records Table
-- Tracks credit usage per user per billing period
CREATE TABLE IF NOT EXISTS usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_identity VARCHAR(255) NOT NULL,
    billing_period VARCHAR(20) NOT NULL,    -- 'YYYY-MM'
    limit_key VARCHAR(100) NOT NULL,
    usage_amount BIGINT NOT NULL DEFAULT 0,
    app_bundle_key VARCHAR(100),
    last_operation_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_identity, billing_period, limit_key, app_bundle_key)
);

CREATE INDEX idx_usage_records_user_period ON usage_records(user_identity, billing_period);
CREATE INDEX idx_usage_records_limit_key ON usage_records(limit_key);
CREATE INDEX idx_usage_records_app ON usage_records(app_bundle_key);

-- API Keys Table
-- Stores encrypted AI provider API keys (admin-managed)
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider VARCHAR(50) NOT NULL UNIQUE,   -- 'openrouter', 'openai', 'anthropic'
    encrypted_key TEXT NOT NULL,
    key_hint VARCHAR(20),                   -- Last 4 chars for display
    is_active BOOLEAN DEFAULT true,
    last_verified_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_api_keys_provider ON api_keys(provider);
CREATE INDEX idx_api_keys_active ON api_keys(is_active);
