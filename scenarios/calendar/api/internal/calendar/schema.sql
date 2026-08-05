
-- Additional Calendar System Tables
-- Version: 1.1.0
-- Description: Tables for attendance tracking, RSVP, templates, and categories

-- ============================================================================
-- EVENT CATEGORIES TABLE
-- ============================================================================
-- Stores event categories for organization and filtering
CREATE TABLE IF NOT EXISTS event_categories (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(50) NOT NULL UNIQUE,
    description TEXT,
    color VARCHAR(7) DEFAULT '#4285F4', -- Hex color code
    icon VARCHAR(50), -- Icon identifier
    is_system BOOLEAN DEFAULT FALSE,
    display_order INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Insert default system categories
INSERT INTO event_categories (name, description, color, icon, is_system, display_order) VALUES
('meeting', 'Team meetings and discussions', '#4285F4', 'users', TRUE, 1),
('appointment', 'Appointments and consultations', '#0F9D58', 'calendar', TRUE, 2),
('task', 'Tasks and deadlines', '#F4B400', 'check-square', TRUE, 3),
('personal', 'Personal events and activities', '#DB4437', 'user', TRUE, 4),
('travel', 'Travel and transportation', '#9C27B0', 'plane', TRUE, 5),
('reminder', 'Reminders and notifications', '#00ACC1', 'bell', TRUE, 6),
('focus', 'Focus time and deep work', '#FF6F00', 'brain', TRUE, 7),
('social', 'Social events and gatherings', '#E91E63', 'heart', TRUE, 8)
ON CONFLICT (name) DO NOTHING;

-- Indexes for categories
CREATE INDEX IF NOT EXISTS idx_categories_name ON event_categories (name);
CREATE INDEX IF NOT EXISTS idx_categories_is_system ON event_categories (is_system);
CREATE INDEX IF NOT EXISTS idx_categories_display_order ON event_categories (display_order);

-- ============================================================================
-- EVENT TEMPLATES TABLE
-- ============================================================================
-- Stores reusable event templates
CREATE TABLE IF NOT EXISTS event_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255), -- Can be NULL for system templates
    name VARCHAR(255) NOT NULL,
    description TEXT,
    template_data JSONB NOT NULL DEFAULT '{}',
    category VARCHAR(50),
    is_system BOOLEAN DEFAULT FALSE,
    use_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT chk_template_system_user CHECK (
        (is_system = TRUE AND user_id IS NULL) OR 
        (is_system = FALSE AND user_id IS NOT NULL)
    )
);

-- Insert default system templates
INSERT INTO event_templates (name, description, template_data, category, is_system) VALUES
('Weekly Team Meeting', 'Regular weekly team sync', 
 '{"title": "Weekly Team Meeting", "description": "Weekly team sync to discuss progress and blockers", "duration_minutes": 60, "event_type": "meeting", "location": "Conference Room A"}',
 'meeting', TRUE),
('1:1 Meeting', 'One-on-one meeting template',
 '{"title": "1:1 Meeting", "description": "One-on-one discussion", "duration_minutes": 30, "event_type": "meeting"}',
 'meeting', TRUE),
('Daily Standup', 'Daily standup meeting',
 '{"title": "Daily Standup", "description": "Quick daily sync", "duration_minutes": 15, "event_type": "meeting"}',
 'meeting', TRUE),
('Client Call', 'Client consultation call',
 '{"title": "Client Call", "description": "Client consultation", "duration_minutes": 45, "event_type": "appointment", "location": "Virtual"}',
 'appointment', TRUE),
('Focus Time', 'Deep work session',
 '{"title": "Focus Time", "description": "Deep work - no interruptions", "duration_minutes": 120, "event_type": "focus"}',
 'focus', TRUE),
('Lunch Break', 'Lunch break',
 '{"title": "Lunch Break", "description": "Lunch time", "duration_minutes": 60, "event_type": "personal"}',
 'personal', TRUE)
ON CONFLICT DO NOTHING;

