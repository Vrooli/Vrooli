-- Payment settings domain schema.
--
-- Singleton row (id = 1) holding admin-configured Stripe credentials. When
-- present these take precedence over environment variables. Forward-only
-- declarative DDL.
CREATE TABLE IF NOT EXISTS payment_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    publishable_key TEXT,
    secret_key TEXT,
    webhook_secret TEXT,
    dashboard_url TEXT,
    updated_at TIMESTAMP DEFAULT NOW()
);
