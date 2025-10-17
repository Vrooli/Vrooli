-- Calendar System Seed Data
-- Version: 1.0.0
-- Description: Optional seed data for development and testing

-- ============================================================================
-- DEVELOPMENT SEED DATA (Optional)
-- ============================================================================

-- Insert test users (for development only)
-- In production, users are created via scenario-authenticator integration
INSERT INTO users (id, auth_user_id, email, display_name, timezone, preferences, created_at) VALUES 
-- Primary development user
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'f47ac10b-58cc-4372-a567-0e02b2c3d479', 
    'dev@example.com',
    'Alex Thompson',
    'America/New_York',
    '{"theme": "dark", "default_event_duration": 60, "reminder_preferences": {"default_minutes": 15, "types": ["email", "push"]}, "work_hours": {"start": "09:00", "end": "17:00"}, "ai_scheduling": {"enabled": true, "optimization_level": "high"}}',
    NOW() - INTERVAL '2 days'
),
-- Secondary user for collaboration scenarios
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d480',
    'f47ac10b-58cc-4372-a567-0e02b2c3d480',
    'sarah.chen@example.com',
    'Sarah Chen',
    'America/Los_Angeles',
    '{"theme": "light", "default_event_duration": 30, "reminder_preferences": {"default_minutes": 10, "types": ["email"]}, "work_hours": {"start": "08:30", "end": "16:30"}, "ai_scheduling": {"enabled": true, "optimization_level": "medium"}}',
    NOW() - INTERVAL '1 day'
),
-- Manager user for delegation scenarios
(
    'f47ac10b-58cc-4372-a567-0e02b2c3d481',
    'f47ac10b-58cc-4372-a567-0e02b2c3d481',
    'mike.johnson@example.com',
    'Mike Johnson',
    'UTC',
    '{"theme": "auto", "default_event_duration": 45, "reminder_preferences": {"default_minutes": 30, "types": ["email", "sms"]}, "work_hours": {"start": "07:00", "end": "15:00"}, "delegation_enabled": true, "ai_scheduling": {"enabled": true, "optimization_level": "aggressive"}}',
    NOW() - INTERVAL '3 days'
) ON CONFLICT (auth_user_id) DO NOTHING;