-- Indexes for templates
CREATE INDEX IF NOT EXISTS idx_templates_user_id ON event_templates (user_id);
CREATE INDEX IF NOT EXISTS idx_templates_category ON event_templates (category);
CREATE INDEX IF NOT EXISTS idx_templates_is_system ON event_templates (is_system);
CREATE INDEX IF NOT EXISTS idx_templates_use_count ON event_templates (use_count DESC);

-- ============================================================================
-- EVENT ATTENDEES TABLE
-- ============================================================================
-- Stores event attendees, RSVP status, and attendance tracking
CREATE TABLE IF NOT EXISTS event_attendees (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL,
    user_id VARCHAR(255) NOT NULL, -- User ID from auth service
    name VARCHAR(255),
    email VARCHAR(255),
    rsvp_status VARCHAR(20) DEFAULT 'pending',
    rsvp_message TEXT,
    response_time TIMESTAMPTZ,
    attendance_status VARCHAR(20),
    check_in_time TIMESTAMPTZ,
    check_in_method VARCHAR(20), -- manual, qr_code, auto
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Ensure unique attendee per event
    CONSTRAINT uq_event_attendee UNIQUE (event_id, user_id),
    
    -- Valid RSVP statuses
    CONSTRAINT chk_rsvp_status CHECK (
        rsvp_status IN ('pending', 'accepted', 'declined', 'tentative')
    ),
    
    -- Valid attendance statuses
    CONSTRAINT chk_attendance_status CHECK (
        attendance_status IS NULL OR 
        attendance_status IN ('attended', 'no_show', 'partial', 'excused')
    ),
    
    -- Valid check-in methods
    CONSTRAINT chk_checkin_method CHECK (
        check_in_method IS NULL OR
        check_in_method IN ('manual', 'qr_code', 'auto', 'proximity')
    )
);

-- Indexes for attendees
CREATE INDEX IF NOT EXISTS idx_attendees_event_id ON event_attendees (event_id);
CREATE INDEX IF NOT EXISTS idx_attendees_user_id ON event_attendees (user_id);
CREATE INDEX IF NOT EXISTS idx_attendees_rsvp_status ON event_attendees (rsvp_status);
CREATE INDEX IF NOT EXISTS idx_attendees_attendance_status ON event_attendees (attendance_status);
CREATE INDEX IF NOT EXISTS idx_attendees_response_time ON event_attendees (response_time);

-- Composite indexes for common queries
CREATE INDEX IF NOT EXISTS idx_attendees_event_rsvp ON event_attendees (event_id, rsvp_status);
CREATE INDEX IF NOT EXISTS idx_attendees_event_attendance ON event_attendees (event_id, attendance_status);

-- ============================================================================
-- TRIGGERS
-- ============================================================================

-- Add update triggers for new tables
DROP TRIGGER IF EXISTS tr_categories_updated_at ON event_categories;
CREATE TRIGGER tr_categories_updated_at 
    BEFORE UPDATE ON event_categories 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS tr_templates_updated_at ON event_templates;
CREATE TRIGGER tr_templates_updated_at 
    BEFORE UPDATE ON event_templates 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS tr_attendees_updated_at ON event_attendees;
CREATE TRIGGER tr_attendees_updated_at 
    BEFORE UPDATE ON event_attendees 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- USEFUL VIEWS
-- ============================================================================

-- View for event attendance summary
CREATE OR REPLACE VIEW event_attendance_summary AS
SELECT 
    e.id as event_id,
    e.title as event_title,
    e.start_time,
    COUNT(DISTINCT a.user_id) as total_invited,
    COUNT(DISTINCT CASE WHEN a.rsvp_status = 'accepted' THEN a.user_id END) as total_accepted,
    COUNT(DISTINCT CASE WHEN a.rsvp_status = 'declined' THEN a.user_id END) as total_declined,
    COUNT(DISTINCT CASE WHEN a.rsvp_status = 'tentative' THEN a.user_id END) as total_tentative,
    COUNT(DISTINCT CASE WHEN a.attendance_status = 'attended' THEN a.user_id END) as total_attended,
    COUNT(DISTINCT CASE WHEN a.attendance_status = 'no_show' THEN a.user_id END) as total_no_show,
    ROUND(100.0 * COUNT(DISTINCT CASE WHEN a.attendance_status = 'attended' THEN a.user_id END) / 
          NULLIF(COUNT(DISTINCT CASE WHEN a.rsvp_status = 'accepted' THEN a.user_id END), 0), 2) as attendance_rate
