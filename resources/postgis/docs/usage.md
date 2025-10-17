# PostGIS Usage Guide

## Quick Start

### Basic Commands
```bash
# Check PostGIS status
vrooli resource postgis status

# View status in JSON format
vrooli resource postgis status --format json

# Start PostGIS (if not running)
vrooli resource postgis start

# Stop PostGIS (avoid unless necessary)
vrooli resource postgis stop
```

### Working with Spatial Data

#### Injecting SQL Files
```bash
# Execute spatial SQL file
vrooli resource postgis inject /path/to/spatial_queries.sql

# Execute in specific database
vrooli resource postgis inject /path/to/queries.sql mydatabase
```

#### Importing Geographic Data
```bash
# Import shapefile
vrooli resource postgis import-shapefile /path/to/data.shp

# Import with custom table name
vrooli resource postgis import-shapefile /path/to/data.shp --table cities

# Export table to shapefile
vrooli resource postgis export-shapefile my_spatial_table
```

## Common Spatial Queries

### Finding Nearby Points
```sql
-- Find all locations within 1km of a point
SELECT name, ST_Distance(
    location::geography,
    ST_MakePoint(-73.985428, 40.748817)::geography
) as distance_meters
FROM locations
WHERE ST_DWithin(
    location::geography,
    ST_MakePoint(-73.985428, 40.748817)::geography,
    1000  -- 1km in meters
)
ORDER BY distance_meters;
```

### Creating Geographic Regions
```sql
-- Create a buffer zone around a point
INSERT INTO zones (name, boundary)
VALUES (
    'Downtown Safety Zone',
    ST_Buffer(
        ST_MakePoint(-73.985428, 40.748817)::geography,
        500  -- 500 meter radius
    )::geometry
);
```

### Spatial Joins
```sql
-- Find which district contains each store
SELECT 
    s.name as store_name,
    d.name as district_name
FROM stores s
JOIN districts d ON ST_Within(s.location, d.boundary);
```

## Data Injection Patterns

### Creating Spatial Tables
```sql
-- cities.sql
CREATE TABLE IF NOT EXISTS cities (
    id SERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    country VARCHAR(50),
    population INTEGER,
    location GEOMETRY(Point, 4326),
    boundary GEOMETRY(Polygon, 4326)
);

-- Add spatial index
CREATE INDEX idx_cities_location ON cities USING GIST(location);
CREATE INDEX idx_cities_boundary ON cities USING GIST(boundary);
```

### Batch Data Import
```sql
-- import_cities.sql
INSERT INTO cities (name, country, population, location) VALUES
    ('New York', 'USA', 8336817, ST_SetSRID(ST_MakePoint(-74.006, 40.7128), 4326)),
    ('London', 'UK', 8982000, ST_SetSRID(ST_MakePoint(-0.1276, 51.5074), 4326)),
    ('Tokyo', 'Japan', 13960000, ST_SetSRID(ST_MakePoint(139.6503, 35.6762), 4326));
```

### Geographic Calculations
```sql
-- Calculate areas and distances
SELECT 
    name,
    ST_Area(boundary::geography) / 1000000 as area_km2,
    ST_Perimeter(boundary::geography) / 1000 as perimeter_km
FROM regions;
```

## Integration Examples

### With n8n Workflows
```javascript
// n8n function node to query PostGIS
const query = `
    SELECT name, ST_AsGeoJSON(location) as geojson
    FROM stores
    WHERE ST_DWithin(
        location::geography,
        ST_MakePoint($1, $2)::geography,
        $3
    )
`;

const result = await $db.query(query, [longitude, latitude, radius]);
return result.map(row => ({
    name: row.name,
    geometry: JSON.parse(row.geojson)
}));
```

### With Python Scripts
```python
# spatial_analysis.py
import psycopg2
from psycopg2.extras import RealDictCursor

conn = psycopg2.connect(
    host="localhost",
    port=5434,
    database="spatial",
    user="vrooli",
    password="vrooli"
)

with conn.cursor(cursor_factory=RealDictCursor) as cur:
    cur.execute("""
        SELECT 
            name,
            ST_X(location) as longitude,
            ST_Y(location) as latitude
        FROM points_of_interest
        WHERE category = %s
    """, ('restaurant',))
    
    restaurants = cur.fetchall()
```

## Performance Tips

### Indexing Strategy
```sql
-- Spatial index for point queries
CREATE INDEX idx_spatial ON my_table USING GIST(geom);

-- Spatial index for geography calculations
CREATE INDEX idx_geography ON my_table USING GIST(geography(geom));

-- Compound index for filtered spatial queries
CREATE INDEX idx_category_spatial ON my_table(category) 
    WHERE geom IS NOT NULL;
```

### Query Optimization
```sql
-- Use ST_DWithin instead of ST_Distance for filtering
-- Good: Uses spatial index
SELECT * FROM points
WHERE ST_DWithin(location, target_point, 1000);

-- Bad: Calculates distance for all rows
SELECT * FROM points
WHERE ST_Distance(location, target_point) < 1000;
```

### Batch Operations
```sql
-- Efficient bulk spatial operations
WITH buffered AS (
    SELECT id, ST_Buffer(geom, 100) as buffer
    FROM points
    WHERE active = true
)
UPDATE regions r
SET covered = true
FROM buffered b
WHERE ST_Intersects(r.boundary, b.buffer);
```

## Troubleshooting

### Common Issues

**Connection Refused**
```bash
# Check if container is running
docker ps | grep postgis

# Restart if needed
vrooli resource postgis start
```

**Extension Not Found**
```sql
-- Enable PostGIS in database
CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS postgis_raster;
```

**Slow Queries**
```sql
-- Check if spatial index exists
\d my_table

-- Analyze query plan
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM my_table WHERE ST_Within(geom, polygon);
```

## Best Practices

1. **Always use SRID**: Specify coordinate system explicitly
   ```sql
   ST_SetSRID(ST_MakePoint(lon, lat), 4326)
   ```

2. **Index spatial columns**: Create GIST indexes for performance
   ```sql
   CREATE INDEX ON table USING GIST(geom_column);
   ```

3. **Use geography for measurements**: More accurate distance/area
   ```sql
   ST_Distance(geom::geography, other::geography)
   ```

4. **Validate geometries**: Ensure data integrity
   ```sql
   ALTER TABLE my_table 
   ADD CONSTRAINT check_valid_geom 
   CHECK (ST_IsValid(geom));
   ```

5. **Simplify for visualization**: Reduce complexity for web display
   ```sql
   ST_Simplify(geom, 0.0001) -- tolerance in degrees
   ```