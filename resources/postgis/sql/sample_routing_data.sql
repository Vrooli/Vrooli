-- Sample road network data for testing pgRouting
-- This creates a simple grid network for demonstration

-- Clear existing data
TRUNCATE road_network, road_vertices CASCADE;

-- Insert sample road segments (grid pattern)
INSERT INTO road_network (source, target, cost, reverse_cost, length_m, speed_kmh, road_type, one_way, geom)
VALUES
-- Horizontal roads
(1, 2, 100, 100, 100, 50, 'local', false, ST_GeomFromText('LINESTRING(-73.99 40.75, -73.98 40.75)', 4326)),
(2, 3, 100, 100, 100, 50, 'local', false, ST_GeomFromText('LINESTRING(-73.98 40.75, -73.97 40.75)', 4326)),
(3, 4, 100, 100, 100, 50, 'local', false, ST_GeomFromText('LINESTRING(-73.97 40.75, -73.96 40.75)', 4326)),

(5, 6, 100, 100, 100, 50, 'local', false, ST_GeomFromText('LINESTRING(-73.99 40.76, -73.98 40.76)', 4326)),
(6, 7, 100, 100, 100, 50, 'local', false, ST_GeomFromText('LINESTRING(-73.98 40.76, -73.97 40.76)', 4326)),
(7, 8, 100, 100, 100, 50, 'local', false, ST_GeomFromText('LINESTRING(-73.97 40.76, -73.96 40.76)', 4326)),

-- Vertical roads
(1, 5, 110, 110, 110, 50, 'local', false, ST_GeomFromText('LINESTRING(-73.99 40.75, -73.99 40.76)', 4326)),
(2, 6, 110, 110, 110, 50, 'local', false, ST_GeomFromText('LINESTRING(-73.98 40.75, -73.98 40.76)', 4326)),
(3, 7, 110, 110, 110, 50, 'local', false, ST_GeomFromText('LINESTRING(-73.97 40.75, -73.97 40.76)', 4326)),
(4, 8, 110, 110, 110, 50, 'local', false, ST_GeomFromText('LINESTRING(-73.96 40.75, -73.96 40.76)', 4326)),

-- Diagonal shortcut (one-way)
(1, 7, 150, -1, 150, 60, 'highway', true, ST_GeomFromText('LINESTRING(-73.99 40.75, -73.97 40.76)', 4326));

-- Insert vertices
INSERT INTO road_vertices (id, geom)
VALUES
(1, ST_GeomFromText('POINT(-73.99 40.75)', 4326)),
(2, ST_GeomFromText('POINT(-73.98 40.75)', 4326)),
(3, ST_GeomFromText('POINT(-73.97 40.75)', 4326)),
(4, ST_GeomFromText('POINT(-73.96 40.75)', 4326)),
(5, ST_GeomFromText('POINT(-73.99 40.76)', 4326)),
(6, ST_GeomFromText('POINT(-73.98 40.76)', 4326)),
(7, ST_GeomFromText('POINT(-73.97 40.76)', 4326)),
(8, ST_GeomFromText('POINT(-73.96 40.76)', 4326));

-- Analyze the topology
SELECT pgr_analyzeGraph('road_network', 0.000001, 'geom', 'id');

-- Create topology for pgRouting
SELECT pgr_createTopology('road_network', 0.000001, 'geom', 'id');

-- Test routing without function
-- Shows shortest path from vertex 1 to vertex 8

-- Test direct routing query
SELECT 'Sample routing from vertex 1 to vertex 8:' as info;
SELECT seq, node, edge, cost, agg_cost FROM pgr_dijkstra(
    'SELECT id, source, target, cost FROM road_network',
    1,  -- Start from vertex 1
    8,  -- End at vertex 8
    false  -- Undirected
);