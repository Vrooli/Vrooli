# PostGIS Scenario Examples

Real-world scenarios that PostGIS enables for Vrooli's automation platform.

## Delivery Route Optimization

### Scenario
Optimize delivery routes for a food delivery service, considering distance, traffic patterns, and delivery time windows.

### Implementation
```sql
-- Store delivery locations
CREATE TABLE deliveries (
    id SERIAL PRIMARY KEY,
    customer_name VARCHAR(100),
    address TEXT,
    location GEOMETRY(Point, 4326),
    time_window tsrange,
    priority INTEGER
);

-- Find optimal next delivery
WITH current_location AS (
    SELECT ST_MakePoint(-73.985, 40.748)::geometry as pos
)
SELECT 
    d.id,
    d.customer_name,
    ST_Distance(d.location::geography, c.pos::geography) as distance_meters,
    d.priority
FROM deliveries d, current_location c
WHERE d.delivered = false
    AND NOW() <@ d.time_window
ORDER BY 
    d.priority DESC,
    distance_meters ASC
LIMIT 1;
```

## Real Estate Analysis

### Scenario
Analyze property values based on proximity to amenities like schools, parks, and public transit.

### Implementation
```sql
-- Calculate property scores
WITH amenity_scores AS (
    SELECT 
        p.id,
        p.address,
        COUNT(DISTINCT s.id) FILTER (
            WHERE ST_DWithin(p.location::geography, s.location::geography, 500)
        ) as nearby_schools,
        COUNT(DISTINCT pk.id) FILTER (
            WHERE ST_DWithin(p.location::geography, pk.location::geography, 300)
        ) as nearby_parks,
        MIN(ST_Distance(p.location::geography, t.location::geography)) as transit_distance
    FROM properties p
    LEFT JOIN schools s ON ST_DWithin(p.location::geography, s.location::geography, 1000)
    LEFT JOIN parks pk ON ST_DWithin(p.location::geography, pk.location::geography, 500)
    LEFT JOIN transit_stops t ON ST_DWithin(p.location::geography, t.location::geography, 2000)
    GROUP BY p.id, p.address
)
SELECT 
    address,
    nearby_schools * 10 + 
    nearby_parks * 5 + 
    CASE 
        WHEN transit_distance < 200 THEN 20
        WHEN transit_distance < 500 THEN 10
        ELSE 0
    END as amenity_score
FROM amenity_scores
ORDER BY amenity_score DESC;
```

## Geofencing and Alerts

### Scenario
Monitor vehicles entering/leaving designated zones and trigger automated alerts.

### Implementation
```sql
-- Define geofence zones
CREATE TABLE geofences (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100),
    zone_type VARCHAR(50), -- 'restricted', 'warehouse', 'customer'
    boundary GEOMETRY(Polygon, 4326),
    alert_email VARCHAR(100)
);

-- Check vehicle positions
CREATE OR REPLACE FUNCTION check_geofence_violations()
RETURNS TABLE(vehicle_id INT, zone_name VARCHAR, violation_type VARCHAR) AS $$
BEGIN
    RETURN QUERY
    WITH current_positions AS (
        SELECT 
            v.id,
            v.plate_number,
            v.last_position,
            v.previous_position
        FROM vehicles v
        WHERE v.last_update > NOW() - INTERVAL '5 minutes'
    )
    SELECT 
        cp.id as vehicle_id,
        g.name as zone_name,
        CASE 
            WHEN ST_Within(cp.last_position, g.boundary) 
                AND NOT ST_Within(cp.previous_position, g.boundary) 
            THEN 'entered'
            WHEN NOT ST_Within(cp.last_position, g.boundary) 
                AND ST_Within(cp.previous_position, g.boundary) 
            THEN 'exited'
        END as violation_type
    FROM current_positions cp
    CROSS JOIN geofences g
    WHERE (ST_Within(cp.last_position, g.boundary) != ST_Within(cp.previous_position, g.boundary));
END;
$$ LANGUAGE plpgsql;
```

## Environmental Monitoring

### Scenario
Track and analyze environmental sensor data across geographic regions.

### Implementation
```sql
-- Store sensor readings with location
CREATE TABLE sensor_readings (
    id SERIAL PRIMARY KEY,
    sensor_id VARCHAR(50),
    location GEOMETRY(Point, 4326),
    temperature DECIMAL(5,2),
    humidity DECIMAL(5,2),
    air_quality_index INTEGER,
    timestamp TIMESTAMPTZ DEFAULT NOW()
);

-- Generate pollution heatmap data
WITH grid AS (
    SELECT 
        ST_SquareGrid(0.01, ST_MakeEnvelope(-74.1, 40.7, -73.9, 40.9, 4326)) as cell
)
SELECT 
    ST_AsGeoJSON(g.cell) as grid_cell,
    AVG(s.air_quality_index) as avg_aqi,
    COUNT(s.id) as reading_count
FROM grid g
LEFT JOIN sensor_readings s 
    ON ST_Within(s.location, g.cell)
    AND s.timestamp > NOW() - INTERVAL '1 hour'
GROUP BY g.cell
HAVING COUNT(s.id) > 0;
```