FROM events e
LEFT JOIN event_attendees a ON e.id = a.event_id
GROUP BY e.id, e.title, e.start_time;

-- View for template usage statistics
CREATE OR REPLACE VIEW template_usage_stats AS
SELECT 
    t.id,
    t.name,
    t.category,
    t.is_system,
    t.use_count,
    COUNT(e.id) as events_created,
    MAX(e.created_at) as last_used_at
FROM event_templates t
LEFT JOIN events e ON e.metadata->>'from_template' = t.id::TEXT
GROUP BY t.id, t.name, t.category, t.is_system, t.use_count
ORDER BY t.use_count DESC;

-- ============================================================================
-- VALIDATION
-- ============================================================================

DO $$
DECLARE
    table_count INTEGER;
BEGIN
    SELECT count(*) INTO table_count 
    FROM information_schema.tables 
    WHERE table_schema = 'public' 
    AND table_name IN ('event_categories', 'event_templates', 'event_attendees');
    
    RAISE NOTICE 'Additional tables installation:';
    RAISE NOTICE '  - Tables created: %', table_count;
    
    IF table_count = 3 THEN
        RAISE NOTICE '✅ Additional schema installation successful!';
    ELSE
        RAISE WARNING '⚠️  Additional schema installation may be incomplete';
    END IF;
END;
$$;
-- Event Templates Table
-- Stores reusable templates for common meeting types

CREATE TABLE IF NOT EXISTS event_templates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    template_data JSONB NOT NULL DEFAULT '{}'::jsonb,
    category VARCHAR(50),
    is_system BOOLEAN DEFAULT FALSE,
    use_count INTEGER DEFAULT 0,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    CONSTRAINT unique_user_template_name UNIQUE(user_id, name)
);

-- Index for faster lookups
CREATE INDEX IF NOT EXISTS idx_event_templates_user_id ON event_templates(user_id);
CREATE INDEX IF NOT EXISTS idx_event_templates_category ON event_templates(category);
CREATE INDEX IF NOT EXISTS idx_event_templates_system ON event_templates(is_system);

-- Sample system templates for all users
INSERT INTO event_templates (user_id, name, description, template_data, category, is_system)
VALUES 
    ('00000000-0000-0000-0000-000000000000', 'Daily Standup', 'Quick team sync meeting',
     '{"title": "Daily Standup", "duration_minutes": 15, "event_type": "meeting", "description": "Daily team sync to discuss progress and blockers", "default_time": "09:00", "recurrence": "daily"}'::jsonb,
     'meeting', true),
     
    ('00000000-0000-0000-0000-000000000000', 'Weekly One-on-One', 'Manager check-in meeting',
     '{"title": "1:1 Meeting", "duration_minutes": 30, "event_type": "meeting", "description": "Weekly one-on-one discussion", "default_time": "14:00", "recurrence": "weekly"}'::jsonb,
     'meeting', true),
     
    ('00000000-0000-0000-0000-000000000000', 'Sprint Planning', 'Sprint planning session',
     '{"title": "Sprint Planning", "duration_minutes": 120, "event_type": "meeting", "description": "Plan work for upcoming sprint", "default_time": "10:00", "recurrence": "biweekly"}'::jsonb,
     'meeting', true),
     
    ('00000000-0000-0000-0000-000000000000', 'Retrospective', 'Sprint retrospective meeting',
     '{"title": "Sprint Retrospective", "duration_minutes": 60, "event_type": "meeting", "description": "Reflect on sprint and identify improvements", "default_time": "16:00", "recurrence": "biweekly"}'::jsonb,
     'meeting', true),
     
    ('00000000-0000-0000-0000-000000000000', 'Client Call', 'External client meeting',
     '{"title": "Client Call", "duration_minutes": 60, "event_type": "appointment", "description": "Meeting with client", "default_time": "14:00", "location": "Zoom"}'::jsonb,
     'appointment', true),
     
    ('00000000-0000-0000-0000-000000000000', 'Focus Time', 'Dedicated focus work block',
     '{"title": "Focus Time", "duration_minutes": 120, "event_type": "focus", "description": "Deep work - no interruptions", "default_time": "09:00"}'::jsonb,
     'focus', true),
     
    ('00000000-0000-0000-0000-000000000000', 'Lunch Break', 'Lunch break',
     '{"title": "Lunch Break", "duration_minutes": 60, "event_type": "personal", "description": "Lunch time", "default_time": "12:00", "recurrence": "daily"}'::jsonb,
     'personal', true),
     
    ('00000000-0000-0000-0000-000000000000', 'Team Meeting', 'All-hands team meeting',
     '{"title": "Team Meeting", "duration_minutes": 45, "event_type": "meeting", "description": "Full team meeting", "default_time": "15:00", "recurrence": "weekly"}'::jsonb,
     'meeting', true)
