-- Migration: 002_add_indexes
-- Description: Add performance indexes and helper functions
-- Author: AI Agent  
-- Date: 2025-01-12

-- Create indexes for users table
CREATE INDEX IF NOT EXISTS idx_users_auth_user_id ON users (auth_user_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users (created_at);

-- Create indexes for events table
CREATE INDEX IF NOT EXISTS idx_events_user_id ON events (user_id);
CREATE INDEX IF NOT EXISTS idx_events_start_time ON events (start_time);
CREATE INDEX IF NOT EXISTS idx_events_end_time ON events (end_time);
CREATE INDEX IF NOT EXISTS idx_events_status ON events (status);
CREATE INDEX IF NOT EXISTS idx_events_type ON events (event_type);
CREATE INDEX IF NOT EXISTS idx_events_created_at ON events (created_at);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_events_user_time ON events (user_id, start_time);
CREATE INDEX IF NOT EXISTS idx_events_user_status ON events (user_id, status);
CREATE INDEX IF NOT EXISTS idx_events_time_range ON events (start_time, end_time);

-- Full-text search indexes
CREATE INDEX IF NOT EXISTS idx_events_title_search ON events USING gin (to_tsvector('english', title));
CREATE INDEX IF NOT EXISTS idx_events_description_search ON events USING gin (to_tsvector('english', coalesce(description, '')));

-- JSONB indexes for metadata queries
CREATE INDEX IF NOT EXISTS idx_events_metadata ON events USING gin (metadata);
CREATE INDEX IF NOT EXISTS idx_events_automation ON events USING gin (automation_config);

-- Create indexes for event_reminders table
CREATE INDEX IF NOT EXISTS idx_reminders_event_id ON event_reminders (event_id);
CREATE INDEX IF NOT EXISTS idx_reminders_scheduled_time ON event_reminders (scheduled_time);
CREATE INDEX IF NOT EXISTS idx_reminders_status ON event_reminders (status);
CREATE INDEX IF NOT EXISTS idx_reminders_type ON event_reminders (notification_type);

-- Index for processing pending reminders
CREATE INDEX IF NOT EXISTS idx_reminders_pending ON event_reminders (scheduled_time, status) WHERE status = 'pending';

-- Create indexes for recurring_patterns table
CREATE INDEX IF NOT EXISTS idx_recurring_parent_event ON recurring_patterns (parent_event_id);
CREATE INDEX IF NOT EXISTS idx_recurring_pattern_type ON recurring_patterns (pattern_type);
CREATE INDEX IF NOT EXISTS idx_recurring_end_date ON recurring_patterns (end_date);

-- Create indexes for event_embeddings table
CREATE INDEX IF NOT EXISTS idx_embeddings_event_id ON event_embeddings (event_id);
CREATE INDEX IF NOT EXISTS idx_embeddings_qdrant_id ON event_embeddings (qdrant_point_id);
CREATE INDEX IF NOT EXISTS idx_embeddings_content_hash ON event_embeddings (content_hash);
CREATE INDEX IF NOT EXISTS idx_embeddings_keywords ON event_embeddings USING gin (keywords);

-- Create helper functions
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for updated_at columns
DROP TRIGGER IF EXISTS tr_users_updated_at ON users;
CREATE TRIGGER tr_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS tr_events_updated_at ON events;
CREATE TRIGGER tr_events_updated_at 
    BEFORE UPDATE ON events 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS tr_embeddings_updated_at ON event_embeddings;
CREATE TRIGGER tr_embeddings_updated_at 
    BEFORE UPDATE ON event_embeddings 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Helper function for reminder time calculation
CREATE OR REPLACE FUNCTION calculate_reminder_time(
    event_start_time TIMESTAMPTZ,
    minutes_before INTEGER
) RETURNS TIMESTAMPTZ AS $$
BEGIN
    RETURN event_start_time - INTERVAL '1 minute' * minutes_before;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Helper function to generate recurring events
CREATE OR REPLACE FUNCTION generate_recurring_events(
    parent_event_id UUID,
    pattern_type VARCHAR,
    interval_val INTEGER DEFAULT 1,
    max_occurrences INTEGER DEFAULT 100
) RETURNS TABLE(occurrence_date TIMESTAMPTZ) AS $$
DECLARE
    parent_event RECORD;
    base_date TIMESTAMPTZ;
    end_limit TIMESTAMPTZ;
    counter INTEGER := 0;
BEGIN
    SELECT start_time, end_time INTO parent_event 
    FROM events WHERE id = parent_event_id;
    
    IF parent_event IS NULL THEN
        RETURN;
    END IF;
    
    base_date := parent_event.start_time;
    end_limit := base_date + INTERVAL '1 year';
    
    WHILE counter < max_occurrences AND base_date < end_limit LOOP
        occurrence_date := base_date;
        RETURN NEXT;
        
        CASE pattern_type
            WHEN 'daily' THEN
                base_date := base_date + INTERVAL '1 day' * interval_val;
            WHEN 'weekly' THEN
                base_date := base_date + INTERVAL '1 week' * interval_val;
            WHEN 'monthly' THEN
                base_date := base_date + INTERVAL '1 month' * interval_val;
            WHEN 'yearly' THEN
                base_date := base_date + INTERVAL '1 year' * interval_val;
            ELSE
                EXIT;
        END CASE;
        
        counter := counter + 1;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- Create views for common queries
CREATE OR REPLACE VIEW upcoming_events AS
SELECT 
    e.id,
    e.title,
    e.description,
    e.start_time,
    e.end_time,
    e.timezone,
    e.location,
    e.event_type,
    e.status,
    u.display_name as user_name,
    u.email as user_email,
    (e.start_time - NOW()) as time_until_start,
    CASE 
        WHEN e.start_time <= NOW() AND e.end_time >= NOW() THEN 'happening_now'
        WHEN e.start_time > NOW() THEN 'upcoming'
        ELSE 'past'
    END as time_status
FROM events e
JOIN users u ON e.user_id = u.id
WHERE e.status = 'active'
ORDER BY e.start_time;

CREATE OR REPLACE VIEW pending_reminders AS
SELECT 
    r.id,
    r.event_id,
    r.scheduled_time,
    r.notification_type,
    r.minutes_before,
    r.retry_count,
    e.title as event_title,
    e.start_time as event_start_time,
    u.email as user_email,
    u.display_name as user_name,
    u.timezone as user_timezone
FROM event_reminders r
JOIN events e ON r.event_id = e.id
JOIN users u ON e.user_id = u.id
WHERE r.status = 'pending' 
  AND r.scheduled_time <= NOW() + INTERVAL '5 minutes'
  AND r.retry_count < 5
ORDER BY r.scheduled_time;

-- Performance analysis function
CREATE OR REPLACE FUNCTION analyze_calendar_performance() 
RETURNS TABLE(
    table_name TEXT,
    row_count BIGINT,
    table_size TEXT,
    index_size TEXT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.table_name::TEXT,
        t.row_count::BIGINT,
        pg_size_pretty(t.table_size)::TEXT,
        pg_size_pretty(t.index_size)::TEXT
    FROM (
        SELECT 
            'users' as table_name,
            (SELECT count(*) FROM users) as row_count,
            pg_total_relation_size('users') as table_size,
            pg_indexes_size('users') as index_size
        UNION ALL
        SELECT 
            'events' as table_name,
            (SELECT count(*) FROM events) as row_count,
            pg_total_relation_size('events') as table_size,
            pg_indexes_size('events') as index_size
        UNION ALL
        SELECT 
            'event_reminders' as table_name,
            (SELECT count(*) FROM event_reminders) as row_count,
            pg_total_relation_size('event_reminders') as table_size,
            pg_indexes_size('event_reminders') as index_size
        UNION ALL
        SELECT 
            'recurring_patterns' as table_name,
            (SELECT count(*) FROM recurring_patterns) as row_count,
            pg_total_relation_size('recurring_patterns') as table_size,
            pg_indexes_size('recurring_patterns') as index_size
    ) t;
END;
$$ LANGUAGE plpgsql;

-- Record this migration as applied
INSERT INTO schema_migrations (version, checksum) 
VALUES ('002_add_indexes', 'f1e2d3c4b5a6978563214785896321470');