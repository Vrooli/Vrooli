-- Calendar System Database Schema
-- Version: 1.0.0
-- Description: Complete schema for calendar events, users, and scheduling functionality
-- Compatible with PostgreSQL 12+

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- For full-text search
CREATE EXTENSION IF NOT EXISTS "btree_gin"; -- For composite indexes

-- Drop existing tables in dependency order (for development/testing)
DROP TABLE IF EXISTS event_reminders CASCADE;
DROP TABLE IF EXISTS recurring_patterns CASCADE;
DROP TABLE IF EXISTS event_embeddings CASCADE;
DROP TABLE IF EXISTS events CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- ============================================================================
-- USERS TABLE
-- ============================================================================
-- Stores user profile information and preferences
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    auth_user_id UUID NOT NULL UNIQUE, -- Reference to scenario-authenticator user
    email VARCHAR(255) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    preferences JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes for users table
CREATE INDEX idx_users_auth_user_id ON users (auth_user_id);
CREATE INDEX idx_users_email ON users (email);
CREATE INDEX idx_users_created_at ON users (created_at);

-- ============================================================================
-- EVENTS TABLE  
-- ============================================================================
-- Core events table with all calendar event information
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    timezone VARCHAR(50) NOT NULL DEFAULT 'UTC',
    location VARCHAR(500),
    event_type VARCHAR(50) NOT NULL DEFAULT 'meeting',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    metadata JSONB DEFAULT '{}',
    automation_config JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT chk_events_time_order CHECK (start_time < end_time),
    CONSTRAINT chk_events_status CHECK (status IN ('active', 'cancelled', 'completed')),
    CONSTRAINT chk_events_type CHECK (event_type IN ('meeting', 'appointment', 'task', 'reminder', 'block', 'personal', 'work', 'travel'))
);

-- Indexes for events table (critical for performance)
CREATE INDEX idx_events_user_id ON events (user_id);
CREATE INDEX idx_events_start_time ON events (start_time);
CREATE INDEX idx_events_end_time ON events (end_time);
CREATE INDEX idx_events_status ON events (status);
CREATE INDEX idx_events_type ON events (event_type);
CREATE INDEX idx_events_created_at ON events (created_at);

-- Composite indexes for common queries
CREATE INDEX idx_events_user_time ON events (user_id, start_time);
CREATE INDEX idx_events_user_status ON events (user_id, status);
CREATE INDEX idx_events_time_range ON events (start_time, end_time);

-- Full-text search index
CREATE INDEX idx_events_title_search ON events USING gin (to_tsvector('english', title));
CREATE INDEX idx_events_description_search ON events USING gin (to_tsvector('english', coalesce(description, '')));

-- JSONB indexes for metadata queries
CREATE INDEX idx_events_metadata ON events USING gin (metadata);
CREATE INDEX idx_events_automation ON events USING gin (automation_config);

-- ============================================================================
-- RECURRING PATTERNS TABLE
-- ============================================================================
-- Defines recurring event patterns and rules
CREATE TABLE recurring_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    parent_event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    pattern_type VARCHAR(20) NOT NULL,
    interval_value INTEGER NOT NULL DEFAULT 1,
    days_of_week INTEGER[], -- Array of day numbers (0=Sunday, 1=Monday, etc.)
    days_of_month INTEGER[], -- Array of month day numbers (1-31)
    weeks_of_month INTEGER[], -- Array of week numbers (1-5, -1 for last)
    months_of_year INTEGER[], -- Array of month numbers (1-12)
    end_date TIMESTAMPTZ,
    max_occurrences INTEGER,
    exceptions TIMESTAMPTZ[], -- Array of exception dates
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT chk_recurring_pattern_type CHECK (
        pattern_type IN ('daily', 'weekly', 'monthly', 'yearly', 'custom')
    ),
    CONSTRAINT chk_recurring_interval CHECK (interval_value > 0),
    CONSTRAINT chk_recurring_days_week CHECK (
        days_of_week IS NULL OR 
        array_length(days_of_week, 1) IS NULL OR 
        array_length(days_of_week, 1) = 0
    ),
    CONSTRAINT chk_recurring_days_month CHECK (
        days_of_month IS NULL OR 
        array_length(days_of_month, 1) IS NULL OR 
        array_length(days_of_month, 1) = 0
    )
);

-- Indexes for recurring patterns
CREATE INDEX idx_recurring_parent_event ON recurring_patterns (parent_event_id);
CREATE INDEX idx_recurring_pattern_type ON recurring_patterns (pattern_type);
CREATE INDEX idx_recurring_end_date ON recurring_patterns (end_date);

-- ============================================================================
-- EVENT REMINDERS TABLE
-- ============================================================================
-- Stores reminder configurations and delivery status
CREATE TABLE event_reminders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    minutes_before INTEGER NOT NULL,
    notification_type VARCHAR(20) NOT NULL DEFAULT 'email',
    scheduled_time TIMESTAMPTZ NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    notification_id VARCHAR(255), -- Reference to notification-hub
    delivered_at TIMESTAMPTZ,
    error_message TEXT,
    retry_count INTEGER NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT chk_reminder_minutes CHECK (minutes_before >= 0),
    CONSTRAINT chk_reminder_type CHECK (
        notification_type IN ('email', 'sms', 'push', 'webhook')
    ),
    CONSTRAINT chk_reminder_status CHECK (
        status IN ('pending', 'sent', 'delivered', 'failed', 'cancelled')
    ),
    CONSTRAINT chk_reminder_retry CHECK (retry_count >= 0 AND retry_count <= 10)
);

