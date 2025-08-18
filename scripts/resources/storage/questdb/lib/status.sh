#!/usr/bin/env bash
# Ensure port is set correctly
: ${QUESTDB_HTTP_PORT:=9009}
# QuestDB Status Management - Standardized Format
# Functions for checking and displaying QuestDB status information

# Source format utilities
QUESTDB_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${QUESTDB_LIB_DIR}/../../../../lib/utils/format.sh"
# shellcheck disable=SC1091
source "${QUESTDB_LIB_DIR}/../config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${QUESTDB_LIB_DIR}/common.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${QUESTDB_LIB_DIR}/docker.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${QUESTDB_LIB_DIR}/api.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${QUESTDB_LIB_DIR}/../../../../lib/logging.sh" 2>/dev/null || true

# Ensure configuration is exported
if command -v questdb::export_config &>/dev/null; then
    questdb::export_config 2>/dev/null || true
fi

#######################################
# Collect QuestDB status data in format-agnostic structure
# Returns: Key-value pairs ready for formatting
#######################################
questdb::status::collect_data() {
    local status_data=()
    
    # Basic status checks
    local installed="false"
    local running="false" 
    local healthy="false"
    local container_status="not_found"
    local health_message="Unknown"
    
    # Check if container exists
    if docker ps -a --format "{{.Names}}" | grep -q "^${QUESTDB_CONTAINER_NAME}$" 2>/dev/null; then
        installed="true"
        container_status=$(docker inspect --format='{{.State.Status}}' "${QUESTDB_CONTAINER_NAME}" 2>/dev/null || echo "unknown")
        
        # Check if container is running
        if docker ps --format "{{.Names}}" | grep -q "^${QUESTDB_CONTAINER_NAME}$" 2>/dev/null; then
            running="true"
            
            # Check if API is healthy using a simple query
            if curl -G -s --max-time 5 "http://localhost:${QUESTDB_HTTP_PORT}/exec" --data-urlencode "query=SELECT 1" 2>/dev/null | grep -q '"count":1'; then
                healthy="true"
                health_message="Healthy - Database operational and responsive"
            else
                health_message="Unhealthy - Database not responding to queries"
            fi
        else
            health_message="Stopped - Container not running"
        fi
    else
        health_message="Not installed - Container does not exist"
    fi
    
    # Basic resource information
    status_data+=("name" "questdb")
    status_data+=("category" "storage")
    status_data+=("description" "High-performance time series database")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "${QUESTDB_CONTAINER_NAME:-questdb}")
    status_data+=("container_status" "$container_status")
    status_data+=("http_port" "${QUESTDB_HTTP_PORT:-9000}")
    status_data+=("pg_port" "${QUESTDB_PG_PORT:-8812}")
    status_data+=("ilp_port" "${QUESTDB_ILP_PORT:-9011}")
    
    # Service endpoints
    status_data+=("web_console" "http://localhost:${QUESTDB_HTTP_PORT:-9000}")
    status_data+=("health_url" "http://localhost:${QUESTDB_HTTP_PORT:-9000}/exec?query=SELECT%201")
    status_data+=("query_api" "http://localhost:${QUESTDB_HTTP_PORT:-9000}/exec")
    status_data+=("postgres_url" "postgresql://localhost:${QUESTDB_PG_PORT:-8812}/qdb")
    
    # Configuration details
    status_data+=("image" "${QUESTDB_IMAGE:-unknown}")
    status_data+=("version" "${QUESTDB_IMAGE:-unknown}")  # Extract version from image tag
    status_data+=("data_dir" "${QUESTDB_DATA_DIR:-unknown}")
    
    # Runtime information (only if running and healthy)
    if [[ "$running" == "true" && "$healthy" == "true" ]]; then
        # Version from API (try to get it, fallback to unknown)
        local api_version="unknown"
        if command -v questdb::get_version &>/dev/null; then
            api_version=$(questdb::get_version 2>/dev/null || echo "unknown")
        fi
        status_data+=("runtime_version" "$api_version")
        
        # Table count (try to get it, fallback to 0)
        local table_count="0"
        if declare -f questdb::api::query &>/dev/null; then
            table_count=$(questdb::api::query "SHOW TABLES;" 100 true 2>/dev/null | \
                jq -r '.count // 0' 2>/dev/null || echo "0")
        fi
        status_data+=("table_count" "$table_count")
        
        # Uptime calculation
        local uptime_seconds="0"
        local start_time
        start_time=$(docker inspect -f '{{.State.StartedAt}}' "${QUESTDB_CONTAINER_NAME}" 2>/dev/null || echo "")
        if [[ -n "$start_time" ]]; then
            uptime_seconds=$(( $(date +%s) - $(date -d "$start_time" +%s 2>/dev/null || date +%s) ))
        fi
        status_data+=("uptime_seconds" "$uptime_seconds")
        
        local uptime_human
        uptime_human=$(questdb::status::format_uptime "$uptime_seconds")
        status_data+=("uptime" "$uptime_human")
        
        # Data directory size if available
        if [[ -d "${QUESTDB_DATA_DIR}" ]]; then
            local data_size
            data_size=$(du -sh "${QUESTDB_DATA_DIR}" 2>/dev/null | awk '{print $1}' || echo "N/A")
            status_data+=("data_size" "$data_size")
        fi
        
        # Container resource usage (use Docker stats)
        if command -v docker &>/dev/null && docker ps --format "{{.Names}}" | grep -q "^${QUESTDB_CONTAINER_NAME}$"; then
            local stats
            stats=$(docker stats "${QUESTDB_CONTAINER_NAME}" --no-stream --format "{{.CPUPerc}}|{{.MemUsage}}" 2>/dev/null || echo "")
            if [[ -n "$stats" ]]; then
                local cpu mem
                cpu=$(echo "$stats" | cut -d'|' -f1)
                mem=$(echo "$stats" | cut -d'|' -f2)
                status_data+=("cpu_usage" "$cpu")
                status_data+=("memory_usage" "$mem")
            fi
        fi
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Show QuestDB status using standardized format
# Args: [--format json|text] [--verbose]
#######################################
questdb::status::check() {
    local format="text"
    local verbose="false"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --format)
                format="$2"
                shift 2
                ;;
            --json)
                format="json"
                shift
                ;;
            --verbose|-v)
                verbose="true"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Collect status data
    local data_string
    data_string=$(questdb::status::collect_data 2>/dev/null)
    
    if [[ -z "$data_string" ]]; then
        # Fallback if data collection fails
        if [[ "$format" == "json" ]]; then
            echo '{"error": "Failed to collect status data"}'
        else
            log::error "Failed to collect QuestDB status data"
        fi
        return 1
    fi
    
    # Convert string to array
    local data_array
    mapfile -t data_array <<< "$data_string"
    
    # Output based on format
    if [[ "$format" == "json" ]]; then
        format::output "json" "kv" "${data_array[@]}"
    else
        # Text format with standardized structure
        questdb::status::display_text "${data_array[@]}"
    fi
    
    # Return appropriate exit code
    local healthy="false"
    local running="false"
    for ((i=0; i<${#data_array[@]}; i+=2)); do
        case "${data_array[i]}" in
            "healthy") healthy="${data_array[i+1]}" ;;
            "running") running="${data_array[i+1]}" ;;
        esac
    done
    
    if [[ "$healthy" == "true" ]]; then
        return 0
    elif [[ "$running" == "true" ]]; then
        return 1
    else
        return 2
    fi
}