ON CONFLICT (user_id, name) DO NOTHING;
-- External calendar integration tables
-- Stores OAuth tokens and sync configuration for Google Calendar and Outlook

-- External calendar connections
CREATE TABLE IF NOT EXISTS external_calendars (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL CHECK (provider IN ('google', 'outlook')),
    access_token TEXT NOT NULL,
    refresh_token TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    calendar_id VARCHAR(255),
    sync_enabled BOOLEAN DEFAULT true,
    sync_direction VARCHAR(20) DEFAULT 'bidirectional' CHECK (sync_direction IN ('bidirectional', 'import_only', 'export_only')),
    last_sync_time TIMESTAMPTZ,
    sync_metadata JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE (user_id, provider)
);

-- OAuth state tokens for verification
CREATE TABLE IF NOT EXISTS oauth_states (
    state VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL
);

-- External sync log for tracking sync history
CREATE TABLE IF NOT EXISTS external_sync_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    calendar_id UUID REFERENCES external_calendars(id) ON DELETE CASCADE,
    sync_type VARCHAR(20) CHECK (sync_type IN ('manual', 'scheduled', 'webhook')),
    direction VARCHAR(20) CHECK (direction IN ('import', 'export', 'bidirectional')),
    events_created INTEGER DEFAULT 0,
    events_updated INTEGER DEFAULT 0,
    events_deleted INTEGER DEFAULT 0,
    errors_count INTEGER DEFAULT 0,
    error_details JSONB,
    started_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    status VARCHAR(20) DEFAULT 'in_progress' CHECK (status IN ('in_progress', 'success', 'partial', 'failed'))
);

-- External event mapping to track synced events
CREATE TABLE IF NOT EXISTS external_event_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    local_event_id UUID REFERENCES events(id) ON DELETE CASCADE,
    external_id VARCHAR(255) NOT NULL,
    provider VARCHAR(50) NOT NULL,
    external_metadata JSONB DEFAULT '{}',
    last_synced_at TIMESTAMPTZ DEFAULT NOW(),
    sync_hash VARCHAR(64), -- To detect changes
    UNIQUE (external_id, provider)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_external_calendars_user_id ON external_calendars(user_id);
CREATE INDEX IF NOT EXISTS idx_external_calendars_sync ON external_calendars(sync_enabled, last_sync_time);
CREATE INDEX IF NOT EXISTS idx_oauth_states_expires ON oauth_states(expires_at);
CREATE INDEX IF NOT EXISTS idx_sync_log_calendar ON external_sync_log(calendar_id, started_at DESC);
CREATE INDEX IF NOT EXISTS idx_external_mappings_local ON external_event_mappings(local_event_id);
CREATE INDEX IF NOT EXISTS idx_external_mappings_external ON external_event_mappings(external_id, provider);

