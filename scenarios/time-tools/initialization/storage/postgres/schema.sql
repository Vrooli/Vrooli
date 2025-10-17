-- Time Tools Database Schema
-- Comprehensive temporal operations and scheduling platform

-- Enable necessary extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "btree_gist";  -- For temporal exclusion constraints

-- Timezone definitions table
CREATE TABLE IF NOT EXISTS timezone_definitions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    timezone_name VARCHAR(100) UNIQUE NOT NULL,  -- e.g., 'America/New_York'
    utc_offset_minutes INTEGER NOT NULL,         -- Current offset in minutes
    dst_offset_minutes INTEGER,                  -- DST offset if applicable
    abbreviation VARCHAR(10),                    -- e.g., 'EST', 'PST'
    has_dst BOOLEAN DEFAULT false,
    dst_start_rule JSONB,                        -- DST transition rules
    dst_end_rule JSONB,
    country_code VARCHAR(2),
    region VARCHAR(100),
    metadata JSONB DEFAULT '{}',
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Scheduled events table
CREATE TABLE IF NOT EXISTS scheduled_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title VARCHAR(255) NOT NULL,
    description TEXT,
    
    -- Temporal information
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    duration_minutes INTEGER GENERATED ALWAYS AS (EXTRACT(EPOCH FROM (end_time - start_time))/60) STORED,
    timezone VARCHAR(100) NOT NULL,
    all_day BOOLEAN DEFAULT false,
    
    -- Event metadata
    event_type VARCHAR(50),  -- meeting, appointment, deadline, reminder
    status VARCHAR(50) DEFAULT 'scheduled',  -- scheduled, in_progress, completed, cancelled
    priority VARCHAR(20) DEFAULT 'normal',   -- low, normal, high, urgent
    
    -- Participants and resources
    organizer_id VARCHAR(255),
    participants JSONB DEFAULT '[]',  -- Array of participant objects
    resources JSONB DEFAULT '[]',      -- Required resources (rooms, equipment)
    
    -- Location
    location VARCHAR(255),
    location_type VARCHAR(50),  -- physical, virtual, hybrid
    virtual_meeting_url TEXT,
    
    -- Recurrence
    recurrence_pattern_id UUID,
    is_recurring BOOLEAN DEFAULT false,
    recurrence_exception_dates DATE[],
    original_start_time TIMESTAMP WITH TIME ZONE,  -- For recurring event instances
    
    -- Notifications
    reminder_minutes INTEGER[],  -- Array of reminder times before event
    notification_sent BOOLEAN DEFAULT false,
    
    -- Integration
    external_id VARCHAR(255),
    external_source VARCHAR(50),  -- google, outlook, etc.
    sync_status VARCHAR(50),
    last_synced_at TIMESTAMP WITH TIME ZONE,
    
    -- Metadata
    tags TEXT[],
    custom_fields JSONB DEFAULT '{}',
    color VARCHAR(7),  -- Hex color for UI
    
    -- Timestamps
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE,
    
    -- Ensure no overlapping events for the same organizer (optional constraint)
    CONSTRAINT no_overlap CHECK (start_time < end_time)
);

-- Recurrence patterns table
CREATE TABLE IF NOT EXISTS recurrence_patterns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    pattern_type VARCHAR(50) NOT NULL,  -- daily, weekly, monthly, yearly, custom
    
    -- Frequency settings
    interval_value INTEGER DEFAULT 1,   -- Every N days/weeks/months
    days_of_week INTEGER[],             -- 0=Sunday, 6=Saturday
    days_of_month INTEGER[],            -- 1-31
    months_of_year INTEGER[],           -- 1-12
    week_of_month INTEGER,              -- 1-5 (5=last)
    
    -- Advanced patterns
    custom_pattern JSONB,               -- For complex patterns
    
    -- Boundaries
    start_date DATE NOT NULL,
    end_date DATE,
    max_occurrences INTEGER,
    
    -- Timezone handling
    timezone VARCHAR(100) NOT NULL,
    adjust_for_dst BOOLEAN DEFAULT true,
    
    -- Metadata
    description TEXT,
    is_active BOOLEAN DEFAULT true,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Business hours table
