#!/usr/bin/env bash
################################################################################
# PostGIS Geocoding Library
# 
# Provides geocoding and reverse geocoding capabilities
################################################################################

set -euo pipefail

# Initialize geocoding tables if they don't exist
postgis::geocoding::init() {
    echo "Initializing geocoding tables..."
    
    local init_query="
        -- Create geocoding cache table
        CREATE TABLE IF NOT EXISTS geocoding_cache (
            id SERIAL PRIMARY KEY,
            address TEXT UNIQUE NOT NULL,
            lat DOUBLE PRECISION,
            lon DOUBLE PRECISION,
            geom GEOMETRY(Point, 4326),
            confidence FLOAT,
            source TEXT,
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
        
        -- Create spatial index
        CREATE INDEX IF NOT EXISTS idx_geocoding_cache_geom 
        ON geocoding_cache USING GIST(geom);
        
        -- Create address index
        CREATE INDEX IF NOT EXISTS idx_geocoding_cache_address 
        ON geocoding_cache(address);
        
        -- Create places table for reverse geocoding
        CREATE TABLE IF NOT EXISTS places (
            id SERIAL PRIMARY KEY,
            name TEXT NOT NULL,
            type TEXT,
            address TEXT,
            city TEXT,
            state TEXT,
            country TEXT,
            postal_code TEXT,
            geom GEOMETRY(Point, 4326),
            boundary GEOMETRY(Polygon, 4326),
            created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
        );
        
        -- Create spatial indices
        CREATE INDEX IF NOT EXISTS idx_places_geom 
        ON places USING GIST(geom);
        
        CREATE INDEX IF NOT EXISTS idx_places_boundary 
        ON places USING GIST(boundary);
    "
    
    docker exec -i postgis-main psql -U vrooli -d spatial -c "$init_query" >/dev/null 2>&1
    
    echo "✅ Geocoding tables initialized"
    return 0
}

# Geocode an address to coordinates
postgis::geocoding::geocode() {
    local address="${1:-}"
    local use_cache="${2:-true}"
    
    if [[ -z "$address" ]]; then
        echo "Usage: resource-postgis geocoding geocode <address> [use_cache]"
        return 1
    fi
    
    # Check cache first
    if [[ "$use_cache" == "true" ]]; then
        local cache_query="
            SELECT lat, lon, confidence, source
            FROM geocoding_cache
            WHERE address = '$address'
            LIMIT 1
        "
        
        local cached_result
        cached_result=$(docker exec -i postgis-main psql -U vrooli -d spatial -t -A -c "$cache_query" 2>/dev/null)
        
        if [[ -n "$cached_result" ]]; then
            echo "✅ Found in cache: $cached_result"
            return 0
        fi
    fi
    
    # If not in cache, use Nominatim-style geocoding
    # For demo purposes, we'll use a simple pattern matching approach
    # In production, this would integrate with a real geocoding service
    
    # Extract potential coordinates from address if they're embedded
    local lat_pattern='(-?[0-9]+\.?[0-9]*)'
    local lon_pattern='(-?[0-9]+\.?[0-9]*)'
    
    if [[ "$address" =~ ${lat_pattern}[,\ ]+${lon_pattern} ]]; then
        local lat="${BASH_REMATCH[1]}"
        local lon="${BASH_REMATCH[2]}"
        
        # Store in cache
        local insert_query="
            INSERT INTO geocoding_cache (address, lat, lon, geom, confidence, source)
            VALUES ('$address', $lat, $lon, ST_SetSRID(ST_MakePoint($lon, $lat), 4326), 1.0, 'coordinates')
            ON CONFLICT (address) DO NOTHING
            RETURNING lat, lon, confidence
        "
        
        docker exec -i postgis-main psql -U vrooli -d spatial -t -A -c "$insert_query" 2>/dev/null
        
        echo "✅ Geocoded: lat=$lat, lon=$lon (confidence=1.0)"
        return 0
    fi
    
    # Try to match against known places
    local place_query="
        SELECT 
            ST_Y(geom) AS lat,
            ST_X(geom) AS lon,
            similarity(address, '$address') AS confidence
        FROM places
        WHERE address % '$address'  -- Uses trigram similarity
        ORDER BY confidence DESC
        LIMIT 1
    "
    
    local place_result
    place_result=$(docker exec -i postgis-main psql -U vrooli -d spatial -t -A -c "$place_query" 2>/dev/null)
    
    if [[ -n "$place_result" ]]; then
        echo "✅ Matched place: $place_result"
        
        # Cache the result
        IFS='|' read -r lat lon confidence <<< "$place_result"
        local cache_insert="
            INSERT INTO geocoding_cache (address, lat, lon, geom, confidence, source)
            VALUES ('$address', $lat, $lon, ST_SetSRID(ST_MakePoint($lon, $lat), 4326), $confidence, 'places')
            ON CONFLICT (address) DO NOTHING
        "
        docker exec -i postgis-main psql -U vrooli -d spatial -c "$cache_insert" >/dev/null 2>&1
        
        return 0
    fi
    
    echo "❌ Unable to geocode address: $address"
    return 1
}

# Reverse geocode coordinates to address
postgis::geocoding::reverse() {
    local lat="${1:-}"
    local lon="${2:-}"
    local radius="${3:-1000}"  # Search radius in meters
    
    if [[ -z "$lat" ]] || [[ -z "$lon" ]]; then
        echo "Usage: resource-postgis geocoding reverse <lat> <lon> [radius_meters]"
        return 1
    fi
    
    # Find nearest place
    local reverse_query="
        SELECT 
            name,
            address,
            city,
            state,
            country,
            postal_code,
            ST_Distance(
                geom::geography,
                ST_SetSRID(ST_MakePoint($lon, $lat), 4326)::geography
            ) AS distance_meters
        FROM places
        WHERE ST_DWithin(
            geom::geography,
            ST_SetSRID(ST_MakePoint($lon, $lat), 4326)::geography,
            $radius
        )
        ORDER BY distance_meters
        LIMIT 1
    "
    
    local result
    result=$(docker exec -i postgis-main psql -U vrooli -d spatial -t -A -c "$reverse_query" 2>/dev/null)
    
    if [[ -n "$result" ]]; then
        echo "✅ Reverse geocoded: $result"
        return 0
    fi
    
    # If no exact place found, generate approximate address
    local approx_query="
        SELECT 
            'Approximately ' || ROUND(ST_Distance(
                geom::geography,
                ST_SetSRID(ST_MakePoint($lon, $lat), 4326)::geography
            )) || ' meters from ' || name AS description
        FROM places
        ORDER BY geom <-> ST_SetSRID(ST_MakePoint($lon, $lat), 4326)
        LIMIT 1
    "
    
    local approx_result
    approx_result=$(docker exec -i postgis-main psql -U vrooli -d spatial -t -A -c "$approx_query" 2>/dev/null)
    
    if [[ -n "$approx_result" ]]; then
        echo "⚠️  Approximate location: $approx_result"
        return 0
    fi
    
    echo "❌ No address found for coordinates: $lat, $lon"
    return 1
}

# Batch geocode multiple addresses
postgis::geocoding::batch() {
    local input_file="${1:-}"
    local output_file="${2:-/tmp/geocoded_results.csv}"
    
    if [[ -z "$input_file" ]] || [[ ! -f "$input_file" ]]; then
        echo "Usage: resource-postgis geocoding batch <input_file> [output_file]"
        echo "Input file should contain one address per line"
        return 1
    fi
    
    echo "address,lat,lon,confidence" > "$output_file"
    
    local total
    total=$(wc -l < "$input_file")
    local current=0
    
    while IFS= read -r address; do
        ((current++))
        echo "Processing $current/$total: $address"
        
        # Geocode each address
        local result=$(postgis::geocoding::geocode "$address" true 2>/dev/null | grep "✅" | sed 's/.*: //')
        
        if [[ -n "$result" ]]; then
            echo "\"$address\",$result" >> "$output_file"
        else
            echo "\"$address\",,,0" >> "$output_file"
        fi
    done < "$input_file"
    
    echo "✅ Batch geocoding complete: $output_file"
    return 0
}

# Import places data for geocoding
postgis::geocoding::import_places() {
    local csv_file="${1:-}"
    
    if [[ -z "$csv_file" ]] || [[ ! -f "$csv_file" ]]; then
        echo "Usage: resource-postgis geocoding import <csv_file>"
        echo "CSV should have columns: name,address,city,state,country,postal_code,lat,lon"
        return 1
    fi
    
    # Copy CSV to container and import
    docker cp "$csv_file" postgis-main:/tmp/places.csv
    
    local import_query="
        COPY places (name, address, city, state, country, postal_code, geom)
        FROM '/tmp/places.csv'
        WITH (FORMAT csv, HEADER true);
        
        UPDATE places 
        SET geom = ST_SetSRID(ST_MakePoint(lon, lat), 4326)
        WHERE geom IS NULL AND lat IS NOT NULL AND lon IS NOT NULL;
    "
    
    docker exec -i postgis-main psql -U vrooli -d spatial -c "$import_query" 2>/dev/null
    
    local count=$(docker exec -i postgis-main psql -U vrooli -d spatial -t -A -c "SELECT COUNT(*) FROM places" 2>/dev/null)
    
    echo "✅ Imported places. Total in database: $count"
    return 0
}

# Show geocoding statistics
postgis::geocoding::stats() {
    local stats_query="
        SELECT 
            'Cache Entries' AS metric,
            COUNT(*) AS value
        FROM geocoding_cache
        UNION ALL
        SELECT 
            'Places' AS metric,
            COUNT(*) AS value
        FROM places
        UNION ALL
        SELECT 
            'Average Confidence' AS metric,
            ROUND(AVG(confidence)::numeric, 3) AS value
        FROM geocoding_cache
        WHERE confidence IS NOT NULL
        UNION ALL
        SELECT 
            'Cache Hit Rate' AS metric,
            ROUND(
                COUNT(CASE WHEN source = 'cache' THEN 1 END)::numeric / 
                NULLIF(COUNT(*), 0) * 100, 1
            ) AS value
        FROM geocoding_cache
    "
    
    echo "📊 Geocoding Statistics:"
    docker exec -i postgis-main psql -U vrooli -d spatial -t -c "$stats_query" 2>/dev/null
    return 0
}

# List available geocoding commands
postgis::geocoding::list() {
    cat <<EOF
PostGIS Geocoding Commands:

  init                          - Initialize geocoding tables
  geocode <address> [use_cache] - Convert address to coordinates
  reverse <lat> <lon> [radius]  - Convert coordinates to address
  batch <input_file> [output]   - Batch geocode addresses from file
  import <csv_file>             - Import places data for geocoding
  stats                         - Show geocoding statistics

Examples:
  resource-postgis geocoding init
  resource-postgis geocoding geocode "1600 Pennsylvania Ave, Washington DC"
  resource-postgis geocoding reverse 38.8977 -77.0365
  resource-postgis geocoding batch addresses.txt results.csv
  resource-postgis geocoding import places.csv
EOF
}

# Main geocoding command handler
postgis::geocoding::main() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        init)
            postgis::geocoding::init
            ;;
        geocode)
            postgis::geocoding::geocode "$@"
            ;;
        reverse)
            postgis::geocoding::reverse "$@"
            ;;
        batch)
            postgis::geocoding::batch "$@"
            ;;
        import|import_places)
            postgis::geocoding::import_places "$@"
            ;;
        stats)
            postgis::geocoding::stats
            ;;
        list|help|"")
            postgis::geocoding::list
            ;;
        *)
            echo "❌ Unknown geocoding command: $subcommand"
            postgis::geocoding::list
            return 1
            ;;
    esac
}