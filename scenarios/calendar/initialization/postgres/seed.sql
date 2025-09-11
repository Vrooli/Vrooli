-- Calendar System Seed Data
-- Version: 1.0.0
-- Description: Optional seed data for development and testing

-- ============================================================================
-- DEVELOPMENT SEED DATA (Optional)
-- ============================================================================

-- Insert test user (for development only)
-- In production, users are created via scenario-authenticator integration
INSERT INTO users (id, auth_user_id, email, display_name, timezone, preferences, created_at) VALUES 
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'f47ac10b-58cc-4372-a567-0e02b2c3d479', 
    'dev@example.com',
    'Development User',
    'UTC',
    '{"theme": "light", "default_event_duration": 60, "reminder_preferences": {"default_minutes": 15, "types": ["email"]}}',
    NOW() - INTERVAL '1 day'
) ON CONFLICT (auth_user_id) DO NOTHING;

-- Insert sample events for testing
INSERT INTO events (
    id, 
    user_id, 
    title, 
    description, 
    start_time, 
    end_time, 
    timezone, 
    location, 
    event_type, 
    status,
    metadata,
    automation_config
) VALUES 
-- Today's events
(
    'e47ac10b-58cc-4372-a567-0e02b2c3d480',
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'Daily Standup',
    'Team synchronization meeting',
    date_trunc('day', NOW()) + INTERVAL '9 hours',
    date_trunc('day', NOW()) + INTERVAL '9 hours 30 minutes',
    'UTC',
    'Conference Room A',
    'meeting',
    'active',
    '{"priority": "high", "attendees": ["john@example.com", "jane@example.com"]}',
    '{}'
),
-- Tomorrow's events  
(
    'e47ac10b-58cc-4372-a567-0e02b2c3d481',
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'Project Review',
    'Quarterly project status review',
    date_trunc('day', NOW()) + INTERVAL '1 day 14 hours',
    date_trunc('day', NOW()) + INTERVAL '1 day 15 hours',
    'UTC',
    'Virtual Meeting',
    'meeting',
    'active',
    '{"priority": "medium", "project_id": "proj-123"}',
    '{}'
),
-- Next week's event
(
    'e47ac10b-58cc-4372-a567-0e02b2c3d482',
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'Lunch with Sarah',
    'Catch up over lunch',
    date_trunc('week', NOW()) + INTERVAL '1 week 12 hours',
    date_trunc('week', NOW()) + INTERVAL '1 week 13 hours',
    'UTC',
    'Downtown Cafe',
    'personal',
    'active',
    '{"type": "social", "contact": "sarah@example.com"}',
    '{}'
),
-- Recurring event parent
(
    'e47ac10b-58cc-4372-a567-0e02b2c3d483',
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'Weekly Team Meeting',
    'Weekly team synchronization and planning',
    date_trunc('week', NOW()) + INTERVAL '1 day 15 hours', -- Every Tuesday 3 PM
    date_trunc('week', NOW()) + INTERVAL '1 day 16 hours',
    'UTC',
    'Conference Room B',
    'meeting',
    'active',
    '{"recurring": true, "meeting_series": "team-weekly"}',
    '{}'
) ON CONFLICT (id) DO NOTHING;

-- Insert recurring pattern for weekly team meeting
INSERT INTO recurring_patterns (
    id,
    parent_event_id,
    pattern_type,
    interval_value,
    days_of_week,
    end_date,
    max_occurrences
) VALUES (
    'r47ac10b-58cc-4372-a567-0e02b2c3d483',
    'e47ac10b-58cc-4372-a567-0e02b2c3d483',
    'weekly',
    1,
    ARRAY[2], -- Tuesday (0=Sunday, 1=Monday, 2=Tuesday, etc.)
    NOW() + INTERVAL '6 months',
    26 -- ~6 months of weekly meetings
) ON CONFLICT (parent_event_id) DO NOTHING;

