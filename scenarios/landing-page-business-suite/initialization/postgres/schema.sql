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

CREATE INDEX IF NOT EXISTS idx_admin_users_email ON admin_users(email);

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

CREATE INDEX IF NOT EXISTS idx_metrics_events_variant ON metrics_events(variant_slug);
CREATE INDEX IF NOT EXISTS idx_metrics_events_type ON metrics_events(event_type);
CREATE INDEX IF NOT EXISTS idx_metrics_events_created ON metrics_events(created_at);
CREATE INDEX IF NOT EXISTS idx_metrics_events_session ON metrics_events(session_id);

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

CREATE INDEX IF NOT EXISTS idx_checkout_sessions_session_id ON checkout_sessions(session_id);
CREATE INDEX IF NOT EXISTS idx_checkout_sessions_status ON checkout_sessions(status);
CREATE INDEX IF NOT EXISTS idx_checkout_sessions_type ON checkout_sessions(session_type);

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

CREATE INDEX IF NOT EXISTS idx_subscriptions_subscription_id ON subscriptions(subscription_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_customer_email ON subscriptions(customer_email);
CREATE INDEX IF NOT EXISTS idx_subscriptions_customer_id ON subscriptions(customer_id);
CREATE INDEX IF NOT EXISTS idx_subscriptions_status ON subscriptions(status);
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS plan_tier VARCHAR(50);
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS price_id VARCHAR(255);
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS bundle_key VARCHAR(100);
ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS billing_cycle_start INTEGER DEFAULT 0;

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

CREATE INDEX IF NOT EXISTS idx_subscription_schedules_schedule_id ON subscription_schedules(schedule_id);
CREATE INDEX IF NOT EXISTS idx_subscription_schedules_subscription_id ON subscription_schedules(subscription_id);

-- NOTE: Content sections are now stored in JSON files (.vrooli/variants/*.json)
-- and loaded into memory at startup via ConfigStore.

-- NOTE: Bundle/pricing plan configuration is now stored in JSON files (.vrooli/plans.json)
-- and loaded into memory at startup via PlanStore. The bundle_products and bundle_prices
-- tables have been removed as part of the migration to file-based configuration.

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

CREATE UNIQUE INDEX IF NOT EXISTS idx_download_assets_bundle_app_platform_variant ON download_assets(bundle_key, app_key, platform, variant_key);

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
    stripe_event_id VARCHAR(255),
    metadata JSONB DEFAULT '{}'::jsonb,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_credit_transactions_customer ON credit_transactions(customer_email);
-- Unique index for idempotency - only index non-null stripe_event_ids
CREATE UNIQUE INDEX IF NOT EXISTS idx_credit_transactions_stripe_event
    ON credit_transactions(stripe_event_id) WHERE stripe_event_id IS NOT NULL;

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

CREATE INDEX IF NOT EXISTS idx_assets_category ON assets(category);
CREATE INDEX IF NOT EXISTS idx_assets_created ON assets(created_at);

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

CREATE INDEX IF NOT EXISTS idx_feedback_requests_type ON feedback_requests(type);
CREATE INDEX IF NOT EXISTS idx_feedback_requests_status ON feedback_requests(status);
CREATE INDEX IF NOT EXISTS idx_feedback_requests_email ON feedback_requests(email);
CREATE INDEX IF NOT EXISTS idx_feedback_requests_created ON feedback_requests(created_at);

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

CREATE INDEX IF NOT EXISTS idx_subscription_tier_limits_tier ON subscription_tier_limits(tier_id);
CREATE INDEX IF NOT EXISTS idx_subscription_tier_limits_type ON subscription_tier_limits(limit_type);
CREATE INDEX IF NOT EXISTS idx_subscription_tier_limits_app ON subscription_tier_limits(app_bundle_key);

-- Usage Records Table
-- Tracks credit usage per user per billing period
CREATE TABLE IF NOT EXISTS usage_records (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_identity VARCHAR(255) NOT NULL,
    billing_period VARCHAR(20) NOT NULL,    -- 'YYYY-MM'
    limit_key VARCHAR(100) NOT NULL,
    usage_amount BIGINT NOT NULL DEFAULT 0,
    app_bundle_key VARCHAR(100),
    operation_id UUID,                      -- Idempotency key for deduplication
    last_operation_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(user_identity, billing_period, limit_key, app_bundle_key)
);

CREATE INDEX IF NOT EXISTS idx_usage_records_user_period ON usage_records(user_identity, billing_period);
CREATE INDEX IF NOT EXISTS idx_usage_records_limit_key ON usage_records(limit_key);
CREATE INDEX IF NOT EXISTS idx_usage_records_app ON usage_records(app_bundle_key);
-- Partial unique index for idempotency - only index non-null operation_ids
CREATE UNIQUE INDEX IF NOT EXISTS idx_usage_records_operation_id ON usage_records(operation_id) WHERE operation_id IS NOT NULL;

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

CREATE INDEX IF NOT EXISTS idx_api_keys_provider ON api_keys(provider);
CREATE INDEX IF NOT EXISTS idx_api_keys_active ON api_keys(is_active);

-- User Authentication Tables (User Auth Implementation)
-- User accounts (linked to Stripe customers)
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email VARCHAR(255) UNIQUE NOT NULL,
    email_verified BOOLEAN DEFAULT FALSE,
    stripe_customer_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    last_login_at TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_users_email ON users(email);
CREATE INDEX IF NOT EXISTS idx_users_stripe_customer ON users(stripe_customer_id);

-- Magic link tokens (short-lived, one-time use)
CREATE TABLE IF NOT EXISTS auth_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(255) NOT NULL,  -- SHA-256 of token
    token_type VARCHAR(50) NOT NULL,   -- 'magic_link', 'refresh'
    expires_at TIMESTAMP NOT NULL,
    used_at TIMESTAMP,                 -- NULL until used
    created_at TIMESTAMP DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT
);

CREATE INDEX IF NOT EXISTS idx_auth_tokens_hash ON auth_tokens(token_hash);
CREATE INDEX IF NOT EXISTS idx_auth_tokens_user_expires ON auth_tokens(user_id, expires_at);

-- Active user sessions (supports "view all sessions" feature)
CREATE TABLE IF NOT EXISTS user_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    refresh_token_hash VARCHAR(255) NOT NULL,
    expires_at TIMESTAMP NOT NULL,
    created_at TIMESTAMP DEFAULT NOW(),
    last_used_at TIMESTAMP DEFAULT NOW(),
    ip_address INET,
    user_agent TEXT,
    device_info JSONB DEFAULT '{}',   -- For future "Chrome on Windows" display
    revoked BOOLEAN DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS idx_user_sessions_user ON user_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_user_sessions_hash ON user_sessions(refresh_token_hash);
CREATE INDEX IF NOT EXISTS idx_user_sessions_active ON user_sessions(user_id, revoked, expires_at);

-- Email Normalization Migration (idempotent)
-- Normalizes existing email data to lowercase/trimmed format
-- This prevents duplicate credit balances or subscription lookup issues
UPDATE users SET email = LOWER(TRIM(email)) WHERE email != LOWER(TRIM(email));
UPDATE subscriptions SET customer_email = LOWER(TRIM(customer_email)) WHERE customer_email IS NOT NULL AND customer_email != LOWER(TRIM(customer_email));
UPDATE credit_wallets SET customer_email = LOWER(TRIM(customer_email)) WHERE customer_email != LOWER(TRIM(customer_email));
UPDATE usage_records SET user_identity = LOWER(TRIM(user_identity)) WHERE user_identity != LOWER(TRIM(user_identity));
UPDATE credit_transactions SET customer_email = LOWER(TRIM(customer_email)) WHERE customer_email != LOWER(TRIM(customer_email));

-- Credit Reservations Table (TOCTOU fix for streaming requests)
-- Tracks pending credit reservations for atomic check-and-charge
CREATE TABLE IF NOT EXISTS credit_reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_identity VARCHAR(255) NOT NULL,
    billing_period VARCHAR(20) NOT NULL,
    limit_key VARCHAR(100) NOT NULL,
    reserved_amount BIGINT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'finalized', 'released', 'expired')),
    created_at TIMESTAMP DEFAULT NOW(),
    finalized_at TIMESTAMP,
    expires_at TIMESTAMP NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_credit_reservations_user ON credit_reservations(user_identity, status);
CREATE INDEX IF NOT EXISTS idx_credit_reservations_expires ON credit_reservations(expires_at) WHERE status = 'pending';