-- Insert comprehensive sample events for testing
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
-- Alex Thompson's events (Primary user)
-- Past completed event
(
    'e47ac10b-58cc-4372-a567-0e02b2c3d478',
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'Client Onboarding Call',
    'Initial consultation with new enterprise client - TechCorp Inc.',
    NOW() - INTERVAL '2 hours',
    NOW() - INTERVAL '1 hour',
    'America/New_York',
    'Zoom Meeting Room 1',
    'meeting',
    'completed',
    '{"priority": "high", "client_id": "client-001", "deal_value": 50000, "attendees": ["alex@example.com", "client@techcorp.com"], "outcome": "successful", "follow_up": "send proposal by Friday"}',
    '{"post_meeting_actions": {"send_summary": true, "create_follow_up_tasks": true, "update_crm": true}, "ai_analysis": {"sentiment": "positive", "next_steps": ["proposal_creation", "technical_demo"]}}'
),
-- Current happening event
(
    'e47ac10b-58cc-4372-a567-0e02b2c3d479',
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'Daily Standup',
    'Team synchronization and blocker discussion',
    NOW() - INTERVAL '5 minutes',
    NOW() + INTERVAL '25 minutes',
    'America/New_York',
    'Conference Room A',
    'meeting',
    'active',
    '{"priority": "medium", "attendees": ["alex@example.com", "sarah.chen@example.com", "mike.johnson@example.com"], "sprint": "2024-Q1-3", "recurring": true}',
    '{"auto_summary": true, "action_item_extraction": true, "integration": {"jira_ticket_creation": true, "slack_notification": true}}'
),
-- Today's upcoming appointment
(
    'e47ac10b-58cc-4372-a567-0e02b2c3d480',
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'Doctor Appointment',
    'Annual physical checkup - Dr. Smith',
    date_trunc('day', NOW()) + INTERVAL '14 hours',
    date_trunc('day', NOW()) + INTERVAL '15 hours',
    'America/New_York',
    'Metropolitan Medical Center, 123 Health St',
    'appointment',
    'active',
    '{"priority": "high", "type": "medical", "doctor": "Dr. Smith", "insurance": "covered", "preparation": ["fasting_required"]}',
    '{"reminders": {"sms_1_day": true, "email_2_hours": true, "push_15_min": true}, "travel_time_buffer": 30}'
),
-- Tomorrow's work event
(
    'e47ac10b-58cc-4372-a567-0e02b2c3d481',
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'Q1 Strategy Planning',
    'Quarterly business review and strategy planning session',
    date_trunc('day', NOW()) + INTERVAL '1 day 10 hours',
    date_trunc('day', NOW()) + INTERVAL '1 day 12 hours',
    'America/New_York',
    'Executive Conference Room',
    'meeting',
    'active',
    '{"priority": "high", "meeting_type": "strategic", "attendees": ["alex@example.com", "mike.johnson@example.com", "ceo@example.com"], "preparation_required": true, "materials": ["q4_report.pdf", "market_analysis.xlsx"]}',
    '{"pre_meeting": {"send_agenda": true, "prep_reminder_24h": true}, "during_meeting": {"record_session": true, "live_transcription": true}, "post_meeting": {"distribute_notes": true, "schedule_follow_ups": true}}'
),
-- Personal travel block
(
    'e47ac10b-58cc-4372-a567-0e02b2c3d482',
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'Flight to San Francisco',
    'Business trip - West Coast client meetings',
    date_trunc('day', NOW()) + INTERVAL '3 days 6 hours',
    date_trunc('day', NOW()) + INTERVAL '3 days 12 hours',
    'America/New_York',
    'JFK Airport → SFO Airport',
    'travel',
    'active',
    '{"priority": "high", "flight_number": "AA123", "confirmation": "ABC123", "seat": "4A", "destination": "San Francisco", "purpose": "business"}',
    '{"travel_prep": {"check_in_24h": true, "weather_update": true, "ground_transport": true}, "notifications": {"departure_gate_changes": true, "delay_alerts": true}}'
),

-- Sarah Chen's events (Collaboration scenarios)
(
    'e47ac10b-58cc-4372-a567-0e02b2c3d490',
    'f47ac10b-58cc-4372-a567-0e02b2c3d480',
    'Design Review Session',
    'Review new product mockups and user journey flows',
    date_trunc('day', NOW()) + INTERVAL '2 hours',
    date_trunc('day', NOW()) + INTERVAL '3 hours 30 minutes',
    'America/Los_Angeles',
    'Design Studio',
    'meeting',
    'active',
    '{"priority": "medium", "project": "mobile-app-v2", "attendees": ["sarah.chen@example.com", "design-team@example.com"], "deliverables": ["updated_mockups", "user_feedback"]}',
    '{"ai_insights": {"design_analysis": true, "accessibility_check": true}, "collaboration": {"figma_integration": true, "live_feedback": true}}'
),
(
    'e47ac10b-58cc-4372-a567-0e02b2c3d491',
    'f47ac10b-58cc-4372-a567-0e02b2c3d480',
    'Coffee Chat with Alex',
    'Cross-team collaboration discussion over coffee',
    date_trunc('day', NOW()) + INTERVAL '1 day 16 hours',
    date_trunc('day', NOW()) + INTERVAL '1 day 17 hours',
    'America/Los_Angeles',
    'Campus Cafe',
    'personal',
    'active',
    '{"type": "networking", "colleague": "alex@example.com", "topics": ["project_sync", "career_development"]}',
    '{"mood_tracking": true, "relationship_building": true}'
),

