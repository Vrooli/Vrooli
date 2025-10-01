-- Migration: 003_fix_schema_mismatches
-- Description: Fix schema mismatches for templates and attendees
-- Author: AI Agent
-- Date: 2025-09-27

-- Fix event_templates table - change user_id to proper type
ALTER TABLE IF EXISTS event_templates 
  ALTER COLUMN user_id TYPE VARCHAR(255) USING user_id::VARCHAR(255);

-- Fix event_attendees table - ensure it exists with proper schema
CREATE TABLE IF NOT EXISTS event_attendees (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    event_id UUID NOT NULL REFERENCES events(id) ON DELETE CASCADE,
    user_id VARCHAR(255) NOT NULL,
    name VARCHAR(255),
    email VARCHAR(255),
    rsvp_status VARCHAR(20) DEFAULT 'pending',
    rsvp_message TEXT,
    response_time TIMESTAMPTZ,
    attendance_status VARCHAR(20),
    check_in_time TIMESTAMPTZ,
    check_in_method VARCHAR(20),
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    
    CONSTRAINT uq_event_attendee UNIQUE (event_id, user_id),
    CONSTRAINT chk_rsvp_status CHECK (
        rsvp_status IN ('pending', 'accepted', 'declined', 'tentative')
    ),
    CONSTRAINT chk_attendance_status CHECK (
        attendance_status IS NULL OR 
        attendance_status IN ('attended', 'no_show', 'partial', 'excused')
    ),
    CONSTRAINT chk_checkin_method CHECK (
        check_in_method IS NULL OR
        check_in_method IN ('manual', 'qr_code', 'auto', 'proximity')
    )
);

-- Create event_templates table if it doesn't exist
CREATE TABLE IF NOT EXISTS event_templates (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id VARCHAR(255),
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

-- Add indexes if they don't exist
CREATE INDEX IF NOT EXISTS idx_attendees_event_id ON event_attendees (event_id);
CREATE INDEX IF NOT EXISTS idx_attendees_user_id ON event_attendees (user_id);
CREATE INDEX IF NOT EXISTS idx_attendees_rsvp_status ON event_attendees (rsvp_status);
CREATE INDEX IF NOT EXISTS idx_attendees_attendance_status ON event_attendees (attendance_status);
CREATE INDEX IF NOT EXISTS idx_attendees_response_time ON event_attendees (response_time);
CREATE INDEX IF NOT EXISTS idx_attendees_event_rsvp ON event_attendees (event_id, rsvp_status);
CREATE INDEX IF NOT EXISTS idx_attendees_event_attendance ON event_attendees (event_id, attendance_status);

CREATE INDEX IF NOT EXISTS idx_templates_user_id ON event_templates (user_id);
CREATE INDEX IF NOT EXISTS idx_templates_category ON event_templates (category);
CREATE INDEX IF NOT EXISTS idx_templates_is_system ON event_templates (is_system);
CREATE INDEX IF NOT EXISTS idx_templates_use_count ON event_templates (use_count DESC);

-- Ensure trigger function exists
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create triggers if they don't exist
DROP TRIGGER IF EXISTS tr_templates_updated_at ON event_templates;
CREATE TRIGGER tr_templates_updated_at 
    BEFORE UPDATE ON event_templates 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

DROP TRIGGER IF EXISTS tr_attendees_updated_at ON event_attendees;
CREATE TRIGGER tr_attendees_updated_at 
    BEFORE UPDATE ON event_attendees 
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Record migration as applied
INSERT INTO schema_migrations (version, checksum) 
VALUES ('003_fix_schema_mismatches', 'fix_templates_attendees_2025_09_27')
ON CONFLICT (version) DO NOTHING;