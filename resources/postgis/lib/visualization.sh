#!/usr/bin/env bash
################################################################################
# PostGIS Visualization Library
# 
# Provides map visualization capabilities for spatial data
################################################################################

set -euo pipefail

# Generate GeoJSON from a SQL query
postgis::visualization::generate_geojson() {
    local query="${1:-}"
    local output_file="${2:-/tmp/postgis_map.geojson}"
    
    if [[ -z "$query" ]]; then
        echo "Usage: resource-postgis visualization geojson <query> [output_file]"
        return 1
    fi
    
    # Construct GeoJSON query
    local geojson_query="
        SELECT json_build_object(
            'type', 'FeatureCollection',
            'features', json_agg(
                json_build_object(
                    'type', 'Feature',
                    'geometry', ST_AsGeoJSON(geom)::json,
                    'properties', row_to_json(properties)
                )
            )
        ) AS geojson
        FROM (
            $query
        ) AS properties
    "
    
    # Execute query and save to file
    docker exec -i postgis-main psql -U vrooli -d spatial -t -A -c "$geojson_query" > "$output_file" 2>/dev/null
    
    if [[ -f "$output_file" ]]; then
        echo "✅ GeoJSON generated: $output_file"
        echo "Size: $(wc -c < "$output_file") bytes"
        return 0
    else
        echo "❌ Failed to generate GeoJSON"
        return 1
    fi
}

# Generate a heat map from point data
postgis::visualization::generate_heatmap() {
    local table="${1:-}"
    local geom_column="${2:-geom}"
    local output_file="${3:-/tmp/postgis_heatmap.json}"
    
    if [[ -z "$table" ]]; then
        echo "Usage: resource-postgis visualization heatmap <table> [geom_column] [output_file]"
        return 1
    fi
    
    # Generate hexagonal bins for heat map
    local heatmap_query="
        WITH hexagons AS (
            SELECT 
                ST_HexagonGrid(0.01, ST_Extent($geom_column)) AS hex,
                COUNT(*) AS intensity
            FROM $table
            GROUP BY hex
        )
        SELECT json_build_object(
            'type', 'FeatureCollection',
            'features', json_agg(
                json_build_object(
                    'type', 'Feature',
                    'geometry', ST_AsGeoJSON(hex)::json,
                    'properties', json_build_object('intensity', intensity)
                )
            )
        ) AS heatmap
        FROM hexagons
        WHERE intensity > 0
    "
    
    # Execute query
    docker exec -i postgis-main psql -U vrooli -d spatial -t -A -c "$heatmap_query" > "$output_file" 2>/dev/null
    
    if [[ -f "$output_file" ]]; then
        echo "✅ Heat map generated: $output_file"
        return 0
    else
        echo "❌ Failed to generate heat map"
        return 1
    fi
}

# Generate a choropleth map (thematic map with colored regions)
postgis::visualization::generate_choropleth() {
    local table="${1:-}"
    local value_column="${2:-}"
    local geom_column="${3:-geom}"
    local output_file="${4:-/tmp/postgis_choropleth.json}"
    
    if [[ -z "$table" ]] || [[ -z "$value_column" ]]; then
        echo "Usage: resource-postgis visualization choropleth <table> <value_column> [geom_column] [output_file]"
        return 1
    fi
    
    # Calculate quartiles and assign colors
    local choropleth_query="
        WITH stats AS (
            SELECT 
                percentile_cont(0.25) WITHIN GROUP (ORDER BY $value_column) AS q1,
                percentile_cont(0.5) WITHIN GROUP (ORDER BY $value_column) AS q2,
                percentile_cont(0.75) WITHIN GROUP (ORDER BY $value_column) AS q3
            FROM $table
        ),
        colored AS (
            SELECT 
                *,
                CASE 
                    WHEN $value_column <= (SELECT q1 FROM stats) THEN '#ffffcc'
                    WHEN $value_column <= (SELECT q2 FROM stats) THEN '#a1dab4'
                    WHEN $value_column <= (SELECT q3 FROM stats) THEN '#41b6c4'
                    ELSE '#0c2c84'
                END AS color
            FROM $table
        )
        SELECT json_build_object(
            'type', 'FeatureCollection',
            'features', json_agg(
                json_build_object(
                    'type', 'Feature',
                    'geometry', ST_AsGeoJSON($geom_column)::json,
                    'properties', json_build_object(
                        'value', $value_column,
                        'color', color
                    )
                )
            )
        ) AS choropleth
        FROM colored
    "
    
    # Execute query
    docker exec -i postgis-main psql -U vrooli -d spatial -t -A -c "$choropleth_query" > "$output_file" 2>/dev/null
    
    if [[ -f "$output_file" ]]; then
        echo "✅ Choropleth map generated: $output_file"
        return 0
    else
        echo "❌ Failed to generate choropleth map"
        return 1
    fi
}