-- Indexes for event reminders
CREATE INDEX idx_reminders_event_id ON event_reminders (event_id);
CREATE INDEX idx_reminders_scheduled_time ON event_reminders (scheduled_time);
CREATE INDEX idx_reminders_status ON event_reminders (status);
CREATE INDEX idx_reminders_type ON event_reminders (notification_type);

-- Index for processing pending reminders
CREATE INDEX idx_reminders_pending ON event_reminders (scheduled_time, status) 
WHERE status = 'pending';

-- ============================================================================
-- EVENT EMBEDDINGS TABLE (for AI search)
-- ============================================================================
-- Stores vector embeddings for semantic search using Qdrant
-- This table stores references and metadata; actual vectors are in Qdrant
CREATE TABLE event_embeddings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    qdrant_point_id UUID NOT NULL,
    embedding_version VARCHAR(20) NOT NULL DEFAULT 'v1.0',
    content_hash VARCHAR(64) NOT NULL, -- Hash of content used to generate embedding
    keywords TEXT[], -- Extracted keywords for fallback search
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Ensure one embedding per event (can be updated)
    CONSTRAINT uq_event_embedding UNIQUE (event_id)
);

-- Indexes for embeddings
CREATE INDEX idx_embeddings_event_id ON event_embeddings (event_id);
CREATE INDEX idx_embeddings_qdrant_id ON event_embeddings (qdrant_point_id);
CREATE INDEX idx_embeddings_content_hash ON event_embeddings (content_hash);
CREATE INDEX idx_embeddings_keywords ON event_embeddings USING gin (keywords);

-- ============================================================================
-- TRIGGERS AND FUNCTIONS
-- ============================================================================

-- Function to update the updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Triggers for updated_at columns
CREATE TRIGGER tr_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_events_updated_at 
    BEFORE UPDATE ON events 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_embeddings_updated_at 
    BEFORE UPDATE ON event_embeddings 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to automatically create reminder times based on event start time
CREATE OR REPLACE FUNCTION calculate_reminder_time(
    event_start_time TIMESTAMPTZ,
    minutes_before INTEGER
) RETURNS TIMESTAMPTZ AS $$
BEGIN
    RETURN event_start_time - INTERVAL '1 minute' * minutes_before;
END;
$$ LANGUAGE plpgsql IMMUTABLE;

-- Function to generate event recurrences (simplified version)
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
    -- Get parent event details
    SELECT start_time, end_time INTO parent_event 
    FROM events WHERE id = parent_event_id;
    
    IF parent_event IS NULL THEN
        RETURN;
    END IF;
    
    base_date := parent_event.start_time;
    end_limit := base_date + INTERVAL '1 year'; -- Safety limit
    
    -- Simple recurring generation (can be enhanced)
    WHILE counter < max_occurrences AND base_date < end_limit LOOP
        occurrence_date := base_date;
        RETURN NEXT;
        
        -- Calculate next occurrence based on pattern
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
                EXIT; -- Unknown pattern
        END CASE;
        
        counter := counter + 1;
    END LOOP;
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- VIEWS FOR COMMON QUERIES
-- ============================================================================

-- View for upcoming events with user information
CREATE VIEW upcoming_events AS
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

-- View for pending reminders ready to be sent
CREATE VIEW pending_reminders AS
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

-- ============================================================================
-- INITIAL DATA AND CONFIGURATIONS
-- ============================================================================

-- Insert system configurations (if needed)
-- This can be extended for system-wide calendar settings

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

-- ============================================================================
-- SCHEMA VALIDATION
-- ============================================================================

-- Verify all tables were created successfully
DO $$
DECLARE
    table_count INTEGER;
    index_count INTEGER;
    function_count INTEGER;
BEGIN
    SELECT count(*) INTO table_count 
    FROM information_schema.tables 
    WHERE table_schema = 'public' 
    AND table_name IN ('users', 'events', 'event_reminders', 'recurring_patterns', 'event_embeddings');
    
    SELECT count(*) INTO index_count
    FROM pg_indexes 
    WHERE schemaname = 'public'
    AND indexname LIKE 'idx_%';
    
    SELECT count(*) INTO function_count
    FROM pg_proc p
    JOIN pg_namespace n ON p.pronamespace = n.oid
    WHERE n.nspname = 'public'
    AND p.proname IN ('update_updated_at_column', 'calculate_reminder_time', 'generate_recurring_events');
    
    RAISE NOTICE 'Calendar schema installation complete:';
    RAISE NOTICE '  - Tables created: %', table_count;
    RAISE NOTICE '  - Indexes created: %', index_count;
    RAISE NOTICE '  - Functions created: %', function_count;
    RAISE NOTICE '  - Views created: 2 (upcoming_events, pending_reminders)';
    
    IF table_count = 5 AND function_count >= 3 THEN
        RAISE NOTICE '✅ Schema installation successful!';
    ELSE
        RAISE WARNING '⚠️  Schema installation may be incomplete';
    END IF;
END;
$$;