-- Mike Johnson's events (Management scenarios)
(
    'e47ac10b-58cc-4372-a567-0e02b2c3d500',
    'f47ac10b-58cc-4372-a567-0e02b2c3d481',
    'Team Performance Reviews',
    'Individual performance review sessions with team members',
    date_trunc('day', NOW()) + INTERVAL '1 day 9 hours',
    date_trunc('day', NOW()) + INTERVAL '1 day 17 hours',
    'UTC',
    'Manager Office',
    'meeting',
    'active',
    '{"priority": "high", "type": "performance_review", "team_members": ["alex@example.com", "sarah.chen@example.com", "junior@example.com"], "quarter": "Q1-2024"}',
    '{"hr_integration": true, "goal_tracking": true, "calendar_blocking": {"no_interruptions": true}, "documentation": {"review_forms": true, "development_plans": true}}'
),
(
    'e47ac10b-58cc-4372-a567-0e02b2c3d501',
    'f47ac10b-58cc-4372-a567-0e02b2c3d481',
    'Board Meeting Preparation',
    'Prepare quarterly board presentation and financial reports',
    date_trunc('day', NOW()) + INTERVAL '2 days 8 hours',
    date_trunc('day', NOW()) + INTERVAL '2 days 10 hours',
    'UTC',
    'Executive Suite',
    'block',
    'active',
    '{"priority": "critical", "type": "deep_work", "deliverable": "board_presentation", "stakeholders": ["cfo@example.com", "ceo@example.com"]}',
    '{"focus_mode": {"block_notifications": true, "do_not_disturb": true}, "preparation": {"gather_metrics": true, "rehearsal_session": true}}'
),

-- Recurring event parent
(
    'e47ac10b-58cc-4372-a567-0e02b2c3d483',
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'Weekly Team Meeting',
    'Agile sprint planning and team synchronization',
    date_trunc('week', NOW()) + INTERVAL '1 day 15 hours', -- Every Tuesday 3 PM
    date_trunc('week', NOW()) + INTERVAL '1 day 16 hours',
    'America/New_York',
    'Conference Room B',
    'meeting',
    'active',
    '{"recurring": true, "meeting_series": "team-weekly", "attendees": ["alex@example.com", "sarah.chen@example.com", "mike.johnson@example.com"], "agenda_template": "sprint_review"}',
    '{"recurring_automation": {"auto_agenda": true, "preparation_reminders": true, "zoom_room_booking": true}, "agile_integration": {"jira_sync": true, "velocity_tracking": true}}'
),

-- Cancelled event example
(
    'e47ac10b-58cc-4372-a567-0e02b2c3d502',
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'Vendor Meeting - Cancelled',
    'Software vendor presentation - rescheduled due to technical issues',
    date_trunc('day', NOW()) + INTERVAL '1 day 11 hours',
    date_trunc('day', NOW()) + INTERVAL '1 day 12 hours',
    'America/New_York',
    'Virtual Meeting',
    'meeting',
    'cancelled',
    '{"priority": "medium", "cancellation_reason": "technical_issues", "reschedule_requested": true, "vendor": "TechVendor Inc"}',
    '{"cancellation_automation": {"notify_attendees": true, "free_up_time": true, "suggest_alternatives": true}}'
),

-- AI-suggested event
(
    'e47ac10b-58cc-4372-a567-0e02b2c3d503',
    'f47ac10b-58cc-4372-a567-0e02b2c3d479',
    'AI Suggested: Focus Time',
    'AI-optimized deep work block for project completion',
    date_trunc('day', NOW()) + INTERVAL '2 days 14 hours',
    date_trunc('day', NOW()) + INTERVAL '2 days 16 hours',
    'America/New_York',
    'Home Office',
    'block',
    'active',
    '{"ai_generated": true, "confidence": 0.95, "optimization_factors": ["energy_levels", "distraction_minimization", "deadline_proximity"], "project": "Q1-deliverables"}',
    '{"ai_optimization": {"dynamic_scheduling": true, "interruption_blocking": true, "productivity_tracking": true}, "adaptive_reminders": true}'
) ON CONFLICT (id) DO NOTHING;

