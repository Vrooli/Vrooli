#!/bin/bash
# PostGIS Cross-Resource Integration Functions

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/*}/../.." && builtin pwd)}"
POSTGIS_INTEGRATION_LIB_DIR="${APP_ROOT}/resources/postgis/lib"

# Source dependencies
source "${POSTGIS_INTEGRATION_LIB_DIR}/common.sh"

# PostGIS configuration
POSTGIS_CONTAINER="postgis-main"
POSTGIS_PORT="${POSTGIS_PORT:-5434}"

#######################################
# Export spatial data for n8n workflow integration
# Args: table_name output_format
# Returns: Export path or error
#######################################
postgis::integration::export_for_n8n() {
    local table_name="$1"
    local format="${2:-geojson}"
    
    if [[ -z "$table_name" ]]; then
        log::error "Usage: postgis integration export-n8n <table_name> [format]"
        return 1
    fi
    
    local container="${POSTGIS_CONTAINER}"
    if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log::error "PostGIS container not running"
        return 1
    fi
    
    local export_dir="${POSTGIS_DATA_DIR}/exports"
    mkdir -p "$export_dir"
    
    local export_file="${export_dir}/${table_name}_$(date +%Y%m%d_%H%M%S).${format}"
    
    log::info "Exporting $table_name to $format format for n8n..."
    
    case "$format" in
        geojson)
            # Export as GeoJSON
            docker exec "${container}" psql -U vrooli -d spatial -t <<EOF > "$export_file"
SELECT json_build_object(
    'type', 'FeatureCollection',
    'features', json_agg(
        json_build_object(
            'type', 'Feature',
            'geometry', ST_AsGeoJSON(geom)::json,
            'properties', row_to_json((SELECT d FROM (SELECT * EXCEPT(geom)) d))
        )
    )
)
FROM $table_name;
EOF
            ;;
        csv)
            # Export as CSV with WKT geometry
            docker exec "${container}" psql -U vrooli -d spatial <<EOF > "$export_file"
\COPY (SELECT *, ST_AsText(geom) as geometry_wkt FROM $table_name) TO STDOUT WITH CSV HEADER
EOF
            ;;
        *)
            log::error "Unsupported format: $format (use geojson or csv)"
            return 1
            ;;
    esac
    
    if [[ -f "$export_file" ]]; then
        log::success "Data exported to: $export_file"
        log::info "n8n integration URL: postgresql://vrooli:vrooli@host.docker.internal:${POSTGIS_PORT}/spatial"
        echo "$export_file"
        return 0
    else
        log::error "Export failed"
        return 1
    fi
}

#######################################
# Create Ollama-compatible embeddings table
# Returns: 0 on success
#######################################
postgis::integration::create_ollama_embeddings_table() {
    local container="${POSTGIS_CONTAINER}"
    if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log::error "PostGIS container not running"
        return 1
    fi
    
    log::header "Creating Ollama Embeddings Table"
    
    docker exec "${container}" psql -U vrooli -d spatial <<'EOF'
-- Create table for storing Ollama embeddings with spatial context
CREATE TABLE IF NOT EXISTS ollama_spatial_embeddings (
    id SERIAL PRIMARY KEY,
    content TEXT NOT NULL,
    embedding vector(768),  -- Assuming 768-dim embeddings
    location GEOGRAPHY(Point, 4326),
    metadata JSONB,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Create indexes for efficient similarity search
CREATE INDEX IF NOT EXISTS idx_ollama_embeddings_vector 
ON ollama_spatial_embeddings USING ivfflat (embedding vector_cosine_ops);

CREATE INDEX IF NOT EXISTS idx_ollama_embeddings_location 
ON ollama_spatial_embeddings USING GIST (location);

-- Create function for combined semantic + spatial search
CREATE OR REPLACE FUNCTION search_embeddings_spatial(
    query_embedding vector(768),
    query_location geography,
    max_distance_meters float DEFAULT 10000,
    limit_results int DEFAULT 10
)
RETURNS TABLE(
    id int,
    content text,
    similarity float,
    distance_meters float
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        e.id,
        e.content,
        1 - (e.embedding <=> query_embedding) as similarity,
        ST_Distance(e.location, query_location) as distance_meters
    FROM ollama_spatial_embeddings e
    WHERE ST_DWithin(e.location, query_location, max_distance_meters)
    ORDER BY e.embedding <=> query_embedding
    LIMIT limit_results;
END;
$$ LANGUAGE plpgsql;
EOF
    
    log::success "Ollama embeddings table created with spatial support"
    log::info "Use search_embeddings_spatial() function for combined semantic + spatial search"
    return 0
}

#######################################
# Setup QuestDB time-series integration
# Returns: Connection info
#######################################
postgis::integration::setup_questdb_sync() {
    local container="${POSTGIS_CONTAINER}"
    if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log::error "PostGIS container not running"
        return 1
    fi
    
    log::header "Setting up QuestDB Integration"
    
    # Create foreign data wrapper for QuestDB
    docker exec "${container}" psql -U vrooli -d spatial <<'EOF'
-- Enable postgres_fdw extension
CREATE EXTENSION IF NOT EXISTS postgres_fdw;

-- Create time-series export table
CREATE TABLE IF NOT EXISTS spatial_timeseries (
    timestamp TIMESTAMP NOT NULL,
    sensor_id VARCHAR(100),
    location GEOGRAPHY(Point, 4326),
    value DOUBLE PRECISION,
    metadata JSONB,
    PRIMARY KEY (timestamp, sensor_id)
);

-- Create function to export to QuestDB format
CREATE OR REPLACE FUNCTION export_to_questdb_format(
    table_name text,
    time_column text DEFAULT 'timestamp',
    value_column text DEFAULT 'value'
)
RETURNS TABLE(csv_line text) AS $$
BEGIN
    RETURN QUERY
    EXECUTE format(
        'SELECT %I || '','' || ST_Y(location::geometry) || '','' || 
                ST_X(location::geometry) || '','' || %I::text
         FROM %I
         ORDER BY %I',
        time_column, value_column, table_name, time_column
    );
END;
$$ LANGUAGE plpgsql;

-- Create materialized view for efficient time-series queries
CREATE MATERIALIZED VIEW IF NOT EXISTS spatial_timeseries_hourly AS
SELECT 
    date_trunc('hour', timestamp) as hour,
    sensor_id,
    location,
    AVG(value) as avg_value,
    MIN(value) as min_value,
    MAX(value) as max_value,
    COUNT(*) as sample_count
FROM spatial_timeseries
GROUP BY date_trunc('hour', timestamp), sensor_id, location;

CREATE INDEX IF NOT EXISTS idx_timeseries_hourly_time 
ON spatial_timeseries_hourly(hour);
EOF
    
    log::success "QuestDB integration tables created"
    log::info "Export data using: SELECT * FROM export_to_questdb_format('table_name');"
    return 0
}

#######################################
# Create Redis-compatible spatial cache
# Returns: 0 on success
#######################################
postgis::integration::create_redis_cache_tables() {
    local container="${POSTGIS_CONTAINER}"
    if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log::error "PostGIS container not running"
        return 1
    fi
    
    log::header "Creating Redis Cache Integration Tables"
    
    docker exec "${container}" psql -U vrooli -d spatial <<'EOF'
-- Create table for caching spatial query results
CREATE TABLE IF NOT EXISTS spatial_cache (
    cache_key VARCHAR(255) PRIMARY KEY,
    query_hash VARCHAR(64) NOT NULL,
    result_geojson TEXT,
    bbox GEOMETRY(Polygon, 4326),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    hit_count INTEGER DEFAULT 0
);

-- Create indexes for cache management
CREATE INDEX IF NOT EXISTS idx_spatial_cache_expires 
ON spatial_cache(expires_at);

CREATE INDEX IF NOT EXISTS idx_spatial_cache_bbox 
ON spatial_cache USING GIST(bbox);

-- Function to generate cache key from query
CREATE OR REPLACE FUNCTION generate_cache_key(
    query_text text,
    params jsonb DEFAULT '{}'
)
RETURNS VARCHAR(255) AS $$
DECLARE
    cache_key VARCHAR(255);
BEGIN
    cache_key := md5(query_text || params::text);
    RETURN cache_key;
END;
$$ LANGUAGE plpgsql;

-- Function to get cached result
CREATE OR REPLACE FUNCTION get_cached_spatial_result(
    p_cache_key VARCHAR(255)
)
RETURNS TEXT AS $$
DECLARE
    cached_result TEXT;
BEGIN
    -- Check if cache entry exists and is not expired
    SELECT result_geojson INTO cached_result
    FROM spatial_cache
    WHERE cache_key = p_cache_key
    AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP);
    
    -- Update hit count if found
    IF cached_result IS NOT NULL THEN
        UPDATE spatial_cache 
        SET hit_count = hit_count + 1
        WHERE cache_key = p_cache_key;
    END IF;
    
    RETURN cached_result;
END;
$$ LANGUAGE plpgsql;

-- Function to cache spatial query result
CREATE OR REPLACE FUNCTION cache_spatial_result(
    p_cache_key VARCHAR(255),
    p_query_hash VARCHAR(64),
    p_result TEXT,
    p_ttl_seconds INTEGER DEFAULT 3600
)
RETURNS VOID AS $$
BEGIN
    INSERT INTO spatial_cache (cache_key, query_hash, result_geojson, expires_at)
    VALUES (p_cache_key, p_query_hash, p_result, CURRENT_TIMESTAMP + (p_ttl_seconds || ' seconds')::INTERVAL)
    ON CONFLICT (cache_key) DO UPDATE
    SET result_geojson = EXCLUDED.result_geojson,
        expires_at = EXCLUDED.expires_at,
        created_at = CURRENT_TIMESTAMP;
END;
$$ LANGUAGE plpgsql;
EOF
    
    log::success "Redis-compatible cache tables created"
    log::info "Use cache functions for query result caching"
    return 0
}

#######################################
# Show integration status and examples
# Returns: Integration information
#######################################
postgis::integration::status() {
    log::header "PostGIS Integration Status"
    
    local container="${POSTGIS_CONTAINER}"
    if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log::error "PostGIS container not running"
        return 1
    fi
    
    log::info "Available integrations:"
    echo
    echo "1. n8n Workflow Integration"
    echo "   Connection: postgresql://vrooli:vrooli@host.docker.internal:${POSTGIS_PORT}/spatial"
    echo "   Export: vrooli resource postgis integration export-n8n <table>"
    echo
    echo "2. Ollama Embeddings"
    echo "   Setup: vrooli resource postgis integration setup-ollama"
    echo "   Table: ollama_spatial_embeddings"
    echo
    echo "3. QuestDB Time-Series"
    echo "   Setup: vrooli resource postgis integration setup-questdb"
    echo "   Table: spatial_timeseries"
    echo
    echo "4. Redis Cache"
    echo "   Setup: vrooli resource postgis integration setup-redis"
    echo "   Table: spatial_cache"
    echo
    
    # Check which integrations are set up
    log::info "Checking installed integrations..."
    
    local tables=$(docker exec "${container}" psql -U vrooli -d spatial -t -c "
        SELECT tablename FROM pg_tables 
        WHERE schemaname = 'public' 
        AND tablename IN ('ollama_spatial_embeddings', 'spatial_timeseries', 'spatial_cache')
        ORDER BY tablename;" 2>/dev/null)
    
    if [[ -n "$tables" ]]; then
        echo "Installed integration tables:"
        echo "$tables" | while read -r table; do
            [[ -n "$table" ]] && echo "  ✅ $table"
        done
    else
        echo "No integration tables found. Run setup commands to enable integrations."
    fi
    
    return 0
}

# Export functions
export -f postgis::integration::export_for_n8n
export -f postgis::integration::create_ollama_embeddings_table
export -f postgis::integration::setup_questdb_sync
export -f postgis::integration::create_redis_cache_tables
export -f postgis::integration::status