-- Calendar System Database Schema
-- Created: 2024-01-04
-- Purpose: Universal scheduling intelligence for Vrooli scenarios

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gin";

-- Create enum types
CREATE TYPE event_status AS ENUM ('active', 'cancelled', 'completed');
CREATE TYPE recurrence_type AS ENUM ('daily', 'weekly', 'monthly', 'yearly', 'custom');
CREATE TYPE reminder_status AS ENUM ('pending', 'sent', 'failed', 'cancelled');
CREATE TYPE notification_type AS ENUM ('email', 'sms', 'push', 'webhook');

-- Users table (may reference scenario-authenticator users)
-- This is a local cache for performance, synced with auth service
CREATE TABLE IF NOT EXISTS users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    auth_user_id UUID NOT NULL UNIQUE, -- References scenario-authenticator user
    email VARCHAR(255) NOT NULL,
    display_name VARCHAR(100),
    timezone VARCHAR(50) DEFAULT 'UTC',
    preferences JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    last_seen TIMESTAMPTZ DEFAULT NOW()
);

-- Events table - core scheduling data
CREATE TABLE events (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Event details
    title VARCHAR(255) NOT NULL,
    description TEXT,
    location VARCHAR(500),
    event_type VARCHAR(50) DEFAULT 'meeting',
    
    -- Timing
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    timezone VARCHAR(50) DEFAULT 'UTC',
    all_day BOOLEAN DEFAULT FALSE,
    
    -- Status and metadata
    status event_status DEFAULT 'active',
    metadata JSONB DEFAULT '{}',
    
    -- Automation configuration
    automation_config JSONB DEFAULT '{}',
    
    -- Recurrence (if this is a recurring event instance)
    parent_event_id UUID REFERENCES events(id) ON DELETE SET NULL,
    recurrence_instance_date DATE,
    
    -- Tracking
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    created_by UUID REFERENCES users(id),
    
    -- Constraints
    CONSTRAINT valid_time_range CHECK (end_time > start_time),
    CONSTRAINT valid_timezone CHECK (timezone IS NOT NULL)
);

-- Recurring patterns table
CREATE TABLE recurring_patterns (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    
    -- Pattern definition
    recurrence_type recurrence_type NOT NULL,
    interval_value INTEGER DEFAULT 1, -- Every N days/weeks/months
    
    -- Days of week (for weekly recurrence) - 0=Sunday, 1=Monday, etc.
    days_of_week INTEGER[] DEFAULT NULL,
    
    -- Monthly specific
    day_of_month INTEGER, -- 1-31, for monthly on specific day
    week_of_month INTEGER, -- 1-5, for "first Monday" type patterns
    
    -- Boundaries
    start_date DATE NOT NULL,
    end_date DATE,
    max_occurrences INTEGER,
    
    -- Exceptions (dates to skip)
    exception_dates DATE[] DEFAULT '{}',
    
    -- Tracking
    created_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT valid_interval CHECK (interval_value > 0),
    CONSTRAINT valid_end_conditions CHECK (
        end_date IS NOT NULL OR max_occurrences IS NOT NULL
    )
);

-- Event reminders table
CREATE TABLE event_reminders (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    
    -- Reminder configuration
    minutes_before INTEGER NOT NULL,
    notification_type notification_type DEFAULT 'email',
    
    -- Status tracking
    status reminder_status DEFAULT 'pending',
    scheduled_time TIMESTAMPTZ,
    sent_time TIMESTAMPTZ,
    
    -- notification-hub integration
    notification_id VARCHAR(255), -- ID from notification service
    notification_response JSONB,
    
    -- Retry logic
    retry_count INTEGER DEFAULT 0,
    max_retries INTEGER DEFAULT 3,
    next_retry TIMESTAMPTZ,
    
    -- Tracking
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT valid_minutes_before CHECK (minutes_before >= 0)
);

-- Event attendees table (for future multi-user events)
CREATE TABLE event_attendees (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Attendance details
    status VARCHAR(20) DEFAULT 'invited', -- invited, accepted, declined, tentative
    role VARCHAR(20) DEFAULT 'attendee', -- organizer, attendee, optional
    
    -- Response tracking
    responded_at TIMESTAMPTZ,
    response_note TEXT,
    
    -- Tracking
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    
    -- Unique constraint
    UNIQUE(event_id, user_id)
);

-- Calendar permissions table (for shared calendars)
CREATE TABLE calendar_permissions (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    calendar_owner_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    shared_with_user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    
    -- Permission levels
    permission_level VARCHAR(20) DEFAULT 'view', -- view, edit, admin
    
    -- Tracking
    created_at TIMESTAMPTZ DEFAULT NOW(),
    granted_by UUID REFERENCES users(id),
    
    -- Unique constraint
    UNIQUE(calendar_owner_id, shared_with_user_id)
);

-- Audit log for all calendar changes
CREATE TABLE calendar_audit_log (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    
    -- What changed
    table_name VARCHAR(50) NOT NULL,
    record_id UUID NOT NULL,
    action VARCHAR(20) NOT NULL, -- insert, update, delete
    old_values JSONB,
    new_values JSONB,
    
    -- Who and when
    user_id UUID REFERENCES users(id),
    timestamp TIMESTAMPTZ DEFAULT NOW(),
    
    -- Context
    ip_address INET,
    user_agent TEXT,
    api_endpoint VARCHAR(200)
);

