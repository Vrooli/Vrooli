#!/bin/bash
# PostGIS Status Functions

# Get script directory
POSTGIS_STATUS_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${POSTGIS_STATUS_LIB_DIR}/common.sh"
# shellcheck disable=SC1091
source "${POSTGIS_STATUS_LIB_DIR}/../../../lib/status-args.sh"

#######################################
# Collect PostGIS status data in format-agnostic structure
# Args: [--fast] - Skip expensive operations for faster response
# Returns: Key-value pairs ready for formatting
#######################################
postgis::status::collect_data() {
    local fast_mode="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --fast)
                fast_mode="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    local status_data=()
    
    # Initialize
    postgis_init_dirs
    
    # Check PostgreSQL connection
    local postgres_available=false
    if postgis_check_postgres; then
        postgres_available=true
    fi
    
    # Check PostGIS installation
    local postgis_installed=false
    if [ "$postgres_available" = "true" ] && postgis_is_installed; then
        postgis_installed=true
    fi
    
    # Check if enabled in default database
    local postgis_enabled=false
    local postgis_version="not_installed"
    if [ "$postgis_installed" = "true" ]; then
        if postgis_is_enabled "$POSTGIS_PG_DATABASE"; then
            postgis_enabled=true
            postgis_version=$(postgis_get_version "$POSTGIS_PG_DATABASE")
        fi
    fi
    
    # Determine health status
    local healthy="false"
    local health_message="PostGIS not configured"
    
    if [ "$postgres_available" = "false" ]; then
        health_message="PostgreSQL not available"
    elif [ "$postgis_installed" = "false" ]; then
        health_message="PostGIS extension not installed in PostgreSQL"
    elif [ "$postgis_enabled" = "false" ]; then
        health_message="PostGIS not enabled in database $POSTGIS_PG_DATABASE"
    else
        healthy="true"
        health_message="PostGIS is installed and enabled"
    fi
    
    # Count spatial tables if enabled (skip in fast mode)
    local spatial_tables=0
    if [ "$postgis_enabled" = "true" ] && [ "$fast_mode" = "false" ]; then
        spatial_tables=$(postgis_exec_sql "$POSTGIS_PG_DATABASE" "SELECT COUNT(*) FROM geometry_columns" || echo "0")
    elif [ "$postgis_enabled" = "true" ]; then
        spatial_tables="N/A"
    fi
    
    # Get SRID count for verbose info (skip in fast mode)
    local srid_count="N/A"
    if [ "$postgis_enabled" = "true" ] && [ "$fast_mode" = "false" ]; then
        srid_count=$(postgis_exec_sql "$POSTGIS_PG_DATABASE" "SELECT COUNT(*) FROM spatial_ref_sys" || echo "0")
    fi
    
    # Basic resource information
    status_data+=("name" "$POSTGIS_RESOURCE_NAME")
    status_data+=("category" "$POSTGIS_RESOURCE_CATEGORY")
    status_data+=("description" "$POSTGIS_RESOURCE_DESCRIPTION")
    status_data+=("installed" "$postgis_installed")
    status_data+=("running" "$postgis_enabled")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("version" "$postgis_version")
    status_data+=("postgres_host" "$POSTGIS_PG_HOST")
    status_data+=("postgres_port" "$POSTGIS_PG_PORT")
    status_data+=("database" "$POSTGIS_PG_DATABASE")
    status_data+=("spatial_tables" "$spatial_tables")
    status_data+=("extensions" "$POSTGIS_EXTENSIONS")
    status_data+=("data_dir" "$POSTGIS_DATA_DIR")
    status_data+=("default_srid" "$POSTGIS_DEFAULT_SRID")
    status_data+=("srid_count" "$srid_count")
    status_data+=("postgres_url" "postgresql://$POSTGIS_PG_HOST:$POSTGIS_PG_PORT/$POSTGIS_PG_DATABASE")
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Display PostGIS status in text format
# Args: data_array (key-value pairs)
#######################################
postgis::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    echo "PostGIS Status Report"
    echo "===================="
    echo
    echo "Description: ${data[description]:-unknown}"
    echo "Category: ${data[category]:-unknown}"
    echo
    
    # Basic status
    echo "Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        echo "  ✅ Installed: Yes"
    else
        echo "  ⚠️  Installed: No (extension not in PostgreSQL)"
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        echo "  ✅ Enabled: Yes (in ${data[database]:-unknown})"
    else
        echo "  ⚠️  Enabled: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        echo "  ✅ Health: Healthy"
    else
        echo "  ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo
    
    # Configuration
    echo "Configuration:"
    echo "  🌐 PostgreSQL Host: ${data[postgres_host]:-unknown}:${data[postgres_port]:-unknown}"
    echo "  🗄️  Database: ${data[database]:-unknown}"
    echo "  📊 Version: ${data[version]:-unknown}"
    echo "  🔧 Extensions: ${data[extensions]:-unknown}"
    echo "  📊 Spatial Tables: ${data[spatial_tables]:-0}"
    echo "  🌍 Default SRID: ${data[default_srid]:-unknown}"
    echo "  📁 Data Directory: ${data[data_dir]:-unknown}"
    echo "  🔗 Connection URL: ${data[postgres_url]:-unknown}"
    echo
    
    # Spatial Reference Systems (if available)
    if [[ "${data[srid_count]:-}" != "N/A" && -n "${data[srid_count]:-}" ]]; then
        echo "Spatial Reference Systems:"
        echo "  🌍 Total SRIDs: ${data[srid_count]:-0}"
        echo
    fi
    
    # Status message
    echo "Status Message:"
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        echo "  ✅ ${data[health_message]:-unknown}"
    else
        echo "  ⚠️  ${data[health_message]:-unknown}"
    fi
}

# Get PostGIS status
postgis_status() {
    status::run_standard "postgis" "postgis::status::collect_data" "postgis::status::display_text" "$@"
}

# Export function for use by other scripts
export -f postgis_status