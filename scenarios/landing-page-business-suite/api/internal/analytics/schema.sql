-- Analytics events. Variant configuration is file-backed, so the event keeps its slug.
CREATE TABLE IF NOT EXISTS metrics_events (id SERIAL PRIMARY KEY, variant_slug VARCHAR(100), event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('page_view','scroll_depth','click','form_submit','conversion','download')), event_data JSONB, session_id VARCHAR(255), visitor_id VARCHAR(255), created_at TIMESTAMP DEFAULT NOW());
CREATE INDEX IF NOT EXISTS idx_metrics_events_variant ON metrics_events(variant_slug);
CREATE INDEX IF NOT EXISTS idx_metrics_events_type ON metrics_events(event_type);
CREATE INDEX IF NOT EXISTS idx_metrics_events_created ON metrics_events(created_at);
CREATE INDEX IF NOT EXISTS idx_metrics_events_session ON metrics_events(session_id);
