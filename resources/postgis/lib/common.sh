#!/bin/bash
# PostGIS Common Functions

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC2034  # POSTGIS_LIB_DIR used by sourcing scripts
POSTGIS_LIB_DIR="${APP_ROOT}/resources/postgis/lib"
POSTGIS_ROOT_DIR="${APP_ROOT}/resources/postgis"

# Default container name
POSTGIS_CONTAINER="${POSTGIS_CONTAINER:-postgis-main}"

# Source dependencies
source "${POSTGIS_ROOT_DIR}/config/defaults.sh"

# Source shared utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC2154  # var_LOG_FILE is set by var.sh
source "${var_LOG_FILE}" 2>/dev/null || true

# Helper function to execute PostgreSQL commands via Docker
postgis_exec_sql() {
    local database="${1:-postgres}"
    local sql="$2"
    docker exec postgis-main psql -U "$POSTGIS_PG_USER" -d "$database" -c "$sql" -t -A 2>/dev/null
}

# Initialize data directories
postgis_init_dirs() {
    mkdir -p "$POSTGIS_DATA_DIR"
    mkdir -p "$POSTGIS_IMPORT_DIR"
    mkdir -p "$POSTGIS_EXPORT_DIR"
    mkdir -p "$POSTGIS_SQL_DIR"
}

# Check if PostgreSQL is available
postgis_check_postgres() {
    local result
    # Use docker exec to connect to postgis container
    result=$(docker exec postgis-main psql -U "$POSTGIS_PG_USER" -d postgres -c "SELECT 1" -t -A 2>/dev/null)
    [ "$result" = "1" ]
}

# Check if PostGIS is installed in PostgreSQL
postgis_is_installed() {
    local result
    # Use docker exec to connect to postgres container
    result=$(docker exec vrooli-postgres-main psql -U "$POSTGIS_PG_USER" -d postgres -c "SELECT 1 FROM pg_available_extensions WHERE name='postgis'" -t -A 2>/dev/null)
    [ "$result" = "1" ]
}

# Check if PostGIS is enabled in a database
postgis_is_enabled() {
    local database="${1:-$POSTGIS_PG_DATABASE}"
    local result
    # Use docker exec to connect to postgres container
    result=$(docker exec vrooli-postgres-main psql -U "$POSTGIS_PG_USER" -d "$database" -c "SELECT 1 FROM pg_extension WHERE extname='postgis'" -t -A 2>/dev/null)
    [ "$result" = "1" ]
}

# Get PostGIS version
postgis_get_version() {
    local database="${1:-$POSTGIS_PG_DATABASE}"
    if postgis_is_enabled "$database"; then
        postgis_exec_sql "$database" "SELECT PostGIS_Version()" | head -1
    else
        echo "not_enabled"
    fi
}

# Enable PostGIS in a database
postgis_enable_database() {
    local database="${1:-$POSTGIS_PG_DATABASE}"
    
    log::header "Enabling PostGIS in database: $database"
    
    # Create database if it doesn't exist
    postgis_exec_sql "postgres" "CREATE DATABASE $database" 2>/dev/null
    
    # Enable extensions
    for ext in $POSTGIS_EXTENSIONS; do
        log::info "Installing extension: $ext"
        if postgis_exec_sql "$database" "CREATE EXTENSION IF NOT EXISTS $ext CASCADE"; then
            log::success "Extension $ext installed successfully"
        else
            log::error "Failed to install extension $ext"
            return 1
        fi
    done
    
    log::success "PostGIS enabled in database $database"
    return 0
}

# Disable PostGIS in a database
postgis_disable_database() {
    local database="${1:-$POSTGIS_PG_DATABASE}"
    
    log::header "Disabling PostGIS in database: $database"
    
    for ext in $(echo "$POSTGIS_EXTENSIONS" | tr ' ' '\n' | tac); do
        log::info "Removing extension: $ext"
        postgis_exec_sql "$database" "DROP EXTENSION IF EXISTS $ext CASCADE"
    done
    
    log::success "PostGIS disabled in database $database"
    return 0
}

# Execute SQL file with PostGIS functions
postgis_execute_sql() {
    local sql_file="$1"
    local database="${2:-$POSTGIS_PG_DATABASE}"
    
    if [ ! -f "$sql_file" ]; then
        log::error "SQL file not found: $sql_file"
        return 1
    fi
    
    log::info "Executing SQL file: $sql_file"
    # For standalone PostGIS, use the spatial database as default
    if docker ps --format "{{.Names}}" | grep -q "^${POSTGIS_CONTAINER}$"; then
        # Use spatial database for standalone PostGIS container
        local exec_database="${database}"
        if [ "$exec_database" = "vrooli" ]; then
            exec_database="spatial"
        fi
        # Copy SQL file to container and execute
        if docker cp "$sql_file" "${POSTGIS_CONTAINER}:/tmp/inject.sql" 2>/dev/null && \
           docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d "$exec_database" -f /tmp/inject.sql 2>&1; then
            log::success "SQL file executed successfully"
            return 0
        else
            log::error "Failed to execute SQL file"
            return 1
        fi
    else
        log::error "PostGIS container not running"
        return 1
    fi
}

