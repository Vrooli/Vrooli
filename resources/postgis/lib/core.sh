#!/bin/bash
# PostGIS Core Functions - Docker Management Wrappers

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
POSTGIS_CORE_LIB_DIR="${APP_ROOT}/resources/postgis/lib"

# Source dependencies
source "${POSTGIS_CORE_LIB_DIR}/common.sh"
source "${POSTGIS_CORE_LIB_DIR}/install.sh"

# Docker wrapper functions for v2.0 compliance

postgis::docker::start() {
    postgis_start "$@"
}

postgis::docker::stop() {
    postgis_stop "$@"
}

postgis::docker::restart() {
    log::info "Restarting PostGIS container..."
    postgis_stop && sleep 2 && postgis_start
}

postgis::docker::logs() {
    local container="${POSTGIS_CONTAINER:-postgis-main}"
    if docker ps -a --format "{{.Names}}" | grep -q "^${container}$"; then
        docker logs "$container" "$@"
    else
        log::error "PostGIS container not found: $container"
        return 1
    fi
}

# Install wrapper
postgis::install::execute() {
    postgis_install "$@"
}

postgis::install::uninstall() {
    postgis_uninstall "$@"
}

# Status wrappers
postgis::status() {
    postgis_status "$@"
}

postgis::status::check() {
    # Smoke test - just check if container is healthy
    local container="${POSTGIS_CONTAINER:-postgis-main}"
    if docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        if docker exec "${container}" pg_isready -U vrooli -d spatial &>/dev/null; then
            log::success "PostGIS is running and accessible"
            return 0
        else
            log::error "PostGIS container running but not accessible"
            return 1
        fi
    else
        log::error "PostGIS container not running"
        return 1
    fi
}

# Test wrapper
postgis::test::smoke() {
    postgis::status::check "$@"
}

postgis::test::integration() {
    postgis_run_tests "$@"
}

# Content functions for spatial data management
postgis::content::add() {
    local source_path="$1"
    shift
    
    if [[ -z "$source_path" ]]; then
        log::error "Usage: content add <sql_file|shapefile|directory> [database]"
        return 1
    fi
    
    postgis_inject "$source_path" "$@"
}

postgis::content::list() {
    local database="${1:-spatial}"
    
    log::header "Spatial Tables in PostGIS"
    
    local container="${POSTGIS_CONTAINER:-postgis-main}"
    if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log::error "PostGIS container not running"
        return 1
    fi
    
    # List tables with geometry columns
    docker exec "${container}" psql -U vrooli -d "$database" \
        -c "SELECT schemaname, tablename, 
            pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
            FROM pg_tables 
            WHERE schemaname = 'public' 
            ORDER BY tablename;" 2>/dev/null || {
        log::error "Failed to list spatial tables"
        return 1
    }
}

postgis::content::get() {
    local table_name="$1"
    local database="${2:-spatial}"
    
    if [[ -z "$table_name" ]]; then
        log::error "Usage: content get <table_name> [database]"
        return 1
    fi
    
    local container="${POSTGIS_CONTAINER:-postgis-main}"
    if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log::error "PostGIS container not running"
        return 1
    fi
    
    log::info "Table structure for: $table_name"
    docker exec "${container}" psql -U vrooli -d "$database" \
        -c "\d+ $table_name" 2>/dev/null || {
        log::error "Failed to describe table: $table_name"
        return 1
    }
}

postgis::content::remove() {
    local table_name="$1"
    local database="${2:-spatial}"
    
    if [[ -z "$table_name" ]]; then
        log::error "Usage: content remove <table_name> [database]"
        return 1
    fi
    
    local container="${POSTGIS_CONTAINER:-postgis-main}"
    if ! docker ps --format "{{.Names}}" | grep -q "^${container}$"; then
        log::error "PostGIS container not running"
        return 1
    fi
    
    log::warning "Removing spatial table: $table_name"
    read -p "Are you sure? (y/N): " -r
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        if docker exec "${container}" psql -U vrooli -d "$database" \
            -c "DROP TABLE IF EXISTS $table_name CASCADE;" 2>/dev/null; then
            log::success "Table removed: $table_name"
        else
            log::error "Failed to remove table: $table_name"
            return 1
        fi
    else
        log::info "Operation cancelled"
    fi
}

postgis::content::execute() {
    local sql_file="$1"
    local database="${2:-spatial}"
    
    if [[ -z "$sql_file" ]]; then
        log::error "Usage: content execute <sql_file> [database]"
        return 1
    fi
    
    postgis_execute_sql "$sql_file" "$database"
}

# Show example spatial queries (from original CLI)
postgis_show_examples() {
    log::header "PostGIS Example Queries"
    
    cat <<'EOF'

# Find all locations within 10km of a point:
SELECT name, ST_Distance(location, ST_MakePoint(-74.006, 40.7128)) AS distance
FROM locations
WHERE ST_DWithin(location, ST_MakePoint(-74.006, 40.7128), 10000)
ORDER BY distance;

# Calculate area of a polygon:
SELECT ST_Area(ST_GeomFromText('POLYGON((0 0, 0 1, 1 1, 1 0, 0 0))'));

# Find nearest neighbor:
SELECT name, location <-> ST_MakePoint(-74.006, 40.7128) AS distance
FROM locations
ORDER BY location <-> ST_MakePoint(-74.006, 40.7128)
LIMIT 5;

# Create a buffer around a point:
SELECT ST_Buffer(ST_MakePoint(-74.006, 40.7128)::geography, 1000);

# Check if point is within polygon:
SELECT ST_Within(
    ST_MakePoint(-74.006, 40.7128),
    ST_GeomFromText('POLYGON((-75 40, -75 41, -73 41, -73 40, -75 40))')
);

# Calculate length of a line:
SELECT ST_Length(ST_GeomFromText('LINESTRING(0 0, 1 1, 2 1)'));

# Union multiple geometries:
SELECT ST_Union(geom) FROM boundaries GROUP BY region;

# Simplify geometry (reduce points):
SELECT ST_Simplify(geom, 0.001) FROM complex_shapes;

EOF
    
    log::info "To run these examples, use: resource-postgis content execute <sql_file>"
}

# Export functions
export -f postgis::docker::start
export -f postgis::docker::stop  
export -f postgis::docker::restart
export -f postgis::docker::logs
export -f postgis::install::execute
export -f postgis::install::uninstall
export -f postgis::status
export -f postgis::status::check
export -f postgis::test::smoke
export -f postgis::test::integration
export -f postgis::content::add
export -f postgis::content::list
export -f postgis::content::get
export -f postgis::content::remove
export -f postgis::content::execute
export -f postgis_show_examples