-- Indexes for performance
-- Event queries
CREATE INDEX idx_events_user_time ON events(user_id, start_time);
CREATE INDEX idx_events_time_range ON events(start_time, end_time);
CREATE INDEX idx_events_status ON events(status) WHERE status = 'active';
CREATE INDEX idx_events_type ON events(event_type);
CREATE INDEX idx_events_parent ON events(parent_event_id) WHERE parent_event_id IS NOT NULL;

-- Full-text search on events
CREATE INDEX idx_events_search ON events USING GIN(
    to_tsvector('english', coalesce(title, '') || ' ' || coalesce(description, '') || ' ' || coalesce(location, ''))
);

-- Reminder queries
CREATE INDEX idx_reminders_scheduled ON event_reminders(scheduled_time, status) 
    WHERE status = 'pending';
CREATE INDEX idx_reminders_event ON event_reminders(event_id);

-- Recurrence patterns
CREATE INDEX idx_recurring_patterns_event ON recurring_patterns(event_id);
CREATE INDEX idx_recurring_patterns_dates ON recurring_patterns(start_date, end_date);

-- Attendees
CREATE INDEX idx_attendees_user ON event_attendees(user_id);
CREATE INDEX idx_attendees_event ON event_attendees(event_id);

-- User lookups
CREATE INDEX idx_users_auth_id ON users(auth_user_id);
CREATE INDEX idx_users_email ON users(email);

-- Audit log
CREATE INDEX idx_audit_table_record ON calendar_audit_log(table_name, record_id);
CREATE INDEX idx_audit_user_time ON calendar_audit_log(user_id, timestamp);

-- Functions for automatic timestamp updates
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

-- Triggers for updated_at
CREATE TRIGGER update_users_updated_at 
    BEFORE UPDATE ON users 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_events_updated_at 
    BEFORE UPDATE ON events 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_reminders_updated_at 
    BEFORE UPDATE ON event_reminders 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_attendees_updated_at 
    BEFORE UPDATE ON event_attendees 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to create audit log entries
CREATE OR REPLACE FUNCTION create_audit_log()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'DELETE' THEN
        INSERT INTO calendar_audit_log(table_name, record_id, action, old_values)
        VALUES (TG_TABLE_NAME, OLD.id, 'delete', to_jsonb(OLD));
        RETURN OLD;
    ELSIF TG_OP = 'UPDATE' THEN
        INSERT INTO calendar_audit_log(table_name, record_id, action, old_values, new_values)
        VALUES (TG_TABLE_NAME, NEW.id, 'update', to_jsonb(OLD), to_jsonb(NEW));
        RETURN NEW;
    ELSIF TG_OP = 'INSERT' THEN
        INSERT INTO calendar_audit_log(table_name, record_id, action, new_values)
        VALUES (TG_TABLE_NAME, NEW.id, 'insert', to_jsonb(NEW));
        RETURN NEW;
    END IF;
    RETURN NULL;
END;
$$ language 'plpgsql';

-- Audit triggers for key tables
CREATE TRIGGER events_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON events
    FOR EACH ROW EXECUTE FUNCTION create_audit_log();

CREATE TRIGGER reminders_audit_trigger
    AFTER INSERT OR UPDATE OR DELETE ON event_reminders
    FOR EACH ROW EXECUTE FUNCTION create_audit_log();

-- Views for common queries
-- Upcoming events view
CREATE VIEW upcoming_events AS
SELECT 
    e.*,
    u.display_name as owner_name,
    u.email as owner_email,
    COUNT(r.id) as reminder_count,
    COUNT(a.id) as attendee_count
FROM events e
JOIN users u ON e.user_id = u.id
LEFT JOIN event_reminders r ON e.id = r.event_id AND r.status = 'pending'
LEFT JOIN event_attendees a ON e.id = a.event_id
WHERE e.status = 'active' 
    AND e.start_time > NOW()
GROUP BY e.id, u.display_name, u.email
ORDER BY e.start_time;

-- Today's events view
CREATE VIEW todays_events AS
SELECT *
FROM events 
WHERE status = 'active'
    AND DATE(start_time AT TIME ZONE timezone) = CURRENT_DATE
ORDER BY start_time;

-- Overdue reminders view
CREATE VIEW overdue_reminders AS
SELECT 
    r.*,
    e.title as event_title,
    e.start_time as event_start,
    u.email as user_email
FROM event_reminders r
JOIN events e ON r.event_id = e.id
JOIN users u ON e.user_id = u.id
WHERE r.status = 'pending' 
    AND r.scheduled_time < NOW()
    AND r.retry_count < r.max_retries
ORDER BY r.scheduled_time;

-- Grant permissions (adjust as needed for your setup)
-- GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO vrooli;
-- GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO vrooli;
-- GRANT EXECUTE ON ALL FUNCTIONS IN SCHEMA public TO vrooli;

-- Performance notes:
-- 1. Consider partitioning events table by date for large datasets
-- 2. Archive old events and audit logs periodically
-- 3. Monitor index usage and add/remove as needed based on query patterns
-- 4. Use connection pooling in application layer for better performance

COMMIT;