# Import shapefile with enhanced ogr2ogr support
postgis_import_shapefile() {
    local shapefile="$1"
    local database="${2:-$POSTGIS_PG_DATABASE}"
    local table_name="${3:-$(basename "$shapefile" .shp)}"
    
    if [ ! -f "$shapefile" ]; then
        log::error "Shapefile not found: $shapefile"
        return 1
    fi
    
    log::info "Importing shapefile: $shapefile"
    
    # Try ogr2ogr first (better format support)
    if command -v ogr2ogr >/dev/null 2>&1; then
        local conn_string="PG:host=$POSTGIS_PG_HOST port=$POSTGIS_PG_PORT dbname=$database user=$POSTGIS_PG_USER password=$POSTGIS_PG_PASSWORD"
        if ogr2ogr -f "PostgreSQL" "$conn_string" "$shapefile" -nln "$table_name" -overwrite -lco GEOMETRY_NAME=geom -lco FID=id -lco SPATIAL_INDEX=GIST 2>&1; then
            log::success "Shapefile imported to table: $table_name (using ogr2ogr)"
            return 0
        fi
    fi
    
    # Fallback to shp2pgsql
    if command -v shp2pgsql >/dev/null 2>&1; then
        shp2pgsql -I -s "$POSTGIS_DEFAULT_SRID" "$shapefile" "$table_name" | \
            PGPASSWORD="$POSTGIS_PG_PASSWORD" psql -h "$POSTGIS_PG_HOST" -p "$POSTGIS_PG_PORT" -U "$POSTGIS_PG_USER" -d "$database" 2>&1
        log::success "Shapefile imported to table: $table_name (using shp2pgsql)"
        return 0
    else
        log::error "Neither ogr2ogr nor shp2pgsql found. Install gdal-bin or postgis-client package."
        return 1
    fi
}

# Import GeoJSON file
postgis_import_geojson() {
    local geojson_file="$1"
    local database="${2:-$POSTGIS_PG_DATABASE}"
    local table_name="${3:-$(basename "$geojson_file" .geojson | sed 's/\.json$//')}"
    
    if [ ! -f "$geojson_file" ]; then
        log::error "GeoJSON file not found: $geojson_file"
        return 1
    fi
    
    log::info "Importing GeoJSON: $geojson_file"
    
    # Use ogr2ogr from container
    local container="${POSTGIS_CONTAINER:-postgis-main}"
    
    # Check if ogr2ogr is available in container
    if docker exec "$container" which ogr2ogr >/dev/null 2>&1; then
        # Copy file to container
        local container_path
        container_path="/tmp/$(basename "$geojson_file")"
        if ! docker cp "$geojson_file" "$container:$container_path" 2>/dev/null; then
            log::error "Failed to copy GeoJSON file to container"
            return 1
        fi
        
        # Import using containerized ogr2ogr
        local conn_string="PG:host=localhost port=5432 dbname=$database user=$POSTGIS_PG_USER password=$POSTGIS_PG_PASSWORD"
        if docker exec "$container" ogr2ogr -f "PostgreSQL" "$conn_string" "$container_path" -nln "$table_name" -overwrite -lco GEOMETRY_NAME=geom -lco FID=id -lco SPATIAL_INDEX=GIST 2>&1; then
            log::success "GeoJSON imported to table: $table_name"
            # Clean up
            docker exec "$container" rm -f "$container_path" 2>/dev/null || true
            return 0
        else
            log::error "Failed to import GeoJSON using ogr2ogr"
            docker exec "$container" rm -f "$container_path" 2>/dev/null || true
            return 1
        fi
    else
        log::error "ogr2ogr not found in container. Rebuild with custom Dockerfile for GeoJSON support."
        return 1
    fi
}