-- Cleanup expired OAuth states
CREATE OR REPLACE FUNCTION cleanup_expired_oauth_states() RETURNS void AS $$
BEGIN
    DELETE FROM oauth_states WHERE expires_at < NOW();
END;
$$ LANGUAGE plpgsql;

-- Add unique constraint on events metadata for external IDs
-- This prevents duplicate imports

-- Calendar System Resource Management Schema Extension
-- Version: 1.1.0
-- Description: Adds resource booking capabilities to prevent double-booking
-- Compatible with PostgreSQL 12+

-- ============================================================================
-- RESOURCES TABLE
-- ============================================================================
-- Stores bookable resources (meeting rooms, equipment, vehicles, etc.)
CREATE TABLE IF NOT EXISTS resources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL,
    resource_type VARCHAR(50) NOT NULL DEFAULT 'room',
    description TEXT,
    location VARCHAR(500),
    capacity INTEGER,
    metadata JSONB DEFAULT '{}',
    availability_rules JSONB DEFAULT '{}', -- Opening hours, maintenance windows, etc.
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT chk_resources_type CHECK (resource_type IN ('room', 'equipment', 'vehicle', 'person', 'virtual', 'other')),
    CONSTRAINT chk_resources_status CHECK (status IN ('active', 'inactive', 'maintenance'))
);

-- Indexes for resources table
CREATE INDEX IF NOT EXISTS idx_resources_type ON resources (resource_type);
CREATE INDEX IF NOT EXISTS idx_resources_status ON resources (status);
CREATE INDEX IF NOT EXISTS idx_resources_name ON resources (name);
CREATE INDEX IF NOT EXISTS idx_resources_metadata ON resources USING gin (metadata);

-- ============================================================================
-- EVENT_RESOURCES TABLE
-- ============================================================================
-- Links events to resources they've booked (many-to-many relationship)
CREATE TABLE IF NOT EXISTS event_resources (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    booking_status VARCHAR(20) NOT NULL DEFAULT 'confirmed',
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Prevent duplicate bookings
    CONSTRAINT uq_event_resource UNIQUE (event_id, resource_id),
    CONSTRAINT chk_booking_status CHECK (booking_status IN ('pending', 'confirmed', 'cancelled'))
);

-- Indexes for event_resources table
CREATE INDEX IF NOT EXISTS idx_event_resources_event_id ON event_resources (event_id);
CREATE INDEX IF NOT EXISTS idx_event_resources_resource_id ON event_resources (resource_id);
CREATE INDEX IF NOT EXISTS idx_event_resources_status ON event_resources (booking_status);

-- ============================================================================
-- RESOURCE_AVAILABILITY TABLE
-- ============================================================================
-- Tracks resource availability exceptions (holidays, maintenance, special schedules)
CREATE TABLE IF NOT EXISTS resource_availability (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    resource_id UUID NOT NULL REFERENCES resources(id) ON DELETE CASCADE,
    availability_type VARCHAR(30) NOT NULL DEFAULT 'unavailable',
    start_time TIMESTAMPTZ NOT NULL,
    end_time TIMESTAMPTZ NOT NULL,
    reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    -- Constraints
    CONSTRAINT chk_availability_time_order CHECK (start_time < end_time),
    CONSTRAINT chk_availability_type CHECK (availability_type IN ('unavailable', 'limited', 'special'))
);

-- Indexes for resource_availability table
CREATE INDEX IF NOT EXISTS idx_resource_availability_resource_id ON resource_availability (resource_id);
CREATE INDEX IF NOT EXISTS idx_resource_availability_time ON resource_availability (start_time, end_time);

-- ============================================================================
-- HELPER FUNCTIONS
-- ============================================================================

-- Function to check if a resource is available for a given time period
CREATE OR REPLACE FUNCTION is_resource_available(
    p_resource_id UUID,
    p_start_time TIMESTAMPTZ,
    p_end_time TIMESTAMPTZ,
    p_exclude_event_id UUID DEFAULT NULL
) RETURNS BOOLEAN AS $$
DECLARE
    conflict_count INTEGER;
