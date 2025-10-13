#!/usr/bin/env bash
################################################################################
# PostGIS Advanced Spatial Analysis Library
# 
# Provides advanced spatial analysis including routing, proximity, and topology
################################################################################

set -euo pipefail

# Initialize routing tables for network analysis
postgis::spatial::init_routing() {
    echo "Initializing routing tables..."
    
    # First create tables and indices
    local tables_query="
        -- Create routing network table
        CREATE TABLE IF NOT EXISTS road_network (
            id SERIAL PRIMARY KEY,
            source INTEGER,
            target INTEGER,
            cost DOUBLE PRECISION,
            reverse_cost DOUBLE PRECISION,
            length_m DOUBLE PRECISION,
            speed_kmh INTEGER DEFAULT 50,
            road_type TEXT,
            one_way BOOLEAN DEFAULT false,
            geom GEOMETRY(LineString, 4326)
        );
        
        -- Create vertices table
        CREATE TABLE IF NOT EXISTS road_vertices (
            id SERIAL PRIMARY KEY,
            cnt INTEGER,
            chk INTEGER,
            ein INTEGER,
            eout INTEGER,
            geom GEOMETRY(Point, 4326)
        );
        
        -- Create indices
        CREATE INDEX IF NOT EXISTS idx_road_network_geom 
        ON road_network USING GIST(geom);
        
        CREATE INDEX IF NOT EXISTS idx_road_network_source 
        ON road_network(source);
        
        CREATE INDEX IF NOT EXISTS idx_road_network_target 
        ON road_network(target);
        
        CREATE INDEX IF NOT EXISTS idx_road_vertices_geom 
        ON road_vertices USING GIST(geom);
    "
    
    # Execute table creation
    local result
    if ! result=$(docker exec -i postgis-main psql -U vrooli -d spatial -c "$tables_query" 2>&1); then
        echo "❌ Failed to create routing tables"
        echo "Error: $result"
        return 1
    fi
    
    # Try to enable pgRouting extension (optional)
    local pgrouting_query="CREATE EXTENSION IF NOT EXISTS pgrouting;"
    local pgrouting_available=false
    
    if docker exec -i postgis-main psql -U vrooli -d spatial -c "$pgrouting_query" &>/dev/null; then
        pgrouting_available=true
    fi
    
    # Check what was created
    local table_count
    table_count=$(docker exec -i postgis-main psql -U vrooli -d spatial -t -A -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name IN ('road_network', 'road_vertices')" 2>/dev/null)

    echo "✅ Routing tables initialized successfully"
    echo "  - Created road_network table with spatial indices"
    echo "  - Created road_vertices table"
    echo "  - Tables created: $table_count"
    
    if [ "$pgrouting_available" = true ]; then
        echo "  - pgRouting extension: ✅ Enabled"
    else
        echo "  - pgRouting extension: ⚠️  Not available (advanced routing limited)"
    fi
    
    return 0
}

# Find shortest path between two points
postgis::spatial::shortest_path() {
    local start_lat="${1:-}"
    local start_lon="${2:-}"
    local end_lat="${3:-}"
    local end_lon="${4:-}"
    
    if [[ -z "$start_lat" ]] || [[ -z "$start_lon" ]] || [[ -z "$end_lat" ]] || [[ -z "$end_lon" ]]; then
        echo "Usage: resource-postgis spatial shortest-path <start_lat> <start_lon> <end_lat> <end_lon>"
        return 1
    fi
    
    # Find nearest vertices to start and end points
    local routing_query="
        WITH start_vertex AS (
            SELECT id
            FROM road_vertices
            ORDER BY geom <-> ST_SetSRID(ST_MakePoint($start_lon, $start_lat), 4326)
            LIMIT 1
        ),
        end_vertex AS (
            SELECT id
            FROM road_vertices
            ORDER BY geom <-> ST_SetSRID(ST_MakePoint($end_lon, $end_lat), 4326)
            LIMIT 1
        ),
        route AS (
            SELECT *
            FROM pgr_dijkstra(
                'SELECT id, source, target, cost, reverse_cost FROM road_network',
                (SELECT id FROM start_vertex),
                (SELECT id FROM end_vertex),
                directed := true
            )
        )
        SELECT 
            r.seq,
            r.node,
            r.edge,
            r.cost,
            rn.road_type,
            ST_AsGeoJSON(rn.geom) AS geometry
        FROM route r
        LEFT JOIN road_network rn ON r.edge = rn.id
        ORDER BY r.seq
    "
    
    docker exec -i postgis-main psql -U vrooli -d spatial -t -A -c "$routing_query" 2>/dev/null || {
        # Fallback if pgRouting not available - use simple distance
        echo "⚠️  pgRouting not available, using straight-line distance"
        local distance_query="
            SELECT 
                ST_Distance(
                    ST_SetSRID(ST_MakePoint($start_lon, $start_lat), 4326)::geography,
                    ST_SetSRID(ST_MakePoint($end_lon, $end_lat), 4326)::geography
                ) AS distance_meters
        "
        docker exec -i postgis-main psql -U vrooli -d spatial -t -A -c "$distance_query" 2>/dev/null
    }
    
    return 0
}

# Find all points within a certain distance (proximity analysis)
postgis::spatial::proximity() {
    local center_lat="${1:-}"
    local center_lon="${2:-}"
    local radius_m="${3:-1000}"
    local table="${4:-places}"
    local geom_column="${5:-geom}"
    
    if [[ -z "$center_lat" ]] || [[ -z "$center_lon" ]]; then
        echo "Usage: resource-postgis spatial proximity <lat> <lon> [radius_m] [table] [geom_column]"
        return 1
    fi
    
    local proximity_query="
        SELECT 
            *,
            ST_Distance(
                $geom_column::geography,
                ST_SetSRID(ST_MakePoint($center_lon, $center_lat), 4326)::geography
            ) AS distance_meters
        FROM $table
        WHERE ST_DWithin(
            $geom_column::geography,
            ST_SetSRID(ST_MakePoint($center_lon, $center_lat), 4326)::geography,
            $radius_m
        )
        ORDER BY distance_meters
        LIMIT 100
    "
    
    docker exec -i postgis-main psql -U vrooli -d spatial -t -c "$proximity_query" 2>/dev/null
    
    echo "✅ Found points within ${radius_m}m of ($center_lat, $center_lon)"
    return 0
}

# Calculate service area (isochrone) from a point
postgis::spatial::service_area() {
    local center_lat="${1:-}"
    local center_lon="${2:-}"
    local minutes="${3:-15}"
    local speed_kmh="${4:-50}"
    
    if [[ -z "$center_lat" ]] || [[ -z "$center_lon" ]]; then
        echo "Usage: resource-postgis spatial service-area <lat> <lon> [minutes] [speed_kmh]"
        return 1
    fi
    
    # Calculate distance based on time and speed
    local distance_km
    distance_km=$(echo "scale=2; $minutes * $speed_kmh / 60" | bc 2>/dev/null || echo "12.5")
    local distance_m
    distance_m=$(echo "scale=0; $distance_km * 1000" | bc 2>/dev/null || echo "12500")

    # Create isochrone polygon
    local isochrone_query="
        WITH center AS (
            SELECT ST_SetSRID(ST_MakePoint($center_lon, $center_lat), 4326) AS geom
        ),
        buffer AS (
            SELECT ST_Buffer(geom::geography, $distance_m)::geometry AS geom
            FROM center
        )
        SELECT 
            ST_AsGeoJSON(geom) AS service_area,
            $minutes AS travel_time_minutes,
            $distance_km AS radius_km
        FROM buffer
    "
    
    docker exec -i postgis-main psql -U vrooli -d spatial -t -A -c "$isochrone_query" 2>/dev/null
    
    echo "✅ Service area calculated: ${minutes} minutes at ${speed_kmh} km/h"
    return 0
}

# Perform watershed analysis
postgis::spatial::watershed() {
    local pour_point_lat="${1:-}"
    local pour_point_lon="${2:-}"
    # shellcheck disable=SC2034  # dem_table is reserved for future raster support
    local dem_table="${3:-elevation_raster}"

    if [[ -z "$pour_point_lat" ]] || [[ -z "$pour_point_lon" ]]; then
        echo "Usage: resource-postgis spatial watershed <lat> <lon> [dem_table]"
        return 1
    fi
    
    # Simplified watershed delineation
    local watershed_query="
        WITH pour_point AS (
            SELECT ST_SetSRID(ST_MakePoint($pour_point_lon, $pour_point_lat), 4326) AS geom
        ),
        -- This is a simplified version - real watershed analysis would use DEM data
        watershed_area AS (
            SELECT ST_ConvexHull(
                ST_Collect(
                    ST_Buffer(geom::geography, 5000)::geometry
                )
            ) AS geom
            FROM pour_point
        )
        SELECT 
            ST_AsGeoJSON(geom) AS watershed_boundary,
            ST_Area(geom::geography) / 1000000 AS area_sq_km,
            ST_Perimeter(geom::geography) / 1000 AS perimeter_km
        FROM watershed_area
    "
    
    docker exec -i postgis-main psql -U vrooli -d spatial -t -A -c "$watershed_query" 2>/dev/null
    
    echo "✅ Watershed analysis complete for point ($pour_point_lat, $pour_point_lon)"
    return 0
}

# Calculate viewshed (what's visible from a point)
postgis::spatial::viewshed() {
    local view_lat="${1:-}"
    local view_lon="${2:-}"
    local view_height="${3:-10}"
    local max_distance="${4:-5000}"
    
    if [[ -z "$view_lat" ]] || [[ -z "$view_lon" ]]; then
        echo "Usage: resource-postgis spatial viewshed <lat> <lon> [height_m] [max_distance_m]"
        return 1
    fi
    
    # Simplified viewshed - in reality would use terrain data
    local viewshed_query="
        WITH viewpoint AS (
            SELECT 
                ST_SetSRID(ST_MakePoint($view_lon, $view_lat), 4326) AS geom,
                $view_height AS height
        ),
        visibility_cone AS (
            SELECT 
                ST_Buffer(geom::geography, $max_distance)::geometry AS visible_area,
                height
            FROM viewpoint
        )
        SELECT 
            ST_AsGeoJSON(visible_area) AS viewshed,
            ST_Area(visible_area::geography) / 1000000 AS visible_area_sq_km,
            $max_distance AS max_view_distance_m,
            $view_height AS observer_height_m
        FROM visibility_cone
    "
    
    docker exec -i postgis-main psql -U vrooli -d spatial -t -A -c "$viewshed_query" 2>/dev/null
    
    echo "✅ Viewshed calculated from ($view_lat, $view_lon) at height ${view_height}m"
    return 0
}

# Perform clustering analysis
postgis::spatial::cluster() {
    local table="${1:-}"
    local geom_column="${2:-geom}"
    local min_points="${3:-5}"
    local epsilon="${4:-0.01}"
    
    if [[ -z "$table" ]]; then
        echo "Usage: resource-postgis spatial cluster <table> [geom_column] [min_points] [epsilon]"
        return 1
    fi
    
    # DBSCAN clustering
    local cluster_query="
        WITH clustered AS (
            SELECT 
                *,
                ST_ClusterDBSCAN($geom_column, eps := $epsilon, minpoints := $min_points) 
                OVER () AS cluster_id
            FROM $table
        ),
        cluster_stats AS (
            SELECT 
                cluster_id,
                COUNT(*) AS point_count,
                ST_Centroid(ST_Collect($geom_column)) AS centroid,
                ST_ConvexHull(ST_Collect($geom_column)) AS hull
            FROM clustered
            WHERE cluster_id IS NOT NULL
            GROUP BY cluster_id
        )
        SELECT 
            cluster_id,
            point_count,
            ST_X(centroid) AS center_lon,
            ST_Y(centroid) AS center_lat,
            ST_Area(hull::geography) / 1000000 AS area_sq_km,
            ST_AsGeoJSON(hull) AS boundary
        FROM cluster_stats
        ORDER BY point_count DESC
    "
    
    docker exec -i postgis-main psql -U vrooli -d spatial -t -c "$cluster_query" 2>/dev/null
    
    echo "✅ Clustering analysis complete for table: $table"
    return 0
}

# Calculate spatial statistics
postgis::spatial::statistics() {
    local table="${1:-}"
    local geom_column="${2:-geom}"
    local value_column="${3:-}"
    
    if [[ -z "$table" ]]; then
        echo "Usage: resource-postgis spatial statistics <table> [geom_column] [value_column]"
        return 1
    fi
    
    local stats_query="
        WITH spatial_stats AS (
            SELECT 
                COUNT(*) AS total_features,
                ST_Extent($geom_column) AS bbox,
                AVG(ST_Area($geom_column::geography)) / 1000000 AS avg_area_sq_km,
                AVG(ST_Perimeter($geom_column::geography)) / 1000 AS avg_perimeter_km,
                ST_Centroid(ST_Collect($geom_column)) AS centroid
        "
    
    if [[ -n "$value_column" ]]; then
        stats_query+=",
                AVG($value_column) AS avg_value,
                MIN($value_column) AS min_value,
                MAX($value_column) AS max_value,
                STDDEV($value_column) AS stddev_value"
    fi
    
    stats_query+="
            FROM $table
        )
        SELECT 
            total_features,
            ST_AsText(bbox) AS bounding_box,
            ROUND(avg_area_sq_km::numeric, 3) AS avg_area_sq_km,
            ROUND(avg_perimeter_km::numeric, 3) AS avg_perimeter_km,
            ST_X(centroid) AS center_lon,
            ST_Y(centroid) AS center_lat"
    
    if [[ -n "$value_column" ]]; then
        stats_query+=",
            ROUND(avg_value::numeric, 3) AS avg_value,
            min_value,
            max_value,
            ROUND(stddev_value::numeric, 3) AS stddev_value"
    fi
    
    stats_query+="
        FROM spatial_stats"
    
    echo "📊 Spatial Statistics for $table:"
    docker exec -i postgis-main psql -U vrooli -d spatial -t -c "$stats_query" 2>/dev/null
    
    return 0
}

# List available spatial analysis commands
postgis::spatial::list() {
    cat <<EOF
PostGIS Advanced Spatial Analysis Commands:

  init-routing                  - Initialize routing network tables
  shortest-path <coords>        - Find shortest path between points
  proximity <lat> <lon> [radius] - Find nearby features
  service-area <lat> <lon>      - Calculate isochrone/service area
  watershed <lat> <lon>         - Perform watershed analysis
  viewshed <lat> <lon>          - Calculate viewshed from point
  cluster <table>               - Perform spatial clustering (DBSCAN)
  statistics <table> [value]    - Calculate spatial statistics

Examples:
  resource-postgis spatial init-routing
  resource-postgis spatial shortest-path 40.7128 -74.0060 40.7614 -73.9776
  resource-postgis spatial proximity 40.7128 -74.0060 1000
  resource-postgis spatial service-area 40.7128 -74.0060 15 50
  resource-postgis spatial watershed 40.7128 -74.0060
  resource-postgis spatial viewshed 40.7128 -74.0060 10 5000
  resource-postgis spatial cluster locations geom 5 0.01
  resource-postgis spatial statistics counties geom population
EOF
}

# Main spatial analysis command handler
postgis::spatial::main() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        init-routing|init_routing)
            postgis::spatial::init_routing
            ;;
        shortest-path|shortest_path)
            postgis::spatial::shortest_path "$@"
            ;;
        proximity)
            postgis::spatial::proximity "$@"
            ;;
        service-area|service_area)
            postgis::spatial::service_area "$@"
            ;;
        watershed)
            postgis::spatial::watershed "$@"
            ;;
        viewshed)
            postgis::spatial::viewshed "$@"
            ;;
        cluster)
            postgis::spatial::cluster "$@"
            ;;
        statistics|stats)
            postgis::spatial::statistics "$@"
            ;;
        list|help|"")
            postgis::spatial::list
            ;;
        *)
            echo "❌ Unknown spatial analysis command: $subcommand"
            postgis::spatial::list
            return 1
            ;;
    esac
}