CREATE TABLE IF NOT EXISTS business_hours (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id VARCHAR(255),             -- Organization/user ID
    entity_type VARCHAR(50),            -- organization, user, resource
    
    -- Weekly schedule
    day_of_week INTEGER NOT NULL CHECK (day_of_week >= 0 AND day_of_week <= 6),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    is_working_day BOOLEAN DEFAULT true,
    
    -- Timezone
    timezone VARCHAR(100) NOT NULL,
    
    -- Breaks
    break_start TIME,
    break_end TIME,
    
    -- Metadata
    effective_from DATE,
    effective_until DATE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(entity_id, entity_type, day_of_week, effective_from)
);

-- Holidays and special dates table
CREATE TABLE IF NOT EXISTS holidays (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(255) NOT NULL,
    date DATE NOT NULL,
    
    -- Holiday type and scope
    holiday_type VARCHAR(50),           -- public, bank, religious, company
    country_code VARCHAR(2),
    region VARCHAR(100),
    is_federal BOOLEAN DEFAULT false,
    
    -- Business impact
    offices_closed BOOLEAN DEFAULT true,
    banks_closed BOOLEAN DEFAULT false,
    schools_closed BOOLEAN DEFAULT false,
    
    -- Recurrence
    is_recurring BOOLEAN DEFAULT false,
    recurrence_rule JSONB,              -- For annual holidays
    
    -- Observance rules
    observed_date DATE,                 -- Actual observed date if different
    observance_rule VARCHAR(50),        -- nearest_weekday, next_monday, etc.
    
    -- Metadata
    description TEXT,
    traditions TEXT,
    year INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE(name, date, country_code)
);

-- Time analytics table
CREATE TABLE IF NOT EXISTS time_analytics (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id VARCHAR(255),
    entity_type VARCHAR(50),            -- user, team, organization
    
    -- Period
    period_start TIMESTAMP WITH TIME ZONE NOT NULL,
    period_end TIMESTAMP WITH TIME ZONE NOT NULL,
    period_type VARCHAR(20),            -- day, week, month, quarter, year
    
    -- Meeting metrics
    total_meetings INTEGER DEFAULT 0,
    total_meeting_minutes INTEGER DEFAULT 0,
    average_meeting_duration INTEGER,
    longest_meeting_duration INTEGER,
    shortest_meeting_duration INTEGER,
    
    -- Scheduling metrics
    meetings_scheduled INTEGER DEFAULT 0,
    meetings_cancelled INTEGER DEFAULT 0,
    meetings_rescheduled INTEGER DEFAULT 0,
    no_show_count INTEGER DEFAULT 0,
    
    -- Time distribution
    morning_meetings INTEGER DEFAULT 0,  -- Before noon
    afternoon_meetings INTEGER DEFAULT 0, -- Noon to 5pm
    evening_meetings INTEGER DEFAULT 0,   -- After 5pm
    weekend_meetings INTEGER DEFAULT 0,
    
    -- Efficiency metrics
    back_to_back_meetings INTEGER DEFAULT 0,
    buffer_time_minutes INTEGER DEFAULT 0,
    focus_time_minutes INTEGER DEFAULT 0,
    
    -- Participation
    unique_participants INTEGER DEFAULT 0,
    average_participants NUMERIC(5,2),
    
    -- Timezone metrics
    cross_timezone_meetings INTEGER DEFAULT 0,
    unique_timezones INTEGER DEFAULT 0,
    
    -- Metadata
    calculated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB DEFAULT '{}'
);

-- Scheduling conflicts table
CREATE TABLE IF NOT EXISTS scheduling_conflicts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event1_id UUID REFERENCES scheduled_events(id) ON DELETE CASCADE,
    event2_id UUID REFERENCES scheduled_events(id) ON DELETE CASCADE,
    
    -- Conflict details
    conflict_type VARCHAR(50),          -- time_overlap, resource, participant
    severity VARCHAR(20),               -- low, medium, high, critical
    overlap_minutes INTEGER,
    
    -- Resolution
    resolution_status VARCHAR(50) DEFAULT 'unresolved',
    resolution_action TEXT,
    resolved_by VARCHAR(255),
    resolved_at TIMESTAMP WITH TIME ZONE,
    
    -- Metadata
    detected_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    notification_sent BOOLEAN DEFAULT false,
    
    UNIQUE(event1_id, event2_id)
);

