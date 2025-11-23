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
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    archived_at TIMESTAMP
);

CREATE INDEX idx_variants_slug ON variants(slug);
CREATE INDEX idx_variants_status ON variants(status);

-- Metrics Events Table (OT-P0-019 through OT-P0-022)
-- Stores analytics events tagged with variant_id
CREATE TABLE IF NOT EXISTS metrics_events (
    id SERIAL PRIMARY KEY,
    variant_id INTEGER REFERENCES variants(id),
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('page_view', 'scroll_depth', 'click', 'form_submit', 'conversion')),
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
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_checkout_sessions_session_id ON checkout_sessions(session_id);
CREATE INDEX idx_checkout_sessions_status ON checkout_sessions(status);

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

-- Content Sections Table (OT-P0-012, OT-P0-013: CUSTOM-SPLIT, CUSTOM-LIVE)
-- Stores customizable landing page sections for live preview editing
CREATE TABLE IF NOT EXISTS content_sections (
    id SERIAL PRIMARY KEY,
    variant_id INTEGER REFERENCES variants(id) ON DELETE CASCADE,
    section_type VARCHAR(50) NOT NULL CHECK (section_type IN ('hero', 'features', 'pricing', 'cta', 'testimonials', 'faq', 'footer', 'video')),
    content JSONB NOT NULL,
    "order" INTEGER DEFAULT 0,
    enabled BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_content_sections_variant ON content_sections(variant_id);
CREATE INDEX idx_content_sections_type ON content_sections(section_type);
CREATE INDEX idx_content_sections_order ON content_sections("order");