BEGIN
    -- Check for conflicts with existing bookings
    SELECT COUNT(*)
    INTO conflict_count
    FROM event_resources er
    JOIN events e ON er.event_id = e.id
    WHERE er.resource_id = p_resource_id
      AND er.booking_status = 'confirmed'
      AND e.status = 'active'
      AND (p_exclude_event_id IS NULL OR e.id != p_exclude_event_id)
      AND (
          (e.start_time >= p_start_time AND e.start_time < p_end_time) OR
          (e.end_time > p_start_time AND e.end_time <= p_end_time) OR
          (e.start_time <= p_start_time AND e.end_time >= p_end_time)
      );
    
    IF conflict_count > 0 THEN
        RETURN FALSE;
    END IF;
    
    -- Check for conflicts with availability exceptions
    SELECT COUNT(*)
    INTO conflict_count
    FROM resource_availability ra
    WHERE ra.resource_id = p_resource_id
      AND ra.availability_type = 'unavailable'
      AND (
          (ra.start_time >= p_start_time AND ra.start_time < p_end_time) OR
          (ra.end_time > p_start_time AND ra.end_time <= p_end_time) OR
          (ra.start_time <= p_start_time AND ra.end_time >= p_end_time)
      );
    
    RETURN conflict_count = 0;
END;
$$ LANGUAGE plpgsql;

-- Function to get conflicting bookings for a resource
CREATE OR REPLACE FUNCTION get_resource_conflicts(
    p_resource_id UUID,
    p_start_time TIMESTAMPTZ,
    p_end_time TIMESTAMPTZ
) RETURNS TABLE(
    event_id UUID,
    event_title VARCHAR(255),
    start_time TIMESTAMPTZ,
    end_time TIMESTAMPTZ,
    conflict_type VARCHAR(20)
) AS $$
BEGIN
    RETURN QUERY
    -- Get booking conflicts
    SELECT 
        e.id as event_id,
        e.title as event_title,
        e.start_time,
        e.end_time,
        'booking'::VARCHAR(20) as conflict_type
    FROM event_resources er
    JOIN events e ON er.event_id = e.id
    WHERE er.resource_id = p_resource_id
      AND er.booking_status = 'confirmed'
      AND e.status = 'active'
      AND (
          (e.start_time >= p_start_time AND e.start_time < p_end_time) OR
          (e.end_time > p_start_time AND e.end_time <= p_end_time) OR
          (e.start_time <= p_start_time AND e.end_time >= p_end_time)
      )
    UNION ALL
    -- Get availability conflicts
    SELECT 
        ra.id as event_id,
        ra.reason as event_title,
        ra.start_time,
        ra.end_time,
        'availability'::VARCHAR(20) as conflict_type
    FROM resource_availability ra
    WHERE ra.resource_id = p_resource_id
      AND ra.availability_type = 'unavailable'
      AND (
          (ra.start_time >= p_start_time AND ra.start_time < p_end_time) OR
          (ra.end_time > p_start_time AND ra.end_time <= p_end_time) OR
          (ra.start_time <= p_start_time AND ra.end_time >= p_end_time)
      );
END;
$$ LANGUAGE plpgsql;

-- ============================================================================
-- SEED DATA
-- ============================================================================

-- Insert sample resources
INSERT INTO resources (name, resource_type, description, location, capacity, metadata) VALUES
    ('Conference Room A', 'room', 'Large conference room with video conferencing', 'Building 1, Floor 2', 20, '{"amenities": ["projector", "whiteboard", "video_conference"]}'),
    ('Conference Room B', 'room', 'Medium meeting room', 'Building 1, Floor 2', 10, '{"amenities": ["projector", "whiteboard"]}'),
    ('Meeting Pod 1', 'room', 'Small private meeting space', 'Building 1, Floor 1', 4, '{"amenities": ["tv_screen", "whiteboard"]}'),
    ('Projector 1', 'equipment', 'Portable HD projector', 'IT Storage', NULL, '{"model": "Epson EB-2250U", "resolution": "1920x1200"}'),
    ('Company Car 1', 'vehicle', 'Toyota Camry - Blue', 'Parking Lot A', 5, '{"license_plate": "ABC-123", "fuel_type": "hybrid"}')
