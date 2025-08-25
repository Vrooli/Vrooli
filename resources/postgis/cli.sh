#!/bin/bash
# PostGIS Resource CLI

set -euo pipefail

# Get the real script location (resolving symlinks)
SCRIPT_PATH="$(readlink -f "${BASH_SOURCE[0]}")"
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
POSTGIS_CLI_DIR="${APP_ROOT}/resources/postgis"

# Source libraries
source "${POSTGIS_CLI_DIR}/lib/common.sh"
source "${POSTGIS_CLI_DIR}/lib/status.sh"
source "${POSTGIS_CLI_DIR}/lib/install.sh"
source "${POSTGIS_CLI_DIR}/lib/inject.sh"
source "${POSTGIS_CLI_DIR}/lib/test.sh"

# CLI usage
usage() {
    cat <<EOF
PostGIS Resource CLI

Usage: resource-postgis [COMMAND] [OPTIONS]

Commands:
    status                    Show PostGIS status
    install                   Install and enable PostGIS
    uninstall                Disable PostGIS in databases
    start                     Enable PostGIS (alias for install)
    stop                      Disable PostGIS (alias for uninstall)
    test                      Run integration tests
    enable-database DB        Enable PostGIS in specific database
    disable-database DB       Disable PostGIS in specific database
    inject FILE [DB]          Execute SQL file with PostGIS functions
    import-shapefile FILE     Import shapefile to PostGIS
    export-shapefile TABLE    Export PostGIS table to shapefile
    examples                  Show example spatial queries
    help                      Show this help message

Examples:
    resource-postgis status
    resource-postgis enable-database myapp_db
    resource-postgis inject /path/to/spatial_queries.sql
    resource-postgis import-shapefile /data/cities.shp
    
EOF
}

# Show example queries
show_examples() {
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
    
    log::info "To run these examples, use: resource-postgis inject <sql_file>"
}

# Main command handler
main() {
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        status)
            postgis_status "$@"
            ;;
        install|start)
            postgis_install
            ;;
        uninstall|stop)
            postgis_uninstall
            ;;
        enable-database)
            postgis_enable_database "$@"
            ;;
        disable-database)
            postgis_disable_database "$@"
            ;;
        inject)
            postgis_execute_sql "$@"
            ;;
        import-shapefile)
            postgis_import_shapefile "$@"
            ;;
        export-shapefile)
            postgis_export_shapefile "$@"
            ;;
        test)
            postgis_run_tests
            ;;
        examples)
            show_examples
            ;;
        help|--help|-h)
            usage
            ;;
        *)
            log::error "Unknown command: $command"
            usage
            exit 1
            ;;
    esac
}

# Run main function
main "$@"
