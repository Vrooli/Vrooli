-- Geofencing Example: Monitor when objects enter/exit zones
-- Useful for delivery tracking, security alerts, location-based automation

-- Create geofence zones
CREATE TABLE IF NOT EXISTS geofence_zones (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    zone GEOMETRY(Polygon, 4326) NOT NULL,
    trigger_on_enter BOOLEAN DEFAULT true,
    trigger_on_exit BOOLEAN DEFAULT true,
    webhook_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create tracked objects (vehicles, devices, people)
CREATE TABLE IF NOT EXISTS tracked_objects (
    id SERIAL PRIMARY KEY,
    identifier VARCHAR(100) UNIQUE NOT NULL,
    type VARCHAR(50), -- vehicle, person, device, etc
    current_location GEOMETRY(Point, 4326),
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    metadata JSONB
);

-- Create geofence event log
CREATE TABLE IF NOT EXISTS geofence_events (
    id SERIAL PRIMARY KEY,
    object_id INTEGER REFERENCES tracked_objects(id),
    zone_id INTEGER REFERENCES geofence_zones(id),
    event_type VARCHAR(20), -- ENTER, EXIT
    location GEOMETRY(Point, 4326),
    occurred_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Example: Create a delivery zone around a warehouse
INSERT INTO geofence_zones (name, description, zone) VALUES 
(
    'Warehouse A',
    'Main distribution center',
    ST_GeomFromText('POLYGON((-122.420 37.775, -122.418 37.775, -122.418 37.777, -122.420 37.777, -122.420 37.775))', 4326)
)
ON CONFLICT DO NOTHING;

-- Function to check geofence triggers
CREATE OR REPLACE FUNCTION check_geofence_trigger(
    p_object_id INTEGER,
    p_new_location GEOMETRY(Point, 4326)
) RETURNS TABLE(zone_name VARCHAR, event_type VARCHAR, webhook_url TEXT) AS $$
DECLARE
    v_old_location GEOMETRY(Point, 4326);
BEGIN
    -- Get previous location
    SELECT current_location INTO v_old_location 
    FROM tracked_objects WHERE id = p_object_id;
    
    -- Check all zones for enter/exit events
    RETURN QUERY
    SELECT 
        gz.name,
        CASE 
            WHEN NOT ST_Contains(gz.zone, v_old_location) AND ST_Contains(gz.zone, p_new_location) THEN 'ENTER'
            WHEN ST_Contains(gz.zone, v_old_location) AND NOT ST_Contains(gz.zone, p_new_location) THEN 'EXIT'
        END as event,
        gz.webhook_url
    FROM geofence_zones gz
    WHERE (gz.trigger_on_enter AND NOT ST_Contains(gz.zone, v_old_location) AND ST_Contains(gz.zone, p_new_location))
       OR (gz.trigger_on_exit AND ST_Contains(gz.zone, v_old_location) AND NOT ST_Contains(gz.zone, p_new_location));
END;
$$ LANGUAGE plpgsql;

-- Example query: Find all objects currently inside a zone
SELECT 
    t.identifier,
    t.type,
    ST_AsText(t.current_location) as location,
    z.name as zone_name
FROM tracked_objects t
JOIN geofence_zones z ON ST_Contains(z.zone, t.current_location)
WHERE z.name = 'Warehouse A';