ON CONFLICT DO NOTHING;

-- Add update trigger for updated_at columns
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers for updated_at
DROP TRIGGER IF EXISTS update_resources_updated_at ON resources;
CREATE TRIGGER update_resources_updated_at
    BEFORE UPDATE ON resources
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
-- Calendar System Database Schema
-- Version: 1.0.0
-- Description: Complete schema for calendar events, users, and scheduling functionality
-- Compatible with PostgreSQL 12+

-- Enable required extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";  -- For full-text search
CREATE EXTENSION IF NOT EXISTS "btree_gin"; -- For composite indexes

-- Drop existing tables in dependency order (for development/testing)
-- ============================================================================
-- USERS TABLE
-- ============================================================================
-- Stores user profile information and preferences
CREATE TABLE IF NOT EXISTS users (
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
CREATE INDEX IF NOT EXISTS idx_users_auth_user_id ON users (auth_user_id);
CREATE INDEX IF NOT EXISTS idx_users_email ON users (email);
CREATE INDEX IF NOT EXISTS idx_users_created_at ON users (created_at);

-- ============================================================================
-- EVENTS TABLE  
-- ============================================================================
-- Core events table with all calendar event information
CREATE TABLE IF NOT EXISTS events (
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

-- Full-text search index
CREATE INDEX IF NOT EXISTS idx_events_title_search ON events USING gin (to_tsvector('english', title));
CREATE INDEX IF NOT EXISTS idx_events_description_search ON events USING gin (to_tsvector('english', coalesce(description, '')));

-- JSONB indexes for metadata queries
CREATE INDEX IF NOT EXISTS idx_events_metadata ON events USING gin (metadata);
CREATE INDEX IF NOT EXISTS idx_events_automation ON events USING gin (automation_config);

-- ============================================================================
-- RECURRING PATTERNS TABLE
-- ============================================================================
-- Defines recurring event patterns and rules
CREATE TABLE IF NOT EXISTS recurring_patterns (
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
CREATE INDEX IF NOT EXISTS idx_recurring_parent_event ON recurring_patterns (parent_event_id);
CREATE INDEX IF NOT EXISTS idx_recurring_pattern_type ON recurring_patterns (pattern_type);
CREATE INDEX IF NOT EXISTS idx_recurring_end_date ON recurring_patterns (end_date);

-- ============================================================================
-- EVENT REMINDERS TABLE
-- ============================================================================
-- Stores reminder configurations and delivery status
CREATE TABLE IF NOT EXISTS event_reminders (
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
CREATE INDEX IF NOT EXISTS idx_reminders_event_id ON event_reminders (event_id);
CREATE INDEX IF NOT EXISTS idx_reminders_scheduled_time ON event_reminders (scheduled_time);
CREATE INDEX IF NOT EXISTS idx_reminders_status ON event_reminders (status);
CREATE INDEX IF NOT EXISTS idx_reminders_type ON event_reminders (notification_type);

-- Index for processing pending reminders
CREATE INDEX IF NOT EXISTS idx_reminders_pending ON event_reminders (scheduled_time, status) 
WHERE status = 'pending';

-- ============================================================================
-- EVENT EMBEDDINGS TABLE (for AI search)
-- ============================================================================
-- Stores vector embeddings for semantic search using Qdrant
-- This table stores references and metadata; actual vectors are in Qdrant
CREATE TABLE IF NOT EXISTS event_embeddings (
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
CREATE INDEX IF NOT EXISTS idx_embeddings_event_id ON event_embeddings (event_id);
CREATE INDEX IF NOT EXISTS idx_embeddings_qdrant_id ON event_embeddings (qdrant_point_id);
CREATE INDEX IF NOT EXISTS idx_embeddings_content_hash ON event_embeddings (content_hash);
CREATE INDEX IF NOT EXISTS idx_embeddings_keywords ON event_embeddings USING gin (keywords);

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

-- View for pending reminders ready to be sent
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
