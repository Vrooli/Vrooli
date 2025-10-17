-- Migration: Add webhook support for API update notifications
-- This adds tables and functionality for webhook subscriptions and event delivery

-- Webhook subscriptions table
CREATE TABLE IF NOT EXISTS webhook_subscriptions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    url VARCHAR(500) NOT NULL,
    events TEXT[] NOT NULL, -- Array of event types to subscribe to
    active BOOLEAN DEFAULT true,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    last_triggered TIMESTAMP,
    failure_count INTEGER DEFAULT 0,
    
    -- Additional metadata
    description TEXT,
    headers JSONB, -- Custom headers to include in webhook requests
    retry_policy JSONB, -- Custom retry configuration
    
    -- Constraints
    CONSTRAINT valid_url CHECK (url ~ '^https?://'),
    CONSTRAINT valid_events CHECK (array_length(events, 1) > 0)
);

-- Webhook delivery logs for audit and debugging
CREATE TABLE IF NOT EXISTS webhook_delivery_logs (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    webhook_id UUID REFERENCES webhook_subscriptions(id) ON DELETE CASCADE,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB NOT NULL,
    delivered_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    response_status INTEGER,
    response_time_ms INTEGER,
    error_message TEXT,
    retry_count INTEGER DEFAULT 0
);

-- Index for faster webhook lookups by event type
CREATE INDEX IF NOT EXISTS idx_webhook_events ON webhook_subscriptions USING GIN(events);
CREATE INDEX IF NOT EXISTS idx_webhook_active ON webhook_subscriptions(active) WHERE active = true;
CREATE INDEX IF NOT EXISTS idx_webhook_logs_delivered ON webhook_delivery_logs(delivered_at DESC);

-- Function to trigger webhooks when APIs are modified
CREATE OR REPLACE FUNCTION trigger_api_webhooks()
RETURNS TRIGGER AS $$
DECLARE
    event_type TEXT;
    event_data JSONB;
BEGIN
    -- Determine event type based on operation
    IF TG_OP = 'INSERT' THEN
        event_type := 'api.created';
        event_data := to_jsonb(NEW);
    ELSIF TG_OP = 'UPDATE' THEN
        -- Check for specific update types
        IF OLD.status != NEW.status AND NEW.status = 'deprecated' THEN
            event_type := 'api.deprecated';
        ELSE
            event_type := 'api.updated';
        END IF;
        event_data := jsonb_build_object(
            'old', to_jsonb(OLD),
            'new', to_jsonb(NEW),
            'changed_fields', (
                SELECT jsonb_object_agg(key, value)
                FROM jsonb_each(to_jsonb(NEW))
                WHERE to_jsonb(OLD) -> key IS DISTINCT FROM value
            )
        );
    ELSIF TG_OP = 'DELETE' THEN
        event_type := 'api.deleted';
        event_data := to_jsonb(OLD);
    END IF;
    
    -- Insert into a notifications queue (to be processed by the application)
    INSERT INTO webhook_notifications_queue (event_type, event_data)
    VALUES (event_type, event_data);
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create notification queue table
CREATE TABLE IF NOT EXISTS webhook_notifications_queue (
    id SERIAL PRIMARY KEY,
    event_type VARCHAR(100) NOT NULL,
    event_data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    processed BOOLEAN DEFAULT false,
    processed_at TIMESTAMP
);

-- Create triggers for API events (commented out by default to avoid automatic triggers)
-- Uncomment these to enable automatic webhook triggers on database changes
-- CREATE TRIGGER api_webhook_trigger
-- AFTER INSERT OR UPDATE OR DELETE ON apis
-- FOR EACH ROW EXECUTE FUNCTION trigger_api_webhooks();

-- Sample webhook event types documentation
COMMENT ON TABLE webhook_subscriptions IS 'Stores webhook subscriptions for API update notifications. 
Supported event types:
- api.created: New API added to the library
- api.updated: API information updated
- api.deleted: API removed from the library
- api.deprecated: API marked as deprecated
- api.configured: API credentials configured
- note.added: Note or gotcha added to an API
- version.added: New version tracked for an API
- price.updated: Pricing information updated';

-- Add sample webhook for testing (disabled by default)
-- INSERT INTO webhook_subscriptions (url, events, description)
-- VALUES (
--     'https://webhook.site/unique-url',
--     ARRAY['api.created', 'api.updated', 'api.deprecated'],
--     'Test webhook for API updates'
-- );