-- Insert sample reminders
INSERT INTO event_reminders (
    id,
    event_id,
    minutes_before,
    notification_type,
    scheduled_time,
    status
) VALUES 
-- Reminder for daily standup
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d480',
    'e47ac10b-58cc-4372-a567-0e02b2c3d480',
    15,
    'email',
    (date_trunc('day', NOW()) + INTERVAL '9 hours') - INTERVAL '15 minutes',
    CASE 
        WHEN (date_trunc('day', NOW()) + INTERVAL '9 hours') - INTERVAL '15 minutes' < NOW()
        THEN 'sent'
        ELSE 'pending'
    END
),
-- Reminder for project review
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d481',
    'e47ac10b-58cc-4372-a567-0e02b2c3d481',
    30,
    'email',
    (date_trunc('day', NOW()) + INTERVAL '1 day 14 hours') - INTERVAL '30 minutes',
    'pending'
),
-- Reminder for lunch with Sarah
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d482',
    'e47ac10b-58cc-4372-a567-0e02b2c3d482',
    60,
    'email',
    (date_trunc('week', NOW()) + INTERVAL '1 week 12 hours') - INTERVAL '1 hour',
    'pending'
) ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- VERIFICATION AND STATISTICS
-- ============================================================================

-- Show seed data statistics
DO $$
DECLARE
    user_count INTEGER;
    event_count INTEGER;
    reminder_count INTEGER;
    pattern_count INTEGER;
BEGIN
    SELECT count(*) INTO user_count FROM users;
    SELECT count(*) INTO event_count FROM events;
    SELECT count(*) INTO reminder_count FROM event_reminders;
    SELECT count(*) INTO pattern_count FROM recurring_patterns;
    
    RAISE NOTICE 'Calendar seed data loaded:';
    RAISE NOTICE '  - Users: %', user_count;
    RAISE NOTICE '  - Events: %', event_count;
    RAISE NOTICE '  - Reminders: %', reminder_count;
    RAISE NOTICE '  - Recurring patterns: %', pattern_count;
    
    -- Show upcoming events
    RAISE NOTICE '';
    RAISE NOTICE 'Sample upcoming events:';
    FOR rec IN 
        SELECT title, start_time, event_type 
        FROM events 
        WHERE start_time > NOW() 
        ORDER BY start_time 
        LIMIT 3
    LOOP
        RAISE NOTICE '  - % (%) at %', rec.title, rec.event_type, rec.start_time;
    END LOOP;
END;
$$;

-- ============================================================================
-- DEVELOPMENT HELPERS
-- ============================================================================

-- Function to create test events for a date range
CREATE OR REPLACE FUNCTION create_test_events(
    test_user_id UUID,
    start_date DATE,
    days INTEGER DEFAULT 7
) RETURNS INTEGER AS $$
DECLARE
    counter INTEGER := 0;
    current_date DATE := start_date;
    event_id UUID;
    event_start TIMESTAMPTZ;
    event_end TIMESTAMPTZ;
BEGIN
    WHILE counter < days LOOP
        -- Create morning event
        event_id := uuid_generate_v4();
        event_start := current_date + TIME '09:00:00';
        event_end := current_date + TIME '10:00:00';
        
        INSERT INTO events (
            id, user_id, title, start_time, end_time, event_type, status
        ) VALUES (
            event_id,
            test_user_id,
            'Morning Meeting ' || (counter + 1),
            event_start,
            event_end,
            'meeting',
            'active'
        );
        
        -- Create afternoon event
        event_id := uuid_generate_v4();
        event_start := current_date + TIME '14:00:00';
        event_end := current_date + TIME '15:30:00';
        
        INSERT INTO events (
            id, user_id, title, start_time, end_time, event_type, status
        ) VALUES (
            event_id,
            test_user_id,
            'Afternoon Workshop ' || (counter + 1),
            event_start,
            event_end,
            'meeting',
            'active'
        );
        
        current_date := current_date + INTERVAL '1 day';
        counter := counter + 1;
    END LOOP;
    
    RETURN counter * 2; -- Return total events created
END;
$$ LANGUAGE plpgsql;

-- Function to clean up test data
CREATE OR REPLACE FUNCTION cleanup_test_data() RETURNS VOID AS $$
BEGIN
    DELETE FROM event_reminders 
    WHERE event_id IN (
        SELECT id FROM events WHERE user_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d479'
    );
    
    DELETE FROM recurring_patterns
    WHERE parent_event_id IN (
        SELECT id FROM events WHERE user_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d479'
    );
    
    DELETE FROM events WHERE user_id = 'f47ac10b-58cc-4372-a567-0e02b2c3d479';
    DELETE FROM users WHERE id = 'f47ac10b-58cc-4372-a567-0e02b2c3d479';
    
    RAISE NOTICE 'Test data cleaned up successfully';
END;
$$ LANGUAGE plpgsql;

-- Note: Run cleanup_test_data() to remove all seed data when moving to production