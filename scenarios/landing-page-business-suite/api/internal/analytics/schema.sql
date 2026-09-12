-- Analytics events. Variant configuration is file-backed, so the event keeps its slug.
CREATE TABLE IF NOT EXISTS metrics_events (
  id SERIAL PRIMARY KEY,
  variant_slug VARCHAR(100),
  event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('page_view','scroll_depth','click','form_submit','conversion','download')),
  event_data JSONB,
  event_id VARCHAR(64),
  session_id VARCHAR(255),
  visitor_id VARCHAR(255),
  referrer_host VARCHAR(255),
  referrer_kind VARCHAR(16) CHECK (referrer_kind IS NULL OR referrer_kind IN ('direct','search','social','referral','paid')),
  utm_source VARCHAR(128),
  utm_medium VARCHAR(128),
  utm_campaign VARCHAR(128),
  landing_path VARCHAR(512),
  country_code CHAR(2),
  device_class VARCHAR(16) CHECK (device_class IS NULL OR device_class IN ('desktop','mobile','tablet','unknown')),
  created_at TIMESTAMP DEFAULT NOW()
);
CREATE TABLE IF NOT EXISTS experiment_exposures (
  id SERIAL PRIMARY KEY,
  visitor_id VARCHAR(255) NOT NULL,
  variant_slug VARCHAR(100) NOT NULL,
  weight_fingerprint VARCHAR(64) NOT NULL,
  first_seen_at TIMESTAMP DEFAULT NOW(),
  UNIQUE (visitor_id, variant_slug, weight_fingerprint)
);
