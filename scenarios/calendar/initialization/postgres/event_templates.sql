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