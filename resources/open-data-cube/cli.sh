#!/bin/bash

# Open Data Cube Resource CLI
# Provides Earth observation data cube management and analysis

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "${SCRIPT_DIR}/lib/core.sh"
source "${SCRIPT_DIR}/lib/test.sh"

# Main command router
main() {
    local command="${1:-help}"
    shift || true

    case "$command" in
        help)
            show_help
            ;;
        info)
            show_info
            ;;
        manage)
            handle_manage "$@"
            ;;
        test)
            handle_test "$@"
            ;;
        content)
            handle_content "$@"
            ;;
        status)
            show_status "$@"
            ;;
        logs)
            show_logs "$@"
            ;;
        credentials)
            show_credentials "$@"
            ;;
        dataset)
            handle_dataset "$@"
            ;;
        query)
            handle_query "$@"
            ;;
        export)
            handle_export "$@"
            ;;
        *)
            echo "Unknown command: $command"
            show_help
            exit 1
            ;;
    esac
}

# Show help information
show_help() {
    cat << EOF
Open Data Cube Resource - Earth Observation Analytics Platform

Usage: resource-open-data-cube [COMMAND] [OPTIONS]

Commands:
  help                Show this help message
  info                Show runtime configuration
  manage              Lifecycle management
    install           Install ODC dependencies
    start             Start ODC stack
    stop              Stop ODC stack
    restart           Restart all services
    uninstall         Remove ODC installation
  test                Run validation tests
    smoke             Quick health validation (<30s)
    integration       Full functionality test (<120s)
    unit              Test library functions (<60s)
    all               Run all test phases
  content             Manage datasets
    list              List available datasets
    add <path>        Add dataset from file
    remove <id>       Remove dataset
    get <id>          Get dataset details
  dataset             Dataset operations
    index <path>      Index new dataset
    list              List indexed datasets
    search <query>    Search datasets
  query               Query data cube
    area <geojson>    Query by area
    time <range>      Query by time range
    product <name>    Query by product type
  export              Export query results
    geotiff <path>    Export to GeoTIFF
    geojson <path>    Export to GeoJSON
    netcdf <path>     Export to NetCDF
  status              Show ODC status
  logs                View ODC logs
  credentials         Display connection details

Examples:
  # Start ODC stack
  resource-open-data-cube manage start

  # Index Sentinel-2 dataset
  resource-open-data-cube dataset index /data/sentinel2/

  # Query specific area and time
  resource-open-data-cube query area '{"type":"Polygon","coordinates":[...]}'
  
  # Export results to GeoTIFF
  resource-open-data-cube export geotiff output.tif

EOF
}

# Execute main function
main "$@"