-- Available time slots (for scheduling optimization)
CREATE TABLE IF NOT EXISTS available_slots (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    entity_id VARCHAR(255),
    entity_type VARCHAR(50),
    
    -- Slot information
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    duration_minutes INTEGER,
    timezone VARCHAR(100) NOT NULL,
    
    -- Availability type
    availability_type VARCHAR(50),      -- free, busy, tentative
    can_schedule BOOLEAN DEFAULT true,
    preferred_slot BOOLEAN DEFAULT false,
    
    -- Metadata
    source VARCHAR(50),                 -- manual, calendar_sync, business_hours
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP WITH TIME ZONE
);

-- Duration calculations log
CREATE TABLE IF NOT EXISTS duration_calculations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    calculation_type VARCHAR(50),       -- business_days, working_hours, elapsed
    
    -- Input parameters
    start_time TIMESTAMP WITH TIME ZONE NOT NULL,
    end_time TIMESTAMP WITH TIME ZONE NOT NULL,
    timezone VARCHAR(100),
    exclude_weekends BOOLEAN DEFAULT false,
    exclude_holidays BOOLEAN DEFAULT false,
    business_hours_id UUID,
    
    -- Results
    total_minutes INTEGER,
    business_minutes INTEGER,
    calendar_days INTEGER,
    business_days INTEGER,
    
    -- Metadata
    requested_by VARCHAR(255),
    calculation_time_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for performance
