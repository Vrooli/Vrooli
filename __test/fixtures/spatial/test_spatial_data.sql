-- Test spatial data for PostGIS integration tests
-- Creates sample geospatial data for testing PostGIS functionality

-- Create test table with geometry column
CREATE TABLE IF NOT EXISTS test_locations (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    category VARCHAR(50),
    location GEOMETRY(POINT, 4326),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Insert test data (various locations around the world)
INSERT INTO test_locations (name, category, location) VALUES
    ('Eiffel Tower', 'landmark', ST_GeomFromText('POINT(2.2945 48.8584)', 4326)),
    ('Statue of Liberty', 'landmark', ST_GeomFromText('POINT(-74.0445 40.6892)', 4326)),
    ('Sydney Opera House', 'landmark', ST_GeomFromText('POINT(151.2153 -33.8568)', 4326)),
    ('Tokyo Tower', 'landmark', ST_GeomFromText('POINT(139.7454 35.6586)', 4326)),
    ('Central Park', 'park', ST_GeomFromText('POINT(-73.9712 40.7831)', 4326));

-- Create spatial index
CREATE INDEX IF NOT EXISTS idx_test_locations_location ON test_locations USING GIST(location);

-- Test query: Find all locations within 100km of New York (40.7128, -74.0060)
SELECT name, category, 
       ST_Distance(location::geography, ST_GeomFromText('POINT(-74.0060 40.7128)', 4326)::geography) / 1000 AS distance_km
FROM test_locations
WHERE ST_DWithin(location::geography, ST_GeomFromText('POINT(-74.0060 40.7128)', 4326)::geography, 100000)
ORDER BY distance_km;