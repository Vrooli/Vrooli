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