CREATE INDEX idx_scheduled_events_start ON scheduled_events(start_time) WHERE deleted_at IS NULL;
CREATE INDEX idx_scheduled_events_end ON scheduled_events(end_time) WHERE deleted_at IS NULL;
CREATE INDEX idx_scheduled_events_organizer ON scheduled_events(organizer_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_scheduled_events_status ON scheduled_events(status);
CREATE INDEX idx_scheduled_events_recurring ON scheduled_events(recurrence_pattern_id) WHERE is_recurring = true;
CREATE INDEX idx_scheduled_events_external ON scheduled_events(external_id, external_source);
CREATE INDEX idx_scheduled_events_timezone ON scheduled_events(timezone);
CREATE INDEX idx_scheduled_events_daterange ON scheduled_events USING gist(tstzrange(start_time, end_time));

CREATE INDEX idx_business_hours_entity ON business_hours(entity_id, entity_type);
CREATE INDEX idx_business_hours_day ON business_hours(day_of_week);

CREATE INDEX idx_holidays_date ON holidays(date);
CREATE INDEX idx_holidays_country ON holidays(country_code);
CREATE INDEX idx_holidays_year ON holidays(year);

CREATE INDEX idx_time_analytics_entity ON time_analytics(entity_id, entity_type);
CREATE INDEX idx_time_analytics_period ON time_analytics(period_start, period_end);

CREATE INDEX idx_conflicts_unresolved ON scheduling_conflicts(resolution_status) WHERE resolution_status = 'unresolved';

CREATE INDEX idx_available_slots_entity ON available_slots(entity_id, entity_type);
CREATE INDEX idx_available_slots_time ON available_slots(start_time, end_time);
CREATE INDEX idx_available_slots_can_schedule ON available_slots(can_schedule) WHERE can_schedule = true;

-- Update triggers for timestamps
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_scheduled_events_updated_at BEFORE UPDATE ON scheduled_events
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_timezone_definitions_updated_at BEFORE UPDATE ON timezone_definitions
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

CREATE TRIGGER update_recurrence_patterns_updated_at BEFORE UPDATE ON recurrence_patterns
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Function to detect scheduling conflicts
CREATE OR REPLACE FUNCTION detect_scheduling_conflicts()
RETURNS TRIGGER AS $$
BEGIN
    -- Check for time overlaps with existing events for the same organizer
    INSERT INTO scheduling_conflicts (event1_id, event2_id, conflict_type, severity, overlap_minutes)
    SELECT 
        NEW.id,
        e.id,
        'time_overlap',
        CASE 
            WHEN NEW.priority = 'urgent' OR e.priority = 'urgent' THEN 'critical'
            WHEN NEW.priority = 'high' OR e.priority = 'high' THEN 'high'
            ELSE 'medium'
        END,
        EXTRACT(EPOCH FROM (
            LEAST(NEW.end_time, e.end_time) - GREATEST(NEW.start_time, e.start_time)
        ))/60
    FROM scheduled_events e
    WHERE e.id != NEW.id
      AND e.organizer_id = NEW.organizer_id
      AND e.deleted_at IS NULL
      AND e.status != 'cancelled'
      AND (e.start_time, e.end_time) OVERLAPS (NEW.start_time, NEW.end_time)
    ON CONFLICT (event1_id, event2_id) DO NOTHING;
    
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER detect_conflicts_on_event_insert
    AFTER INSERT ON scheduled_events
    FOR EACH ROW
    EXECUTE FUNCTION detect_scheduling_conflicts();

-- Function to calculate business hours between two timestamps
CREATE OR REPLACE FUNCTION calculate_business_hours(
    start_ts TIMESTAMP WITH TIME ZONE,
    end_ts TIMESTAMP WITH TIME ZONE,
    tz VARCHAR(100),
    entity_id_param VARCHAR(255)
) RETURNS INTEGER AS $$
DECLARE
    total_minutes INTEGER := 0;
    current_date DATE;
    day_start TIME;
    day_end TIME;
    current_start TIMESTAMP WITH TIME ZONE;
    current_end TIMESTAMP WITH TIME ZONE;
BEGIN
    current_date := DATE(start_ts AT TIME ZONE tz);
    
    WHILE current_date <= DATE(end_ts AT TIME ZONE tz) LOOP
        -- Get business hours for this day
        SELECT start_time, end_time INTO day_start, day_end
        FROM business_hours
        WHERE entity_id = entity_id_param
          AND day_of_week = EXTRACT(DOW FROM current_date)
          AND is_working_day = true
          AND (effective_from IS NULL OR effective_from <= current_date)
          AND (effective_until IS NULL OR effective_until >= current_date)
        ORDER BY effective_from DESC
        LIMIT 1;
        
        IF day_start IS NOT NULL THEN
            -- Calculate the portion of this day that falls within our range
            current_start := GREATEST(
                start_ts,
                (current_date || ' ' || day_start)::TIMESTAMP AT TIME ZONE tz
            );
            current_end := LEAST(
                end_ts,
                (current_date || ' ' || day_end)::TIMESTAMP AT TIME ZONE tz
            );
            
            IF current_start < current_end THEN
                total_minutes := total_minutes + EXTRACT(EPOCH FROM (current_end - current_start))/60;
            END IF;
        END IF;
        
        current_date := current_date + INTERVAL '1 day';
    END LOOP;
    
    RETURN total_minutes;
END;
$$ LANGUAGE plpgsql;

-- Useful views for time management
CREATE OR REPLACE VIEW upcoming_events AS
SELECT 
    se.id,
    se.title,
    se.start_time,
    se.end_time,
    se.duration_minutes,
    se.timezone,
    se.event_type,
    se.status,
    se.location,
    se.organizer_id,
    array_length(se.participants, 1) as participant_count
FROM scheduled_events se
WHERE se.start_time > CURRENT_TIMESTAMP
  AND se.deleted_at IS NULL
  AND se.status != 'cancelled'
ORDER BY se.start_time
LIMIT 100;

CREATE OR REPLACE VIEW daily_schedule AS
SELECT 
    se.id,
    se.title,
    se.start_time::TIME as start_time,
    se.end_time::TIME as end_time,
    se.duration_minutes,
    se.event_type,
    se.location,
    se.priority
FROM scheduled_events se
WHERE DATE(se.start_time) = CURRENT_DATE
  AND se.deleted_at IS NULL
  AND se.status != 'cancelled'
ORDER BY se.start_time;

CREATE OR REPLACE VIEW conflict_summary AS
SELECT 
    sc.id,
    e1.title as event1_title,
    e2.title as event2_title,
    sc.conflict_type,
    sc.severity,
    sc.overlap_minutes,
    sc.resolution_status,
    sc.detected_at
FROM scheduling_conflicts sc
JOIN scheduled_events e1 ON sc.event1_id = e1.id
JOIN scheduled_events e2 ON sc.event2_id = e2.id
WHERE sc.resolution_status = 'unresolved'
ORDER BY sc.severity DESC, sc.detected_at DESC;