# Generate map tiles in MVT (Mapbox Vector Tile) format
postgis::visualization::generate_tiles() {
    local table="${1:-}"
    local zoom="${2:-10}"
    local x="${3:-}"
    local y="${4:-}"
    local geom_column="${5:-geom}"
    
    if [[ -z "$table" ]] || [[ -z "$x" ]] || [[ -z "$y" ]]; then
        echo "Usage: resource-postgis visualization tiles <table> <zoom> <x> <y> [geom_column]"
        return 1
    fi
    
    # Generate MVT tile
    local tile_query="
        SELECT ST_AsMVT(q, '$table', 4096, 'geom')
        FROM (
            SELECT 
                ST_AsMVTGeom(
                    ST_Transform($geom_column, 3857),
                    ST_TileEnvelope($zoom, $x, $y),
                    4096, 256, true
                ) AS geom,
                *
            FROM $table
            WHERE ST_Intersects(
                $geom_column,
                ST_Transform(ST_TileEnvelope($zoom, $x, $y), 4326)
            )
        ) AS q
    "
    
    # Execute and return tile
    docker exec -i postgis-main psql -U vrooli -d spatial -t -A -c "$tile_query" 2>/dev/null
    
    echo "✅ Tile generated for zoom=$zoom, x=$x, y=$y"
    return 0
}

# Generate a simple HTML map viewer
postgis::visualization::generate_viewer() {
    local geojson_file="${1:-/tmp/postgis_map.geojson}"
    local output_file="${2:-/tmp/postgis_map.html}"
    
    if [[ ! -f "$geojson_file" ]]; then
        echo "❌ GeoJSON file not found: $geojson_file"
        return 1
    fi
    
    # Create a simple Leaflet-based map viewer
    cat > "$output_file" <<'EOF'
<!DOCTYPE html>
<html>
<head>
    <title>PostGIS Spatial Data Viewer</title>
    <meta charset="utf-8" />
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <link rel="stylesheet" href="https://unpkg.com/leaflet@1.9.4/dist/leaflet.css" />
    <script src="https://unpkg.com/leaflet@1.9.4/dist/leaflet.js"></script>
    <style>
        body { margin: 0; padding: 0; }
        #map { position: absolute; top: 0; bottom: 0; width: 100%; }
        .info {
            padding: 6px 8px;
            background: white;
            background: rgba(255,255,255,0.9);
            box-shadow: 0 0 15px rgba(0,0,0,0.2);
            border-radius: 5px;
            position: absolute;
            top: 10px;
            right: 10px;
            z-index: 1000;
        }
    </style>
</head>
<body>
    <div id="map"></div>
    <div class="info">
        <h4>PostGIS Visualization</h4>
        <p>Interactive map of spatial data</p>
    </div>
    <script>
        var map = L.map('map').setView([0, 0], 2);
        
        L.tileLayer('https://{s}.tile.openstreetmap.org/{z}/{x}/{y}.png', {
            attribution: '© OpenStreetMap contributors'
        }).addTo(map);
        
        // Load GeoJSON data
        fetch('GEOJSON_FILE')
            .then(response => response.json())
            .then(data => {
                var geoJsonLayer = L.geoJSON(data, {
                    style: function(feature) {
                        return {
                            color: feature.properties.color || '#3388ff',
                            weight: 2,
                            opacity: 0.7,
                            fillOpacity: 0.5
                        };
                    },
                    onEachFeature: function(feature, layer) {
                        if (feature.properties) {
                            var popupContent = '<h4>Properties</h4><pre>' + 
                                JSON.stringify(feature.properties, null, 2) + '</pre>';
                            layer.bindPopup(popupContent);
                        }
                    }
                }).addTo(map);
                
                // Fit map to GeoJSON bounds
                map.fitBounds(geoJsonLayer.getBounds());
            })
            .catch(error => console.error('Error loading GeoJSON:', error));
    </script>
</body>
</html>
EOF
    
    # Replace placeholder with actual GeoJSON file path
    sed -i "s|GEOJSON_FILE|file://$geojson_file|g" "$output_file"
    
    echo "✅ Map viewer generated: $output_file"
    echo "Open in browser: file://$output_file"
    return 0
}

# List available visualization commands
postgis::visualization::list() {
    cat <<EOF
PostGIS Visualization Commands:

  geojson <query> [output]     - Generate GeoJSON from SQL query
  heatmap <table> [geom] [out]  - Generate heat map from point data
  choropleth <table> <value>    - Generate choropleth (colored regions) map
  tiles <table> <z> <x> <y>     - Generate map tiles (MVT format)
  viewer [geojson] [output]     - Generate HTML map viewer

Examples:
  resource-postgis visualization geojson "SELECT name, population, geom FROM cities"
  resource-postgis visualization heatmap crime_data location
  resource-postgis visualization choropleth counties population geom
  resource-postgis visualization tiles roads 10 512 340
  resource-postgis visualization viewer /tmp/map.geojson /tmp/map.html
EOF
}

# Main visualization command handler
postgis::visualization::main() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        geojson)
            postgis::visualization::generate_geojson "$@"
            ;;
        heatmap)
            postgis::visualization::generate_heatmap "$@"
            ;;
        choropleth)
            postgis::visualization::generate_choropleth "$@"
            ;;
        tiles)
            postgis::visualization::generate_tiles "$@"
            ;;
        viewer)
            postgis::visualization::generate_viewer "$@"
            ;;
        list|help|"")
            postgis::visualization::list
            ;;
        *)
            echo "❌ Unknown visualization command: $subcommand"
            postgis::visualization::list
            return 1
            ;;
    esac
}