-- Insert recurring patterns for multiple events
INSERT INTO recurring_patterns (
    id,
    parent_event_id,
    pattern_type,
    interval_value,
    days_of_week,
    end_date,
    max_occurrences,
    exceptions
) VALUES 
-- Weekly team meeting pattern
(
    'r47ac10b-58cc-4372-a567-0e02b2c3d483',
    'e47ac10b-58cc-4372-a567-0e02b2c3d483',
    'weekly',
    1,
    ARRAY[2], -- Tuesday (0=Sunday, 1=Monday, 2=Tuesday, etc.)
    NOW() + INTERVAL '6 months',
    26, -- ~6 months of weekly meetings
    ARRAY[NOW() + INTERVAL '2 weeks', NOW() + INTERVAL '8 weeks'] -- Holiday exceptions
),
-- Daily standup pattern (for demonstration)
(
    'r47ac10b-58cc-4372-a567-0e02b2c3d479',
    'e47ac10b-58cc-4372-a567-0e02b2c3d479',
    'daily',
    1,
    ARRAY[1,2,3,4,5], -- Monday through Friday
    NOW() + INTERVAL '3 months',
    65, -- ~3 months of daily standups
    ARRAY[NOW() + INTERVAL '30 days'] -- Company holiday exception
),
-- Monthly all-hands meeting pattern
(
    'r47ac10b-58cc-4372-a567-0e02b2c3d500',
    'e47ac10b-58cc-4372-a567-0e02b2c3d500',
    'monthly',
    1,
    NULL, -- Not day-of-week based
    NOW() + INTERVAL '1 year',
    12, -- 12 monthly meetings
    NULL
) ON CONFLICT (parent_event_id) DO NOTHING;

-- Insert sample event embeddings for AI/Qdrant integration
INSERT INTO event_embeddings (
    id,
    event_id,
    qdrant_point_id,
    embedding_version,
    content_hash,
    keywords
) VALUES 
-- Client onboarding call embedding
(
    'emb47ac10b-58cc-4372-a567-0e02b2c3d478',
    'e47ac10b-58cc-4372-a567-0e02b2c3d478',
    'qdrant-point-001',
    'v1.0',
    sha256('Client Onboarding Call Initial consultation with new enterprise client - TechCorp Inc.'),
    ARRAY['client', 'onboarding', 'enterprise', 'consultation', 'techcorp', 'sales', 'meeting']
),
-- Daily standup embedding
(
    'emb47ac10b-58cc-4372-a567-0e02b2c3d479',
    'e47ac10b-58cc-4372-a567-0e02b2c3d479',
    'qdrant-point-002',
    'v1.0',
    sha256('Daily Standup Team synchronization and blocker discussion'),
    ARRAY['standup', 'team', 'synchronization', 'blockers', 'agile', 'scrum', 'daily']
),
-- Doctor appointment embedding
(
    'emb47ac10b-58cc-4372-a567-0e02b2c3d480',
    'e47ac10b-58cc-4372-a567-0e02b2c3d480',
    'qdrant-point-003',
    'v1.0',
    sha256('Doctor Appointment Annual physical checkup - Dr. Smith'),
    ARRAY['doctor', 'medical', 'appointment', 'physical', 'checkup', 'health', 'annual']
),
-- Strategy planning embedding
(
    'emb47ac10b-58cc-4372-a567-0e02b2c3d481',
    'e47ac10b-58cc-4372-a567-0e02b2c3d481',
    'qdrant-point-004',
    'v1.0',
    sha256('Q1 Strategy Planning Quarterly business review and strategy planning session'),
    ARRAY['strategy', 'planning', 'quarterly', 'business', 'review', 'strategic', 'q1']
),
-- Travel embedding
(
    'emb47ac10b-58cc-4372-a567-0e02b2c3d482',
    'e47ac10b-58cc-4372-a567-0e02b2c3d482',
    'qdrant-point-005',
    'v1.0',
    sha256('Flight to San Francisco Business trip - West Coast client meetings'),
    ARRAY['flight', 'travel', 'business', 'trip', 'san francisco', 'client', 'meetings', 'west coast']
) ON CONFLICT (event_id) DO NOTHING;

