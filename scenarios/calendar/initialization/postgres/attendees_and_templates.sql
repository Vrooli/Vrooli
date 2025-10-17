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
CREATE TRIGGER tr_categories_updated_at 
    BEFORE UPDATE ON event_categories 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER tr_templates_updated_at 
    BEFORE UPDATE ON event_templates 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

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