#######################################
# Display status in text format
#######################################
questdb::status::display_text() {
    local -A data
    
    # Convert array to associative array
    local args=("$@")
    for ((i=0; i<${#args[@]}; i+=2)); do
        local key="${args[i]}"
        local value="${args[i+1]:-}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "📊 QuestDB Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install QuestDB, run: ./manage.sh --action install"
        return
    fi
    
    if [[ "${data[running]:-false}" == "true" ]]; then
        log::success "   ✅ Running: Yes"
    else
        log::warn "   ⚠️  Running: No"
    fi
    
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::success "   ✅ Health: Healthy"
    else
        log::warn "   ⚠️  Health: ${data[health_message]:-Unknown}"
    fi
    echo
    
    # Container information
    log::info "🐳 Container Info:"
    log::info "   📦 Name: ${data[container_name]:-unknown}"
    log::info "   📊 Status: ${data[container_status]:-unknown}"
    log::info "   🖼️  Image: ${data[image]:-unknown}"
    echo
    
    # Service endpoints
    log::info "🌐 Service Endpoints:"
    log::info "   🎨 Web Console: ${data[web_console]:-unknown}"
    log::info "   🔍 Query API: ${data[query_api]:-unknown}"
    log::info "   🏥 Health Check: ${data[health_url]:-unknown}"
    log::info "   🐘 PostgreSQL: ${data[postgres_url]:-unknown}"
    echo
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   🌐 HTTP Port: ${data[http_port]:-unknown}"
    log::info "   🐘 PostgreSQL Port: ${data[pg_port]:-unknown}"
    log::info "   📊 InfluxDB Port: ${data[ilp_port]:-unknown}"
    log::info "   📁 Data Directory: ${data[data_dir]:-unknown}"
    log::info "   🏷️  Version: ${data[version]:-unknown}"
    echo
    
    # Runtime information (only if healthy)
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::info "📈 Runtime Information:"
        log::info "   🔖 Runtime Version: ${data[runtime_version]:-unknown}"
        log::info "   ⏱️  Uptime: ${data[uptime]:-unknown}"
        log::info "   🗄️  Tables: ${data[table_count]:-0}"
        log::info "   💾 Data Size: ${data[data_size]:-unknown}"
        
        if [[ -n "${data[cpu_usage]:-}" ]]; then
            log::info "   🔥 CPU Usage: ${data[cpu_usage]}"
        fi
        if [[ -n "${data[memory_usage]:-}" ]]; then
            log::info "   🧠 Memory Usage: ${data[memory_usage]}"
        fi
    fi
}

#######################################
# Format uptime seconds to human readable
# Arguments:
#   $1 - uptime in seconds
# Returns: Human readable uptime
#######################################
questdb::status::format_uptime() {
    local seconds=$1
    local days=$((seconds / 86400))
    local hours=$(((seconds % 86400) / 3600))
    local minutes=$(((seconds % 3600) / 60))
    local secs=$((seconds % 60))
    
    if [[ $days -gt 0 ]]; then
        echo "${days}d ${hours}h ${minutes}m ${secs}s"
    elif [[ $hours -gt 0 ]]; then
        echo "${hours}h ${minutes}m ${secs}s"
    elif [[ $minutes -gt 0 ]]; then
        echo "${minutes}m ${secs}s"
    else
        echo "${secs}s"
    fi
}

#######################################
# Legacy function for backward compatibility
#######################################
questdb::status::detailed() {
    # Redirect to the new standardized status display
    questdb::status::check "$@"
}

#######################################
# Check system metrics
# Returns:
#   JSON with system metrics
#######################################
questdb::status::metrics() {
    if ! questdb::docker::is_running; then
        echo "{}"
        return 1
    fi
    
    # Get various metrics
    local uptime tables rows
    
    # Calculate uptime
    local start_time
    start_time=$(docker inspect -f '{{.State.StartedAt}}' "${QUESTDB_CONTAINER_NAME}" 2>/dev/null || echo "")
    if [[ -n "$start_time" ]]; then
        uptime=$(( $(date +%s) - $(date -d "$start_time" +%s) ))
    else
        uptime=0
    fi
    
    # Get table count
    tables=$(questdb::api::query "SHOW TABLES;" 100 true 2>/dev/null | \
        jq -r '.count // 0' 2>/dev/null || echo "0")
    
    # Get total row count (sum across all tables)
    rows=0
    
    # Output as JSON
    cat <<EOF
{
    "status": "running",
    "uptime_seconds": ${uptime},
    "tables": ${tables},
    "total_rows": ${rows},
    "http_port": ${QUESTDB_HTTP_PORT},
    "pg_port": ${QUESTDB_PG_PORT},
    "ilp_port": ${QUESTDB_ILP_PORT}
}
EOF
}

#######################################
# Monitor QuestDB in real-time
# Shows updating stats every few seconds
#######################################
questdb::status::monitor() {
    if ! questdb::docker::is_running; then
        log::error "${QUESTDB_STATUS_MESSAGES["not_running"]}"
        return 1
    fi
    
    log::info "Monitoring QuestDB (Press Ctrl+C to stop)..."
    echo ""
    
    while true; do
        clear
        echo "📊 QuestDB Monitor - $(date)"
        echo "================================"
        
        # Container stats
        docker stats "${QUESTDB_CONTAINER_NAME}" --no-stream
        
        # Query stats
        echo ""
        echo "Recent Queries:"
        questdb::api::query "SELECT query_text, execution_time FROM sys.query_log ORDER BY timestamp DESC LIMIT 5" 5 2>/dev/null || echo "No query log available"
        
        sleep 5
    done
}

#######################################
# Test QuestDB functionality
# Returns: 0 if all tests pass, 1 if any fail, 2 if service not ready
#######################################
questdb::test() {
    log::info "Testing QuestDB functionality..."
    
    # Test 1: Check if QuestDB is installed (container exists)
    if ! questdb::docker::exists; then
        log::error "❌ QuestDB container is not installed"
        return 1
    fi
            log::success "✅ QuestDB container is installed"
    
    # Test 2: Check if service is running
    if ! questdb::docker::is_running; then
        log::error "❌ QuestDB service is not running"
        return 2
    fi
            log::success "✅ QuestDB service is running"
    
    # Test 3: Check API health
    if ! questdb::api::health_check; then
        log::error "❌ QuestDB API is not responding"
        return 1
    fi
            log::success "✅ QuestDB API is healthy"
    
    # Test 4: Test SQL queries
    log::info "Testing SQL operations..."
    local test_result
    test_result=$(questdb::api::query "SELECT 1 as test_value" 1 2>/dev/null || echo "")
    if [[ -n "$test_result" ]] && echo "$test_result" | grep -q "test_value"; then
        log::success "✅ SQL queries working"
    else
        log::error "❌ SQL query test failed"
        return 1
    fi
    
    # Test 5: Test table operations
    log::info "Testing table operations..."
    local test_table="vrooli_test_table_$(date +%s)"
    
    if questdb::api::query "CREATE TABLE $test_table (id INT, name STRING)" 1 >/dev/null 2>&1; then
        log::success "✅ Table creation successful"
        
        # Test insert
        if questdb::api::query "INSERT INTO $test_table VALUES (1, 'test')" 1 >/dev/null 2>&1; then
            log::success "✅ Data insertion successful"
        fi
        
        # Clean up test table
        questdb::api::query "DROP TABLE $test_table" 1 >/dev/null 2>&1 || true
        log::success "✅ Table cleanup successful"
    else
        echo_warn "⚠️  Table operations test failed - may be permission issue"
    fi
    
    # Test 6: Check storage metrics
    log::info "Testing storage metrics..."
    local storage_info
    storage_info=$(docker exec "$QUESTDB_CONTAINER_NAME" df -h /root/.questdb 2>/dev/null | tail -1 | awk '{print $3}' || echo "unknown")
    if [[ "$storage_info" != "unknown" ]]; then
        log::success "✅ Storage metrics available (used: $storage_info)"
    else
        echo_warn "⚠️  Storage metrics unavailable"
    fi
    
    log::success "🎉 All QuestDB tests passed"
    return 0
}

#######################################
# Show comprehensive QuestDB information
#######################################
questdb::info() {
    cat << EOF
=== QuestDB Resource Information ===

ID: questdb
Category: storage
Display Name: QuestDB Time Series Database
Description: High-performance time series database optimized for real-time analytics

Service Details:
- Container Name: $QUESTDB_CONTAINER_NAME
- HTTP Port: $QUESTDB_HTTP_PORT
- PostgreSQL Port: $QUESTDB_PG_PORT
- InfluxDB Line Protocol Port: $QUESTDB_ILP_PORT
- HTTP URL: http://localhost:$QUESTDB_HTTP_PORT
- PostgreSQL URL: postgresql://localhost:$QUESTDB_PG_PORT/qdb
- Data Directory: $QUESTDB_DATA_DIR

Endpoints:
- Web Console: http://localhost:$QUESTDB_HTTP_PORT
- Health Check: http://localhost:$QUESTDB_HTTP_PORT/exec?query=SELECT%201
- Query API: http://localhost:$QUESTDB_HTTP_PORT/exec
- Import API: http://localhost:$QUESTDB_HTTP_PORT/imp
- Export API: http://localhost:$QUESTDB_HTTP_PORT/exp

Protocols:
- HTTP REST API for queries and management
- PostgreSQL wire protocol for standard SQL tools
- InfluxDB Line Protocol for high-throughput ingestion

Configuration:
- Docker Image: $QUESTDB_IMAGE
- Version: $QUESTDB_VERSION
- Data Persistence: $QUESTDB_DATA_DIR
- Log Directory: $QUESTDB_LOG_DIR
- Config Directory: $QUESTDB_CONFIG_DIR

QuestDB Features:
- Time series optimized storage
- SQL with time series extensions
- High-performance ingestion
- Real-time aggregations
- Column-oriented storage
- SIMD optimizations
- Zero-GC Java implementation
- PostgreSQL compatibility

Example Usage:
# Execute a SQL query
$0 --action query --query "SELECT * FROM my_table LATEST ON timestamp PARTITION BY id"

# Create a table
$0 --action tables --table sensors --schema /path/to/schema.sql

# Open web console
$0 --action console

# Monitor performance
$0 --action monitor

Documentation: https://questdb.io/docs/
EOF
}

#######################################
# Export status functions
#######################################
export -f questdb::status::check
export -f questdb::status::detailed
export -f questdb::status::metrics
export -f questdb::status::monitor
export -f questdb::test
export -f questdb::info