-- Insert comprehensive sample reminders for testing
INSERT INTO event_reminders (
    id,
    event_id,
    minutes_before,
    notification_type,
    scheduled_time,
    status,
    notification_id,
    delivered_at,
    retry_count
) VALUES 
-- Delivered reminder for completed client call
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d478',
    'e47ac10b-58cc-4372-a567-0e02b2c3d478',
    30,
    'email',
    NOW() - INTERVAL '2 hours 30 minutes',
    'delivered',
    'notif-001-email',
    NOW() - INTERVAL '2 hours 25 minutes',
    0
),
-- Sent reminder for current standup
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d479',
    'e47ac10b-58cc-4372-a567-0e02b2c3d479',
    15,
    'push',
    NOW() - INTERVAL '20 minutes',
    'sent',
    'notif-002-push',
    NULL,
    0
),
-- Pending reminders for doctor appointment (multiple types)
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d480a',
    'e47ac10b-58cc-4372-a567-0e02b2c3d480',
    1440, -- 24 hours
    'email',
    (date_trunc('day', NOW()) + INTERVAL '14 hours') - INTERVAL '24 hours',
    CASE 
        WHEN (date_trunc('day', NOW()) + INTERVAL '14 hours') - INTERVAL '24 hours' < NOW()
        THEN 'sent'
        ELSE 'pending'
    END,
    NULL,
    NULL,
    0
),
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d480b',
    'e47ac10b-58cc-4372-a567-0e02b2c3d480',
    120, -- 2 hours
    'sms',
    (date_trunc('day', NOW()) + INTERVAL '14 hours') - INTERVAL '2 hours',
    'pending',
    NULL,
    NULL,
    0
),
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d480c',
    'e47ac10b-58cc-4372-a567-0e02b2c3d480',
    15, -- 15 minutes
    'push',
    (date_trunc('day', NOW()) + INTERVAL '14 hours') - INTERVAL '15 minutes',
    'pending',
    NULL,
    NULL,
    0
),
-- Strategy planning reminders
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d481a',
    'e47ac10b-58cc-4372-a567-0e02b2c3d481',
    1440, -- 24 hours - preparation reminder
    'email',
    (date_trunc('day', NOW()) + INTERVAL '1 day 10 hours') - INTERVAL '24 hours',
    CASE 
        WHEN (date_trunc('day', NOW()) + INTERVAL '1 day 10 hours') - INTERVAL '24 hours' < NOW()
        THEN 'sent'
        ELSE 'pending'
    END,
    NULL,
    NULL,
    0
),
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d481b',
    'e47ac10b-58cc-4372-a567-0e02b2c3d481',
    30, -- 30 minutes - final reminder
    'push',
    (date_trunc('day', NOW()) + INTERVAL '1 day 10 hours') - INTERVAL '30 minutes',
    'pending',
    NULL,
    NULL,
    0
),
-- Flight reminders (multiple types for travel)
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d482a',
    'e47ac10b-58cc-4372-a567-0e02b2c3d482',
    1440, -- 24 hours - check-in reminder
    'email',
    (date_trunc('day', NOW()) + INTERVAL '3 days 6 hours') - INTERVAL '24 hours',
    'pending',
    NULL,
    NULL,
    0
),
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d482b',
    'e47ac10b-58cc-4372-a567-0e02b2c3d482',
    120, -- 2 hours - departure reminder
    'sms',
    (date_trunc('day', NOW()) + INTERVAL '3 days 6 hours') - INTERVAL '2 hours',
    'pending',
    NULL,
    NULL,
    0
),
-- Sarah's design review reminder
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d490',
    'e47ac10b-58cc-4372-a567-0e02b2c3d490',
    60,
    'email',
    (date_trunc('day', NOW()) + INTERVAL '2 hours') - INTERVAL '1 hour',
    CASE 
        WHEN (date_trunc('day', NOW()) + INTERVAL '2 hours') - INTERVAL '1 hour' < NOW()
        THEN 'sent'
        ELSE 'pending'
    END,
    NULL,
    NULL,
    0
),
-- Coffee chat reminder
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d491',
    'e47ac10b-58cc-4372-a567-0e02b2c3d491',
    30,
    'push',
    (date_trunc('day', NOW()) + INTERVAL '1 day 16 hours') - INTERVAL '30 minutes',
    'pending',
    NULL,
    NULL,
    0
),
-- Mike's performance review preparation
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d500',
    'e47ac10b-58cc-4372-a567-0e02b2c3d500',
    60,
    'email',
    (date_trunc('day', NOW()) + INTERVAL '1 day 9 hours') - INTERVAL '1 hour',
    'pending',
    NULL,
    NULL,
    0
),
-- Board meeting preparation (critical)
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d501',
    'e47ac10b-58cc-4372-a567-0e02b2c3d501',
    120, -- 2 hours early for critical prep
    'email',
    (date_trunc('day', NOW()) + INTERVAL '2 days 8 hours') - INTERVAL '2 hours',
    'pending',
    NULL,
    NULL,
    0
),
-- Weekly team meeting reminder
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d483',
    'e47ac10b-58cc-4372-a567-0e02b2c3d483',
    15,
    'push',
    (date_trunc('week', NOW()) + INTERVAL '1 day 15 hours') - INTERVAL '15 minutes',
    'pending',
    NULL,
    NULL,
    0
),
-- AI focus time reminder
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d503',
    'e47ac10b-58cc-4372-a567-0e02b2c3d503',
    30,
    'push',
    (date_trunc('day', NOW()) + INTERVAL '2 days 14 hours') - INTERVAL '30 minutes',
    'pending',
    NULL,
    NULL,
    0
),
-- Failed reminder example (with retry)
(
    'd47ac10b-58cc-4372-a567-0e02b2c3d504',
    'e47ac10b-58cc-4372-a567-0e02b2c3d481',
    60,
    'webhook',
    (date_trunc('day', NOW()) + INTERVAL '1 day 10 hours') - INTERVAL '1 hour',
    'failed',
    'notif-webhook-001',
    NULL,
    2
) ON CONFLICT (id) DO NOTHING;

