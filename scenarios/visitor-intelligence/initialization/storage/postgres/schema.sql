-- Visitor Intelligence PostgreSQL Schema
-- Optimized for high-volume inserts and time-series analytics

-- Enable UUID extension for generating unique identifiers
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Enable btree_gin extension for better indexing performance
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- Visitors table - core visitor profiles
CREATE TABLE IF NOT EXISTS visitors (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    fingerprint VARCHAR(64) NOT NULL UNIQUE,
    first_seen TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    last_seen TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    session_count INTEGER NOT NULL DEFAULT 1,
    identified BOOLEAN NOT NULL DEFAULT FALSE,
    
    -- Optional user identification
    email VARCHAR(255) NULL,
    name VARCHAR(255) NULL,
    
    -- Browser/device information
    user_agent TEXT,
    ip_address INET,
    timezone VARCHAR(50),
    language VARCHAR(10),
    screen_resolution VARCHAR(20),
    device_type VARCHAR(20),
    
    -- Behavioral summary
    total_page_views INTEGER NOT NULL DEFAULT 0,
    total_session_duration INTEGER NOT NULL DEFAULT 0, -- in seconds
    
    -- Tagging and segmentation
    tags TEXT[] DEFAULT '{}',
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Visitor sessions table - individual visit sessions
CREATE TABLE IF NOT EXISTS visitor_sessions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    visitor_id UUID NOT NULL REFERENCES visitors(id) ON DELETE CASCADE,
    scenario VARCHAR(100) NOT NULL,
    
    -- Session timing
    start_time TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    end_time TIMESTAMP WITH TIME ZONE NULL,
    duration INTEGER NULL, -- in seconds
    
    -- Page activity
    page_views INTEGER NOT NULL DEFAULT 0,
    entry_page TEXT,
    exit_page TEXT,
    
    -- Traffic source
    referrer TEXT,
    utm_source VARCHAR(100),
    utm_medium VARCHAR(100),
    utm_campaign VARCHAR(100),
    utm_term VARCHAR(100),
    utm_content VARCHAR(100),
    
    -- Geographic/device context
    ip_address INET,
    country VARCHAR(2),
    city VARCHAR(100),
    device_type VARCHAR(20),
    
    -- Session quality metrics
    bounce BOOLEAN DEFAULT FALSE,
    engaged BOOLEAN DEFAULT FALSE, -- session > 30s or >1 pageview
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Visitor events table - granular behavioral tracking
-- Partitioned by date for performance with high-volume data
CREATE TABLE IF NOT EXISTS visitor_events (
    id UUID DEFAULT uuid_generate_v4(),
    visitor_id UUID NOT NULL,
    session_id UUID NOT NULL,
    scenario VARCHAR(100) NOT NULL,
    
    -- Event details
    event_type VARCHAR(50) NOT NULL,
    page_url TEXT NOT NULL,
    timestamp TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    -- Event properties (flexible JSON storage)
    properties JSONB DEFAULT '{}',
    
    -- Quick access fields (for common queries)
    element_tag VARCHAR(20),
    element_id VARCHAR(100),
    element_class VARCHAR(200),
    
    -- Primary key includes timestamp for partitioning
    PRIMARY KEY (id, timestamp)
) PARTITION BY RANGE (timestamp);

-- Create event partitions for the current and next month
-- (Additional partitions should be created via automated scripts)
CREATE TABLE IF NOT EXISTS visitor_events_current PARTITION OF visitor_events
    FOR VALUES FROM (date_trunc('month', CURRENT_DATE)) 
    TO (date_trunc('month', CURRENT_DATE + INTERVAL '1 month'));

CREATE TABLE IF NOT EXISTS visitor_events_next PARTITION OF visitor_events
    FOR VALUES FROM (date_trunc('month', CURRENT_DATE + INTERVAL '1 month'))
    TO (date_trunc('month', CURRENT_DATE + INTERVAL '2 months'));

-- Visitor segments table - for marketing automation
CREATE TABLE IF NOT EXISTS visitor_segments (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    
    -- Segment criteria (stored as JSON for flexibility)
    criteria JSONB NOT NULL,
    
    -- Segment stats
    visitor_count INTEGER NOT NULL DEFAULT 0,
    last_calculated TIMESTAMP WITH TIME ZONE,
    
    -- Metadata
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

-- Visitor segment memberships (many-to-many)
CREATE TABLE IF NOT EXISTS visitor_segment_memberships (
    visitor_id UUID NOT NULL REFERENCES visitors(id) ON DELETE CASCADE,
    segment_id UUID NOT NULL REFERENCES visitor_segments(id) ON DELETE CASCADE,
    added_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    
    PRIMARY KEY (visitor_id, segment_id)
);

-- Performance indexes
-- Visitors table indexes
CREATE INDEX IF NOT EXISTS idx_visitors_fingerprint ON visitors(fingerprint);
CREATE INDEX IF NOT EXISTS idx_visitors_last_seen ON visitors(last_seen DESC);
CREATE INDEX IF NOT EXISTS idx_visitors_identified ON visitors(identified) WHERE identified = TRUE;
CREATE INDEX IF NOT EXISTS idx_visitors_email ON visitors(email) WHERE email IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_visitors_tags ON visitors USING GIN(tags);

-- Visitor sessions indexes  
CREATE INDEX IF NOT EXISTS idx_visitor_sessions_visitor_id ON visitor_sessions(visitor_id);
CREATE INDEX IF NOT EXISTS idx_visitor_sessions_scenario ON visitor_sessions(scenario);
CREATE INDEX IF NOT EXISTS idx_visitor_sessions_start_time ON visitor_sessions(start_time DESC);
CREATE INDEX IF NOT EXISTS idx_visitor_sessions_utm_source ON visitor_sessions(utm_source) WHERE utm_source IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_visitor_sessions_referrer ON visitor_sessions(referrer) WHERE referrer IS NOT NULL;

-- Visitor events indexes (on partitions)
CREATE INDEX IF NOT EXISTS idx_visitor_events_visitor_id ON visitor_events(visitor_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_visitor_events_session_id ON visitor_events(session_id, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_visitor_events_scenario ON visitor_events(scenario, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_visitor_events_type ON visitor_events(event_type, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_visitor_events_properties ON visitor_events USING GIN(properties);

-- Composite indexes for common query patterns
CREATE INDEX IF NOT EXISTS idx_visitor_events_scenario_type ON visitor_events(scenario, event_type, timestamp DESC);
CREATE INDEX IF NOT EXISTS idx_visitor_sessions_scenario_time ON visitor_sessions(scenario, start_time DESC);

-- Trigger to update visitor last_seen timestamp
CREATE OR REPLACE FUNCTION update_visitor_last_seen()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE visitors 
    SET 
        last_seen = NEW.timestamp,
        updated_at = NOW()
    WHERE id = NEW.visitor_id;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER trigger_update_visitor_last_seen
    AFTER INSERT ON visitor_events
    FOR EACH ROW
    EXECUTE FUNCTION update_visitor_last_seen();

-- Trigger to update session end_time and duration
CREATE OR REPLACE FUNCTION update_session_metrics()
RETURNS TRIGGER AS $$
BEGIN
    -- Update the session with latest activity
    UPDATE visitor_sessions 
    SET 
        end_time = NEW.timestamp,
        duration = EXTRACT(EPOCH FROM (NEW.timestamp - start_time))::INTEGER,
        updated_at = NOW()
    WHERE id = NEW.session_id;
    
    -- Increment page view count for pageview events
    IF NEW.event_type = 'pageview' THEN
        UPDATE visitor_sessions 
        SET page_views = page_views + 1
        WHERE id = NEW.session_id;
        
        UPDATE visitors
        SET total_page_views = total_page_views + 1
        WHERE id = NEW.visitor_id;
    END IF;
    
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER trigger_update_session_metrics
    AFTER INSERT ON visitor_events
    FOR EACH ROW
    EXECUTE FUNCTION update_session_metrics();

-- Function to calculate visitor engagement score
CREATE OR REPLACE FUNCTION calculate_engagement_score(visitor_uuid UUID)
RETURNS INTEGER AS $$
DECLARE
    score INTEGER := 0;
    visitor_data RECORD;
    session_data RECORD;
BEGIN
    -- Get visitor basic data
    SELECT 
        session_count,
        total_page_views,
        total_session_duration,
        identified
    INTO visitor_data
    FROM visitors WHERE id = visitor_uuid;
    
    IF NOT FOUND THEN
        RETURN 0;
    END IF;
    
    -- Base scoring
    score := score + LEAST(visitor_data.session_count * 10, 100); -- Up to 100 for sessions
    score := score + LEAST(visitor_data.total_page_views * 2, 100); -- Up to 100 for pageviews  
    score := score + LEAST(visitor_data.total_session_duration / 60, 50); -- Up to 50 for time
    
    -- Identification bonus
    IF visitor_data.identified THEN
        score := score + 50;
    END IF;
    
    -- Recent activity bonus (last 7 days)
    SELECT COUNT(*) INTO session_data
    FROM visitor_sessions
    WHERE visitor_id = visitor_uuid 
    AND start_time > NOW() - INTERVAL '7 days';
    
    IF session_data > 0 THEN
        score := score + 25;
    END IF;
    
    RETURN LEAST(score, 500); -- Cap at 500
END;
$$ LANGUAGE plpgsql;

-- Materialized view for visitor analytics (refresh periodically)
CREATE MATERIALIZED VIEW IF NOT EXISTS visitor_analytics AS
SELECT 
    scenario,
    DATE(v.created_at) as date,
    COUNT(DISTINCT v.id) as unique_visitors,
    COUNT(vs.id) as total_sessions,
    AVG(vs.duration) as avg_session_duration,
    SUM(vs.page_views) as total_page_views,
    COUNT(CASE WHEN vs.bounce THEN 1 END)::FLOAT / COUNT(vs.id) as bounce_rate,
    COUNT(CASE WHEN v.identified THEN 1 END) as identified_visitors
FROM visitors v
LEFT JOIN visitor_sessions vs ON v.id = vs.visitor_id
GROUP BY scenario, DATE(v.created_at)
ORDER BY scenario, date DESC;

-- Create unique index on materialized view for efficient refreshes
CREATE UNIQUE INDEX IF NOT EXISTS idx_visitor_analytics_unique 
ON visitor_analytics(scenario, date);

-- Insert default visitor segments
INSERT INTO visitor_segments (name, description, criteria) VALUES
('New Visitors', 'First-time visitors within last 24 hours', 
 '{"session_count": {"operator": "eq", "value": 1}, "first_seen": {"operator": "gte", "value": "24h"}}'),
('Returning Visitors', 'Visitors with more than one session',
 '{"session_count": {"operator": "gt", "value": 1}}'),
('High Engagement', 'Visitors with >5 page views or >5min session time',
 '{"or": [{"total_page_views": {"operator": "gt", "value": 5}}, {"total_session_duration": {"operator": "gt", "value": 300}}]}'),
('Identified Users', 'Visitors who have provided identification',
 '{"identified": {"operator": "eq", "value": true}}'),
('Recent Activity', 'Visitors active in last 7 days',
 '{"last_seen": {"operator": "gte", "value": "7d"}}')
ON CONFLICT (name) DO NOTHING;

-- Comments for documentation
COMMENT ON TABLE visitors IS 'Core visitor profiles with fingerprint-based identification';
COMMENT ON TABLE visitor_sessions IS 'Individual visit sessions with traffic source attribution';
COMMENT ON TABLE visitor_events IS 'Granular behavioral tracking events (partitioned by timestamp)';
COMMENT ON TABLE visitor_segments IS 'Dynamic visitor segments for marketing automation';
COMMENT ON COLUMN visitors.fingerprint IS 'Browser fingerprint hash for anonymous identification';
COMMENT ON COLUMN visitor_events.properties IS 'Flexible JSON storage for event-specific data';
COMMENT ON FUNCTION calculate_engagement_score IS 'Calculate visitor engagement score (0-500)';
COMMENT ON MATERIALIZED VIEW visitor_analytics IS 'Pre-aggregated analytics data (refresh periodically)';

-- Success message
DO $$
BEGIN
    RAISE NOTICE 'Visitor Intelligence database schema initialized successfully';
    RAISE NOTICE 'Tables created: visitors, visitor_sessions, visitor_events (partitioned), visitor_segments';
    RAISE NOTICE 'Indexes optimized for high-volume tracking and analytics queries';
    RAISE NOTICE 'Triggers configured for automatic metric updates';
END $$;