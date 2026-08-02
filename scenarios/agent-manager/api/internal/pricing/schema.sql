-- ============================================================================
-- Model Pricing - Cached pricing data from providers
-- ============================================================================
CREATE TABLE IF NOT EXISTS price_book_revisions (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    fetched_at TEXT NOT NULL,
    source_digest TEXT NOT NULL,
    model_count INTEGER NOT NULL DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_price_book_revisions_provider_fetched
    ON price_book_revisions(provider, fetched_at);

-- Subscription periods are immutable billing facts used when a resource is
-- covered by a plan rather than metered per token.
CREATE TABLE IF NOT EXISTS subscription_periods (
    id TEXT PRIMARY KEY,
    provider TEXT NOT NULL,
    plan_ref TEXT NOT NULL,
    starts_at TEXT NOT NULL,
    ends_at TEXT NOT NULL,
    amount_micro_usd INTEGER NOT NULL DEFAULT 0,
    quota_tokens INTEGER NOT NULL DEFAULT 0,
    created_at TEXT DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_subscription_periods_provider_window
    ON subscription_periods(provider, starts_at, ends_at);

CREATE TABLE IF NOT EXISTS model_pricing (
    id TEXT PRIMARY KEY,
    canonical_model_name TEXT NOT NULL,
    provider TEXT NOT NULL,
    price_book_revision_id TEXT,

    -- Per-component pricing (USD per token)
    input_token_price REAL,
    output_token_price REAL,
    cache_read_price REAL,
    cache_creation_price REAL,
    web_search_price REAL,
    server_tool_use_price REAL,

    -- Per-component sources (manual_override, provider_api, historical_average, unknown)
    input_token_source TEXT DEFAULT 'unknown',
    output_token_source TEXT DEFAULT 'unknown',
    cache_read_source TEXT,
    cache_creation_source TEXT,
    web_search_source TEXT,
    server_tool_use_source TEXT,

    -- Metadata
    fetched_at TEXT NOT NULL,
    expires_at TEXT NOT NULL,
    pricing_version TEXT,

    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),

    UNIQUE(canonical_model_name, provider)
);

CREATE INDEX IF NOT EXISTS idx_model_pricing_model ON model_pricing(canonical_model_name);
CREATE INDEX IF NOT EXISTS idx_model_pricing_provider ON model_pricing(provider);
CREATE INDEX IF NOT EXISTS idx_model_pricing_expires ON model_pricing(expires_at);
CREATE INDEX IF NOT EXISTS idx_model_pricing_revision ON model_pricing(price_book_revision_id);

-- ============================================================================
-- Model Aliases - Maps runner model names to canonical names
-- ============================================================================
CREATE TABLE IF NOT EXISTS model_aliases (
    id TEXT PRIMARY KEY,
    runner_model TEXT NOT NULL,
    runner_type TEXT NOT NULL,
    canonical_model TEXT NOT NULL,
    provider TEXT NOT NULL,

    created_at TEXT DEFAULT (datetime('now')),
    updated_at TEXT DEFAULT (datetime('now')),

    UNIQUE(runner_model, runner_type)
);

CREATE INDEX IF NOT EXISTS idx_model_aliases_runner ON model_aliases(runner_model, runner_type);
CREATE INDEX IF NOT EXISTS idx_model_aliases_canonical ON model_aliases(canonical_model);

-- ============================================================================
-- Manual Price Overrides - User-specified pricing overrides
-- ============================================================================
CREATE TABLE IF NOT EXISTS manual_price_overrides (
    id TEXT PRIMARY KEY,
    canonical_model_name TEXT NOT NULL,
    component TEXT NOT NULL,
    price_usd REAL NOT NULL,
    note TEXT,
    created_by TEXT,

    created_at TEXT DEFAULT (datetime('now')),
    expires_at TEXT,

    UNIQUE(canonical_model_name, component)
);

CREATE INDEX IF NOT EXISTS idx_manual_overrides_model ON manual_price_overrides(canonical_model_name);
CREATE INDEX IF NOT EXISTS idx_manual_overrides_expires ON manual_price_overrides(expires_at);

-- ============================================================================
-- Pricing Settings - Global pricing configuration (singleton table)
-- ============================================================================
CREATE TABLE IF NOT EXISTS pricing_settings (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    historical_average_days INTEGER DEFAULT 7,
    provider_cache_ttl_seconds INTEGER DEFAULT 21600,

    updated_at TEXT DEFAULT (datetime('now'))
);

-- Insert default settings if not exists
INSERT OR IGNORE INTO pricing_settings (id, historical_average_days, provider_cache_ttl_seconds)
VALUES (1, 7, 21600);

CREATE TRIGGER IF NOT EXISTS update_model_pricing_updated_at
    AFTER UPDATE ON model_pricing
    FOR EACH ROW
BEGIN
    UPDATE model_pricing SET updated_at = datetime('now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_model_aliases_updated_at
    AFTER UPDATE ON model_aliases
    FOR EACH ROW
BEGIN
    UPDATE model_aliases SET updated_at = datetime('now') WHERE id = NEW.id;
END;

CREATE TRIGGER IF NOT EXISTS update_pricing_settings_updated_at
    AFTER UPDATE ON pricing_settings
    FOR EACH ROW
BEGIN
    UPDATE pricing_settings SET updated_at = datetime('now') WHERE id = NEW.id;
END;
