-- Metrics domain schema.
--
-- Append-only analytics event log. Foreign-keys the variant each event belongs
-- to; applied after the variant schema. Forward-only declarative DDL.
CREATE TABLE IF NOT EXISTS metrics_events (
    id SERIAL PRIMARY KEY,
    variant_id INTEGER REFERENCES variants(id),
    event_type VARCHAR(50) NOT NULL CHECK (event_type IN ('page_view', 'scroll_depth', 'click', 'form_submit', 'conversion', 'download')),
    event_data JSONB,
    session_id VARCHAR(255),
    visitor_id VARCHAR(255),
    created_at TIMESTAMP DEFAULT NOW()
);
CREATE INDEX IF NOT EXISTS idx_metrics_events_variant ON metrics_events(variant_id);
CREATE INDEX IF NOT EXISTS idx_metrics_events_type ON metrics_events(event_type);
CREATE INDEX IF NOT EXISTS idx_metrics_events_created ON metrics_events(created_at);
CREATE INDEX IF NOT EXISTS idx_metrics_events_session ON metrics_events(session_id);