# Import KML/KMZ file
postgis_import_kml() {
    local kml_file="$1"
    local database="${2:-$POSTGIS_PG_DATABASE}"
    local table_name="${3:-$(basename "$kml_file" .kml | sed 's/\.kmz$//')}"
    
    if [ ! -f "$kml_file" ]; then
        log::error "KML/KMZ file not found: $kml_file"
        return 1
    fi
    
    log::info "Importing KML/KMZ: $kml_file"
    
    # Use ogr2ogr for KML import
    if command -v ogr2ogr >/dev/null 2>&1; then
        local conn_string="PG:host=$POSTGIS_PG_HOST port=$POSTGIS_PG_PORT dbname=$database user=$POSTGIS_PG_USER password=$POSTGIS_PG_PASSWORD"
        if ogr2ogr -f "PostgreSQL" "$conn_string" "$kml_file" -nln "$table_name" -overwrite -lco GEOMETRY_NAME=geom -lco FID=id -lco SPATIAL_INDEX=GIST 2>&1; then
            log::success "KML/KMZ imported to table: $table_name"
            return 0
        else
            log::error "Failed to import KML/KMZ using ogr2ogr"
            return 1
        fi
    else
        log::error "ogr2ogr not found. Install gdal-bin package for KML support."
        return 1
    fi
}

# Universal GIS format importer
postgis_import_gis() {
    local input_file="$1"
    local database="${2:-$POSTGIS_PG_DATABASE}"
    local table_name="${3:-}"
    
    if [ ! -f "$input_file" ]; then
        log::error "Input file not found: $input_file"
        return 1
    fi
    
    # Detect file type and call appropriate importer
    local extension="${input_file##*.}"
    extension="${extension,,}" # Convert to lowercase
    
    # Generate table name if not provided
    if [ -z "$table_name" ]; then
        table_name="$(basename "$input_file" ".$extension" | sed 's/[^a-zA-Z0-9_]/_/g')"
    fi
    
    case "$extension" in
        shp)
            postgis_import_shapefile "$input_file" "$database" "$table_name"
            ;;
        geojson|json)
            postgis_import_geojson "$input_file" "$database" "$table_name"
            ;;
        kml|kmz)
            postgis_import_kml "$input_file" "$database" "$table_name"
            ;;
        gpx)
            # GPX files (GPS tracks)
            log::info "Importing GPX: $input_file"
            if command -v ogr2ogr >/dev/null 2>&1; then
                local conn_string="PG:host=$POSTGIS_PG_HOST port=$POSTGIS_PG_PORT dbname=$database user=$POSTGIS_PG_USER password=$POSTGIS_PG_PASSWORD"
                ogr2ogr -f "PostgreSQL" "$conn_string" "$input_file" -nln "$table_name" -overwrite -lco GEOMETRY_NAME=geom -lco FID=id -lco SPATIAL_INDEX=GIST 2>&1
                log::success "GPX imported to table: $table_name"
            else
                log::error "ogr2ogr not found. Install gdal-bin package."
                return 1
            fi
            ;;
        csv)
            # CSV with coordinates
            log::info "Importing CSV with coordinates: $input_file"
            if command -v ogr2ogr >/dev/null 2>&1; then
                local conn_string="PG:host=$POSTGIS_PG_HOST port=$POSTGIS_PG_PORT dbname=$database user=$POSTGIS_PG_USER password=$POSTGIS_PG_PASSWORD"
                # Assume X,Y or lon,lat columns
                ogr2ogr -f "PostgreSQL" "$conn_string" "$input_file" -nln "$table_name" -overwrite -oo X_POSSIBLE_NAMES=lon,longitude,x -oo Y_POSSIBLE_NAMES=lat,latitude,y -lco GEOMETRY_NAME=geom -lco FID=id 2>&1
                log::success "CSV imported to table: $table_name"
            else
                log::error "ogr2ogr not found. Install gdal-bin package."
                return 1
            fi
            ;;
        *)
            log::error "Unsupported file format: .$extension"
            echo "Supported formats: shp, geojson, json, kml, kmz, gpx, csv"
            return 1
            ;;
    esac
}

# Export table to shapefile
postgis_export_shapefile() {
    local table_name="$1"
    local output_file="$2"
    local database="${3:-$POSTGIS_PG_DATABASE}"
    
    if [ -z "$output_file" ]; then
        output_file="$POSTGIS_EXPORT_DIR/${table_name}.shp"
    fi
    
    format_info "Exporting table $table_name to shapefile"
    
    if command -v pgsql2shp >/dev/null 2>&1; then
        PGPASSWORD="$POSTGIS_PG_PASSWORD" pgsql2shp -f "$output_file" \
            -h "$POSTGIS_PG_HOST" -p "$POSTGIS_PG_PORT" -u "$POSTGIS_PG_USER" \
            "$database" "$table_name" 2>&1
        format_success "Table exported to: $output_file"
        return 0
    else
        format_error "pgsql2shp not found. Install postgis-client package."
        return 1
    fi
}