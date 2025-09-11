-- Time Tools Seed Data
-- Initial timezone definitions, holidays, and sample data

-- Insert common timezone definitions
INSERT INTO timezone_definitions (timezone_name, utc_offset_minutes, dst_offset_minutes, abbreviation, has_dst, country_code, region) VALUES
    ('UTC', 0, NULL, 'UTC', false, NULL, 'Universal'),
    ('America/New_York', -300, -240, 'EST/EDT', true, 'US', 'Eastern'),
    ('America/Chicago', -360, -300, 'CST/CDT', true, 'US', 'Central'),
    ('America/Denver', -420, -360, 'MST/MDT', true, 'US', 'Mountain'),
    ('America/Los_Angeles', -480, -420, 'PST/PDT', true, 'US', 'Pacific'),
    ('Europe/London', 0, 60, 'GMT/BST', true, 'GB', 'British'),
    ('Europe/Paris', 60, 120, 'CET/CEST', true, 'FR', 'Central European'),
    ('Europe/Berlin', 60, 120, 'CET/CEST', true, 'DE', 'Central European'),
    ('Asia/Tokyo', 540, NULL, 'JST', false, 'JP', 'Japan'),
    ('Asia/Shanghai', 480, NULL, 'CST', false, 'CN', 'China'),
    ('Asia/Singapore', 480, NULL, 'SGT', false, 'SG', 'Singapore'),
    ('Asia/Dubai', 240, NULL, 'GST', false, 'AE', 'Gulf'),
    ('Australia/Sydney', 600, 660, 'AEST/AEDT', true, 'AU', 'Eastern Australia')
ON CONFLICT (timezone_name) DO NOTHING;

-- Insert common US holidays for 2024
INSERT INTO holidays (name, date, holiday_type, country_code, is_federal, offices_closed, banks_closed) VALUES
    ('New Year''s Day', '2024-01-01', 'public', 'US', true, true, true),
    ('Martin Luther King Jr. Day', '2024-01-15', 'public', 'US', true, true, true),
    ('Presidents'' Day', '2024-02-19', 'public', 'US', true, true, true),
    ('Memorial Day', '2024-05-27', 'public', 'US', true, true, true),
    ('Independence Day', '2024-07-04', 'public', 'US', true, true, true),
    ('Labor Day', '2024-09-02', 'public', 'US', true, true, true),
    ('Columbus Day', '2024-10-14', 'public', 'US', true, false, true),
    ('Veterans Day', '2024-11-11', 'public', 'US', true, true, true),
    ('Thanksgiving Day', '2024-11-28', 'public', 'US', true, true, true),
    ('Christmas Day', '2024-12-25', 'public', 'US', true, true, true)
ON CONFLICT (name, date, country_code) DO NOTHING;

-- Insert default business hours (Mon-Fri 9-5)
INSERT INTO business_hours (entity_id, entity_type, day_of_week, start_time, end_time, timezone, break_start, break_end) VALUES
    ('default', 'organization', 0, '09:00:00', '17:00:00', 'America/New_York', NULL, NULL), -- Sunday (closed)
    ('default', 'organization', 1, '09:00:00', '17:00:00', 'America/New_York', '12:00:00', '13:00:00'), -- Monday
    ('default', 'organization', 2, '09:00:00', '17:00:00', 'America/New_York', '12:00:00', '13:00:00'), -- Tuesday
    ('default', 'organization', 3, '09:00:00', '17:00:00', 'America/New_York', '12:00:00', '13:00:00'), -- Wednesday
    ('default', 'organization', 4, '09:00:00', '17:00:00', 'America/New_York', '12:00:00', '13:00:00'), -- Thursday
    ('default', 'organization', 5, '09:00:00', '17:00:00', 'America/New_York', '12:00:00', '13:00:00'), -- Friday
    ('default', 'organization', 6, '09:00:00', '17:00:00', 'America/New_York', NULL, NULL) -- Saturday (closed)
ON CONFLICT (entity_id, entity_type, day_of_week, effective_from) DO NOTHING;

-- Update business hours for weekends to mark as non-working days
UPDATE business_hours 
SET is_working_day = false 
WHERE entity_id = 'default' 
  AND entity_type = 'organization' 
  AND day_of_week IN (0, 6);

-- Insert sample scheduled events for demonstration
INSERT INTO scheduled_events (title, description, start_time, end_time, timezone, event_type, status, priority, organizer_id) VALUES
    ('Team Standup', 'Daily team synchronization meeting', 
     CURRENT_DATE + INTERVAL '9 hours', CURRENT_DATE + INTERVAL '9 hours 15 minutes',
     'America/New_York', 'meeting', 'scheduled', 'normal', 'team-lead'),
    ('Project Review', 'Quarterly project review with stakeholders',
     CURRENT_DATE + INTERVAL '2 days 14 hours', CURRENT_DATE + INTERVAL '2 days 15 hours',
     'America/New_York', 'meeting', 'scheduled', 'high', 'project-manager'),
    ('Lunch & Learn', 'Technical presentation on new features',
     CURRENT_DATE + INTERVAL '7 days 12 hours', CURRENT_DATE + INTERVAL '7 days 13 hours',
     'America/New_York', 'presentation', 'scheduled', 'normal', 'tech-lead')
ON CONFLICT DO NOTHING;

-- Insert sample recurrence pattern for daily standup
INSERT INTO recurrence_patterns (pattern_type, interval_value, days_of_week, start_date, timezone, description) VALUES
    ('weekly', 1, ARRAY[1,2,3,4,5], CURRENT_DATE, 'America/New_York', 'Weekday standup meetings'),
    ('monthly', 1, NULL, CURRENT_DATE, 'America/New_York', 'Monthly all-hands meeting')
ON CONFLICT DO NOTHING;

-- Insert sample available slots for scheduling
INSERT INTO available_slots (entity_id, entity_type, start_time, end_time, duration_minutes, timezone, availability_type, can_schedule) VALUES
    ('alice', 'user', CURRENT_TIMESTAMP + INTERVAL '1 day 10 hours', CURRENT_TIMESTAMP + INTERVAL '1 day 11 hours', 60, 'America/New_York', 'free', true),
    ('alice', 'user', CURRENT_TIMESTAMP + INTERVAL '1 day 14 hours', CURRENT_TIMESTAMP + INTERVAL '1 day 16 hours', 120, 'America/New_York', 'free', true),
    ('bob', 'user', CURRENT_TIMESTAMP + INTERVAL '1 day 9 hours', CURRENT_TIMESTAMP + INTERVAL '1 day 12 hours', 180, 'America/New_York', 'free', true),
    ('bob', 'user', CURRENT_TIMESTAMP + INTERVAL '1 day 13 hours', CURRENT_TIMESTAMP + INTERVAL '1 day 15 hours', 120, 'America/New_York', 'free', true)
ON CONFLICT DO NOTHING;