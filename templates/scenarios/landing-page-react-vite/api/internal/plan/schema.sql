-- Plan domain schema.
--
-- Bundle products and their Stripe price rows, which back the public pricing
-- page and the admin bundle catalog. bundle_prices foreign-keys its product;
-- both tables live here so the FK is satisfied within one schema application.
-- Forward-only declarative DDL.
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
CREATE INDEX IF NOT EXISTS idx_bundle_products_env ON bundle_products(environment);

CREATE TABLE IF NOT EXISTS bundle_prices (
    id SERIAL PRIMARY KEY,
    product_id INTEGER REFERENCES bundle_products(id) ON DELETE CASCADE,
    stripe_price_id VARCHAR(255) UNIQUE NOT NULL,
    plan_name VARCHAR(100) NOT NULL,
    plan_tier VARCHAR(50) NOT NULL CHECK (plan_tier IN ('solo','pro','studio','business','credits','donation')),
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
CREATE INDEX IF NOT EXISTS idx_bundle_prices_tier ON bundle_prices(plan_tier);
CREATE INDEX IF NOT EXISTS idx_bundle_prices_interval ON bundle_prices(billing_interval);
