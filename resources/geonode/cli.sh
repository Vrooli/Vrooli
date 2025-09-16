#!/bin/bash

# GeoNode Resource CLI
# Provides geospatial content management through GeoNode stack

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
        import)
            handle_import "$@"
            ;;
        export)
            handle_export "$@"
            ;;
        metadata)
            handle_metadata "$@"
            ;;
        config)
            handle_config "$@"
            ;;
        validate)
            validate_file "$@"
            ;;
        optimize)
            handle_optimize "$@"
            ;;
        webhook)
            handle_webhook "$@"
            ;;
        stats)
            show_stats
            ;;
        *)
            echo "Unknown command: $command" >&2
            show_help
            exit 1
            ;;
    esac
}

show_help() {
    cat << EOF
GeoNode Resource - Geospatial Content Management System

USAGE:
    vrooli resource geonode <command> [options]

COMMANDS:
    help            Show this help message
    info            Show resource information and configuration
    manage          Lifecycle management (install/start/stop/restart/uninstall)
    test            Run tests (smoke/integration/unit/all)
    content         Content management (add-layer/list-layers/get-layer/remove-layer)
    status          Show service status
    logs            View service logs
    credentials     Display API credentials
    import          Import spatial data files
    export          Export layers and maps
    metadata        Manage metadata
    config          Configure integrations
    validate        Validate spatial data files
    optimize        Performance optimization
    webhook         Webhook management
    stats           Show resource statistics

LIFECYCLE:
    manage install              Install dependencies
    manage start [--wait]       Start GeoNode stack
    manage stop                 Stop all services
    manage restart              Restart all services
    manage uninstall           Remove GeoNode and data

CONTENT:
    content add-layer <file>            Upload spatial dataset
    content list-layers                 List all layers
    content get-layer <name>            Get layer details
    content remove-layer <name>         Delete layer
    content create-map                  Create new map
    content list-maps                   List all maps
    content export-map <name>           Export map configuration

IMPORT/EXPORT:
    import shapefile <file>             Import shapefile
    import geotiff <file>               Import raster data
    import geojson <file>               Import GeoJSON
    export layer <name> <format>        Export layer
    export map <name>                   Export map config

EXAMPLES:
    vrooli resource geonode manage start --wait
    vrooli resource geonode content add-layer cities.shp
    vrooli resource geonode import geotiff satellite.tif
    vrooli resource geonode export layer cities geojson

For detailed documentation, see: resources/geonode/README.md
EOF
}

show_info() {
    load_config
    
    cat << EOF
GeoNode Resource Information
============================
Version: 4.4.3
Stack: Django + GeoServer + PostGIS + Redis
Status: $(get_status)

Configuration:
- Django Port: ${GEONODE_PORT}
- GeoServer Port: ${GEONODE_GEOSERVER_PORT}
- Database: ${GEONODE_DB_NAME:-geonode}
- Admin User: ${GEONODE_ADMIN_USER:-admin}

Services:
- Web Portal: http://localhost:${GEONODE_PORT}
- GeoServer: http://localhost:${GEONODE_GEOSERVER_PORT}/geoserver
- REST API: http://localhost:${GEONODE_PORT}/api/

Features:
- Multi-format spatial data support
- OGC-compliant map services
- RESTful API for automation
- Metadata catalog
- User management and permissions

Integration Ready:
- PostGIS for spatial database
- MinIO for object storage
- QuestDB for timeseries
- Keycloak for authentication
EOF
}

handle_manage() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        install)
            install_geonode "$@"
            ;;
        start)
            start_geonode "$@"
            ;;
        stop)
            stop_geonode "$@"
            ;;
        restart)
            restart_geonode "$@"
            ;;
        uninstall)
            uninstall_geonode "$@"
            ;;
        *)
            echo "Usage: manage [install|start|stop|restart|uninstall]" >&2
            exit 1
            ;;
    esac
}

handle_content() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        add|add-layer)
            add_layer "$@"
            ;;
        list|list-layers)
            list_layers "$@"
            ;;
        get|get-layer)
            get_layer "$@"
            ;;
        remove|remove-layer)
            remove_layer "$@"
            ;;
        execute)
            # Execute for GeoNode means process spatial data
            if [[ -n "$1" ]]; then
                import_data "auto" "$@"
            else
                echo "Usage: content execute <file>" >&2
                exit 1
            fi
            ;;
        create-map)
            create_map "$@"
            ;;
        list-maps)
            list_maps "$@"
            ;;
        export-map)
            export_map "$@"
            ;;
        *)
            echo "Usage: content [add|list|get|remove|execute|add-layer|list-layers|get-layer|remove-layer|create-map|list-maps|export-map]" >&2
            echo "" >&2
            echo "Standard v2.0 commands:" >&2
            echo "  add <file>      Add spatial layer" >&2
            echo "  list            List all layers" >&2
            echo "  get <name>      Get layer details" >&2
            echo "  remove <name>   Remove layer" >&2
            echo "  execute <file>  Auto-detect format and import" >&2
            exit 1
            ;;
    esac
}

handle_import() {
    local format="${1:-}"
    local file="${2:-}"
    
    if [[ -z "$format" || -z "$file" ]]; then
        echo "Usage: import <format> <file>" >&2
        echo "Formats: shapefile, geotiff, geojson, kml" >&2
        exit 1
    fi
    
    import_data "$format" "$file"
}

handle_export() {
    local type="${1:-}"
    local name="${2:-}"
    local format="${3:-}"
    
    case "$type" in
        layer)
            if [[ -z "$name" || -z "$format" ]]; then
                echo "Usage: export layer <name> <format>" >&2
                exit 1
            fi
            export_layer "$name" "$format"
            ;;
        map)
            if [[ -z "$name" ]]; then
                echo "Usage: export map <name>" >&2
                exit 1
            fi
            export_map "$name"
            ;;
        *)
            echo "Usage: export [layer|map] ..." >&2
            exit 1
            ;;
    esac
}

handle_metadata() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        update)
            update_metadata "$@"
            ;;
        search)
            search_metadata "$@"
            ;;
        *)
            echo "Usage: metadata [update|search]" >&2
            exit 1
            ;;
    esac
}

handle_config() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        set-postgres)
            set_postgres_config "$@"
            ;;
        set-storage)
            set_storage_config "$@"
            ;;
        show)
            show_config
            ;;
        *)
            echo "Usage: config [set-postgres|set-storage|show]" >&2
            exit 1
            ;;
    esac
}

handle_optimize() {
    local target="${1:-}"
    
    case "$target" in
        cache)
            optimize_cache
            ;;
        indexes)
            optimize_indexes
            ;;
        *)
            echo "Usage: optimize [cache|indexes]" >&2
            exit 1
            ;;
    esac
}

handle_webhook() {
    local action="${1:-}"
    shift || true
    
    case "$action" in
        create)
            create_webhook "$@"
            ;;
        list)
            list_webhooks
            ;;
        delete)
            delete_webhook "$@"
            ;;
        *)
            echo "Usage: webhook [create|list|delete]" >&2
            exit 1
            ;;
    esac
}

main "$@"