## Emergency Response Planning

### Scenario
Find optimal locations for emergency services based on population density and response time requirements.

### Implementation
```sql
-- Analyze coverage gaps
WITH population_centers AS (
    SELECT 
        id,
        location,
        population
    FROM census_blocks
    WHERE population > 1000
),
emergency_coverage AS (
    SELECT 
        pc.id,
        pc.population,
        MIN(ST_Distance(pc.location::geography, es.location::geography)) as nearest_service
    FROM population_centers pc
    CROSS JOIN emergency_services es
    GROUP BY pc.id, pc.population
)
SELECT 
    ST_AsGeoJSON(ST_Centroid(ST_Union(cb.geometry))) as suggested_location,
    SUM(ec.population) as affected_population,
    AVG(ec.nearest_service) as avg_distance_to_service
FROM emergency_coverage ec
JOIN census_blocks cb ON cb.id = ec.id
WHERE ec.nearest_service > 5000  -- More than 5km from service
GROUP BY ST_SnapToGrid(cb.location, 0.01)  -- Group into grid cells
ORDER BY affected_population DESC
LIMIT 10;
```

## Supply Chain Optimization

### Scenario
Optimize warehouse placement and inventory distribution based on customer locations and demand patterns.

### Implementation
```sql
-- Find optimal warehouse location using K-means clustering
WITH customer_clusters AS (
    SELECT 
        ST_ClusterKMeans(location, 5) OVER () as cluster_id,
        location,
        avg_monthly_orders
    FROM customers
    WHERE active = true
)
SELECT 
    cluster_id,
    ST_AsGeoJSON(ST_Centroid(ST_Collect(location))) as suggested_warehouse_location,
    SUM(avg_monthly_orders) as total_demand,
    COUNT(*) as customer_count,
    AVG(ST_Distance(
        location::geography,
        ST_Centroid(ST_Collect(location) OVER (PARTITION BY cluster_id))::geography
    )) as avg_distance_to_center
FROM customer_clusters
GROUP BY cluster_id;
```

## Tourism and Recreation

### Scenario
Create personalized tour routes based on user preferences and time constraints.

### Implementation
```sql
-- Generate tourist route
CREATE OR REPLACE FUNCTION generate_tour_route(
    start_point geometry,
    interests text[],
    max_duration_hours integer
)
RETURNS TABLE(
    step integer,
    poi_name varchar,
    poi_type varchar,
    visit_duration integer,
    travel_time integer,
    location geometry
) AS $$
DECLARE
    current_location geometry := start_point;
    total_time integer := 0;
    step_counter integer := 0;
BEGIN
    WHILE total_time < max_duration_hours * 60 DO
        step_counter := step_counter + 1;
        
        INSERT INTO temp_route
        SELECT 
            step_counter,
            p.name,
            p.category,
            p.typical_duration,
            ST_Distance(current_location::geography, p.location::geography) / 83 as walk_time, -- 5km/h walking
            p.location
        FROM points_of_interest p
        WHERE p.category = ANY(interests)
            AND NOT EXISTS (
                SELECT 1 FROM temp_route tr WHERE tr.poi_name = p.name
            )
            AND ST_DWithin(current_location::geography, p.location::geography, 2000)
        ORDER BY 
            p.rating DESC,
            ST_Distance(current_location::geography, p.location::geography)
        LIMIT 1;
        
        -- Update current location and time
        SELECT location, visit_duration + travel_time 
        INTO current_location, total_time
        FROM temp_route 
        WHERE step = step_counter;
        
        total_time := total_time + total_time;
    END LOOP;
    
    RETURN QUERY SELECT * FROM temp_route ORDER BY step;
END;
$$ LANGUAGE plpgsql;
```

## Integration with Vrooli Workflows

These scenarios can be automated through Vrooli by:
1. **Scheduled Jobs**: Run spatial analysis queries periodically
2. **Event Triggers**: Execute geofencing checks on location updates
3. **API Endpoints**: Expose spatial queries as REST APIs
4. **Workflow Integration**: Chain spatial operations with other resources
5. **Visualization**: Generate maps and charts from spatial data

Each scenario becomes a reusable capability that other Vrooli applications can leverage, creating compound value as the system grows.