-- ============================================================================
-- VERIFICATION AND STATISTICS
-- ============================================================================

-- Show comprehensive seed data statistics
DO $$
DECLARE
    user_count INTEGER;
    event_count INTEGER;
    reminder_count INTEGER;
    pattern_count INTEGER;
    embedding_count INTEGER;
    rec RECORD;
    status_counts RECORD;
BEGIN
    SELECT count(*) INTO user_count FROM users;
    SELECT count(*) INTO event_count FROM events;
    SELECT count(*) INTO reminder_count FROM event_reminders;
    SELECT count(*) INTO pattern_count FROM recurring_patterns;
    SELECT count(*) INTO embedding_count FROM event_embeddings;
    
    RAISE NOTICE '🎯 Calendar System - Comprehensive Seed Data Loaded';
    RAISE NOTICE '================================================';
    RAISE NOTICE '📊 Data Summary:';
    RAISE NOTICE '  - Users: % (multi-timezone, AI-enabled)', user_count;
    RAISE NOTICE '  - Events: % (diverse types & scenarios)', event_count;
    RAISE NOTICE '  - Reminders: % (multi-channel notifications)', reminder_count;
    RAISE NOTICE '  - Recurring patterns: % (daily, weekly, monthly)', pattern_count;
    RAISE NOTICE '  - AI embeddings: % (semantic search ready)', embedding_count;
    RAISE NOTICE '';
    
    -- Show event distribution by status
    RAISE NOTICE '📋 Event Status Distribution:';
    FOR status_counts IN 
        SELECT status, count(*) as count 
        FROM events 
        GROUP BY status 
        ORDER BY count DESC
    LOOP
        RAISE NOTICE '  - %: %', UPPER(status_counts.status), status_counts.count;
    END LOOP;
    RAISE NOTICE '';
    
    -- Show event type distribution
    RAISE NOTICE '📅 Event Type Distribution:';
    FOR status_counts IN 
        SELECT event_type, count(*) as count 
        FROM events 
        GROUP BY event_type 
        ORDER BY count DESC
    LOOP
        RAISE NOTICE '  - %: %', UPPER(status_counts.event_type), status_counts.count;
    END LOOP;
    RAISE NOTICE '';
    
    -- Show reminder status distribution
    RAISE NOTICE '🔔 Reminder Status Distribution:';
    FOR status_counts IN 
        SELECT status, count(*) as count 
        FROM event_reminders 
        GROUP BY status 
        ORDER BY count DESC
    LOOP
        RAISE NOTICE '  - %: %', UPPER(status_counts.status), status_counts.count;
    END LOOP;
    RAISE NOTICE '';
    
    -- Show sample upcoming events with user context
    RAISE NOTICE '⏰ Sample Upcoming Events:';
    FOR rec IN 
        SELECT 
            e.title, 
            e.start_time, 
            e.event_type,
            e.status,
            u.display_name,
            u.timezone,
            CASE 
                WHEN e.start_time <= NOW() AND e.end_time >= NOW() THEN '🔴 HAPPENING NOW'
                WHEN e.start_time > NOW() AND e.start_time <= NOW() + INTERVAL '2 hours' THEN '🟡 VERY SOON'
                WHEN e.start_time > NOW() AND e.start_time <= NOW() + INTERVAL '1 day' THEN '🟢 TODAY'
                ELSE '🔵 FUTURE'
            END as time_indicator
        FROM events e
        JOIN users u ON e.user_id = u.id
        WHERE e.start_time >= NOW() - INTERVAL '30 minutes' -- Include events happening now
        AND e.status = 'active'
        ORDER BY e.start_time 
        LIMIT 5
    LOOP
        RAISE NOTICE '  % [%] % (%) - % in %', 
            rec.time_indicator,
            to_char(rec.start_time, 'MM/DD HH24:MI'),
            rec.title, 
            UPPER(rec.event_type),
            rec.display_name,
            rec.timezone;
    END LOOP;
    RAISE NOTICE '';
    
    -- Show AI/automation features summary
    RAISE NOTICE '🤖 AI & Automation Features Demonstrated:';
    RAISE NOTICE '  - AI-optimized scheduling with confidence scores';
    RAISE NOTICE '  - Natural language event processing keywords';
    RAISE NOTICE '  - Cross-scenario integration configurations';
    RAISE NOTICE '  - Advanced notification workflows (email, SMS, push, webhook)';
    RAISE NOTICE '  - Travel time buffers and preparation automation';
    RAISE NOTICE '  - Multi-timezone coordination and scheduling';
    RAISE NOTICE '  - Recurring event management with exceptions';
    RAISE NOTICE '  - Post-meeting action automation and CRM integration';
    RAISE NOTICE '';
    
    -- Show integration points
    RAISE NOTICE '🔗 External Integration Examples:';
    RAISE NOTICE '  - scenario-authenticator: User authentication';
    RAISE NOTICE '  - notification-hub: Multi-channel reminders';
    RAISE NOTICE '  - qdrant: Semantic event search & recommendations';
    RAISE NOTICE '  - CRM systems: Client interaction tracking';
    RAISE NOTICE '  - Agile tools: Sprint/JIRA integration';
    RAISE NOTICE '  - Communication: Slack/Teams notifications';
    RAISE NOTICE '';
    
    RAISE NOTICE '✅ Calendar seed data installation complete!';
    RAISE NOTICE '🚀 System ready for development and testing scenarios';
    RAISE NOTICE '================================================';
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