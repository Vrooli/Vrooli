#!/bin/bash
# PostGIS Standalone Status Functions

# PostGIS Docker configuration
POSTGIS_CONTAINER="postgis-main"
POSTGIS_STANDALONE_PORT="${POSTGIS_STANDALONE_PORT:-5434}"

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/*}/../.." && builtin pwd)}"
POSTGIS_STATUS_LIB_DIR="${APP_ROOT}/resources/postgis/lib"

# Source dependencies
source "${POSTGIS_STATUS_LIB_DIR}/common.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/status-args.sh"

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
    
    # Check if container exists
    local container_exists=false
    local container_running=false
    local healthy="false"
    local health_message="PostGIS not installed"
    local postgis_version="not_installed"
    
    if docker ps -a --format "{{.Names}}" | grep -q "^${POSTGIS_CONTAINER}$"; then
        container_exists=true
        
        if docker ps --format "{{.Names}}" | grep -q "^${POSTGIS_CONTAINER}$"; then
            container_running=true
            
            # Check if PostGIS is accessible
            if docker exec "${POSTGIS_CONTAINER}" pg_isready -U vrooli -d spatial &>/dev/null; then
                healthy="true"
                health_message="PostGIS is running and accessible"
                
                # Get PostGIS version (skip in fast mode for expensive operations)
                if [[ "$fast_mode" == "false" ]]; then
                    postgis_version=$(docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d spatial -t -c "SELECT PostGIS_Version();" 2>/dev/null | xargs)
                else
                    postgis_version="N/A"
                fi
                
                if [[ -z "$postgis_version" || "$postgis_version" == "N/A" ]]; then
                    postgis_version="unknown"
                fi
            else
                health_message="PostGIS container running but not accessible"
            fi
        else
            health_message="PostGIS container exists but not running"
        fi
    fi
    
    # Get spatial tables count (skip in fast mode)
    local spatial_tables=0
    if [[ "$container_running" == "true" && "$healthy" == "true" ]]; then
        if [[ "$fast_mode" == "false" ]]; then
            spatial_tables=$(docker exec "${POSTGIS_CONTAINER}" psql -U vrooli -d spatial -t -c \
                "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public' AND table_name NOT LIKE 'spatial_%';" 2>/dev/null | xargs)
            [[ -z "$spatial_tables" ]] && spatial_tables=0
        else
            spatial_tables="N/A"
        fi
    fi
    
    # Container status
    local container_status="not_found"
    if [[ "$container_exists" == "true" ]]; then
        container_status=$(docker inspect --format='{{.State.Status}}' "$POSTGIS_CONTAINER" 2>/dev/null || echo "unknown")
    fi
    
    # Check for test results (skip in fast mode)
    local test_status="not_run"
    local test_timestamp=""
    local test_results_file="${POSTGIS_DATA_DIR}/test_results.json"
    
    if [[ "$fast_mode" == "false" && -f "$test_results_file" ]]; then
        # Read last test results
        if [[ -r "$test_results_file" ]]; then
            test_timestamp=$(jq -r '.timestamp // ""' "$test_results_file" 2>/dev/null)
            local test_passed
            test_passed=$(jq -r '.passed // "unknown"' "$test_results_file" 2>/dev/null)
            local test_total
            test_total=$(jq -r '.total // "unknown"' "$test_results_file" 2>/dev/null)

            if [[ -n "$test_timestamp" && "$test_passed" != "unknown" ]]; then
                if [[ "$test_passed" == "$test_total" ]]; then
                    test_status="passed"
                else
                    test_status="failed"
                fi
            fi
        fi
    fi
    
    # Basic resource information
    status_data+=("name" "postgis")
    status_data+=("category" "storage")
    status_data+=("description" "Spatial database extension for PostgreSQL")
    status_data+=("installed" "$container_exists")
    status_data+=("running" "$container_running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$POSTGIS_CONTAINER")
    status_data+=("container_status" "$container_status")
    status_data+=("version" "$postgis_version")
    status_data+=("port" "$POSTGIS_STANDALONE_PORT")
    status_data+=("database" "spatial")
    status_data+=("user" "vrooli")
    status_data+=("spatial_tables" "$spatial_tables")
    status_data+=("default_srid" "4326")
    status_data+=("data_dir" "$POSTGIS_DATA_DIR")
    status_data+=("host" "localhost")
    status_data+=("postgres_url" "postgresql://localhost:$POSTGIS_STANDALONE_PORT/spatial")
    status_data+=("test_status" "$test_status")
    status_data+=("test_timestamp" "$test_timestamp")
    
    # Docker image if available
    if [[ "$container_exists" == "true" ]]; then
        local image
        image=$(docker inspect --format='{{.Config.Image}}' "$POSTGIS_CONTAINER" 2>/dev/null || echo "unknown")
        status_data+=("image" "$image")
    fi
    
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
        echo "  ✅ Installed: Yes (Docker container)"
    else
        echo "  ❌ Installed: No"
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        echo "  ✅ Running: Yes"
    else
        echo "  ⚠️  Running: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        echo "  ✅ Health: Healthy"
    else
        echo "  ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo
    
    # Container information
    echo "Container Info:"
    echo "  📦 Name: ${data[container_name]:-unknown}"
    echo "  📊 Status: ${data[container_status]:-unknown}"
    echo "  🖼️  Image: ${data[image]:-unknown}"
    echo
    
    # Configuration
    echo "Configuration:"
    echo "  🌐 Host: ${data[host]:-unknown}:${data[port]:-unknown}"
    echo "  🗄️  Database: ${data[database]:-unknown}"
    echo "  👤 User: ${data[user]:-unknown}"
    echo "  📊 Version: ${data[version]:-unknown}"
    echo "  📊 Spatial Tables: ${data[spatial_tables]:-0}"
    echo "  🌍 Default SRID: ${data[default_srid]:-unknown}"
    echo "  📁 Data Directory: ${data[data_dir]:-unknown}"
    echo "  🔗 Connection URL: ${data[postgres_url]:-unknown}"
    echo
    
    # Test results
    if [[ -n "${data[test_timestamp]}" ]]; then
        echo "Test Results:"
        if [[ "${data[test_status]}" == "passed" ]]; then
            echo "  ✅ Status: All tests passed"
        elif [[ "${data[test_status]}" == "failed" ]]; then
            echo "  ❌ Status: Some tests failed"
        else
            echo "  ⚠️  Status: ${data[test_status]:-unknown}"
        fi
        echo "  🕐 Last Run: ${data[test_timestamp]}"
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

#######################################
# Display PostGIS credentials and connection information
# Args: [--format text|json|env] [--show-secrets]
#######################################
postgis::credentials() {
    local format="text"
    local show_secrets="false"

    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="$2"
                shift 2
                ;;
            --show-secrets)
                show_secrets="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done

    # Check if PostGIS is running
    if ! docker ps --format "{{.Names}}" | grep -q "^${POSTGIS_CONTAINER}$"; then
        if [[ "$format" == "json" ]]; then
            echo '{"error": "PostGIS container not running"}'
        else
            log::error "PostGIS container not running"
        fi
        return 2
    fi

    # Collect credential data
    local host="localhost"
    local port="${POSTGIS_STANDALONE_PORT:-5434}"
    local database="spatial"
    local user="vrooli"
    local password="vrooli"
    local connection_url

    # Mask password if not showing secrets
    local display_password="********"
    if [[ "$show_secrets" == "true" ]]; then
        display_password="$password"
        connection_url="postgresql://${user}:${password}@${host}:${port}/${database}"
    else
        connection_url="postgresql://${user}:********@${host}:${port}/${database}"
    fi

    # Format output
    case "$format" in
        json)
            cat <<EOF
{
  "host": "$host",
  "port": $port,
  "database": "$database",
  "user": "$user",
  "password": "$display_password",
  "connection_url": "$connection_url",
  "psql_command": "psql -h $host -p $port -U $user -d $database",
  "docker_command": "docker exec -it $POSTGIS_CONTAINER psql -U $user -d $database"
}
EOF
            ;;
        env)
            echo "export POSTGIS_HOST=\"$host\""
            echo "export POSTGIS_PORT=\"$port\""
            echo "export POSTGIS_DATABASE=\"$database\""
            echo "export POSTGIS_USER=\"$user\""
            if [[ "$show_secrets" == "true" ]]; then
                echo "export POSTGIS_PASSWORD=\"$password\""
            else
                echo "export POSTGIS_PASSWORD=\"********\""
            fi
            echo "export POSTGIS_URL=\"$connection_url\""
            ;;
        text|*)
            echo "PostGIS Connection Credentials"
            echo "=============================="
            echo
            echo "Connection Details:"
            echo "  Host: $host"
            echo "  Port: $port"
            echo "  Database: $database"
            echo "  User: $user"
            echo "  Password: $display_password"
            echo
            echo "Connection URL:"
            echo "  $connection_url"
            echo
            echo "Connect via psql:"
            echo "  psql -h $host -p $port -U $user -d $database"
            echo
            echo "Connect via Docker:"
            echo "  docker exec -it $POSTGIS_CONTAINER psql -U $user -d $database"
            echo
            if [[ "$show_secrets" == "false" ]]; then
                echo "💡 Use --show-secrets to display actual password values"
            fi
            ;;
    esac
}

# Export functions
export -f postgis_status
export -f postgis::credentials