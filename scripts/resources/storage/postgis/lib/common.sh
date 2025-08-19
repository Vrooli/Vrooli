#!/bin/bash
# PostGIS Common Functions

# Get script directory
POSTGIS_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
POSTGIS_ROOT_DIR="$(dirname "$POSTGIS_LIB_DIR")"

# Source dependencies
source "${POSTGIS_ROOT_DIR}/config/defaults.sh"

# Source shared utilities
SCRIPTS_DIR="$(dirname "$(dirname "$(dirname "$POSTGIS_ROOT_DIR")")")"
if [ -f "${SCRIPTS_DIR}/lib/utils/format.sh" ]; then
    source "${SCRIPTS_DIR}/lib/utils/format.sh"
fi
if [ -f "${SCRIPTS_DIR}/lib/utils/var.sh" ]; then
    source "${SCRIPTS_DIR}/lib/utils/var.sh"
fi
if [ -f "${SCRIPTS_DIR}/lib/utils/log.sh" ]; then
    source "${SCRIPTS_DIR}/lib/utils/log.sh"
fi

# Helper function to execute PostgreSQL commands via Docker
postgis_exec_sql() {
    local database="${1:-postgres}"
    local sql="$2"
    docker exec vrooli-postgres-main psql -U "$POSTGIS_PG_USER" -d "$database" -c "$sql" -t -A 2>/dev/null
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
    # Use docker exec to connect to postgres container
    result=$(docker exec vrooli-postgres-main psql -U "$POSTGIS_PG_USER" -d postgres -c "SELECT 1" -t -A 2>/dev/null)
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
        format_error "SQL file not found: $sql_file"
        return 1
    fi
    
    format_info "Executing SQL file: $sql_file"
    if PGPASSWORD="$POSTGIS_PG_PASSWORD" psql -h "$POSTGIS_PG_HOST" -p "$POSTGIS_PG_PORT" -U "$POSTGIS_PG_USER" -d "$database" -f "$sql_file" 2>&1; then
        format_success "SQL file executed successfully"
        return 0
    else
        format_error "Failed to execute SQL file"
        return 1
    fi
}

# Import shapefile
postgis_import_shapefile() {
    local shapefile="$1"
    local database="${2:-$POSTGIS_PG_DATABASE}"
    local table_name="${3:-$(basename "$shapefile" .shp)}"
    
    if [ ! -f "$shapefile" ]; then
        format_error "Shapefile not found: $shapefile"
        return 1
    fi
    
    format_info "Importing shapefile: $shapefile"
    
    # Use shp2pgsql to convert and import
    if command -v shp2pgsql >/dev/null 2>&1; then
        shp2pgsql -I -s "$POSTGIS_DEFAULT_SRID" "$shapefile" "$table_name" | \
            PGPASSWORD="$POSTGIS_PG_PASSWORD" psql -h "$POSTGIS_PG_HOST" -p "$POSTGIS_PG_PORT" -U "$POSTGIS_PG_USER" -d "$database" 2>&1
        format_success "Shapefile imported to table: $table_name"
        return 0
    else
        format_error "shp2pgsql not found. Install postgis-client package."
        return 1
    fi
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