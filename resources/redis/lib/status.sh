#!/usr/bin/env bash
# Redis Status Management - Standardized Format
# Functions for checking and displaying Redis status information

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
REDIS_STATUS_DIR="${APP_ROOT}/resources/redis/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/format.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/status-args.sh"
# shellcheck disable=SC1091
source "${REDIS_STATUS_DIR}/../config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${REDIS_STATUS_DIR}/../config/messages.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${REDIS_STATUS_DIR}/common.sh" 2>/dev/null || true

# Ensure configuration is exported
if command -v redis::export_config &>/dev/null; then
    redis::export_config 2>/dev/null || true
fi
if command -v redis::messages::init &>/dev/null; then
    redis::messages::init 2>/dev/null || true
fi

#######################################
# Collect Redis status data in format-agnostic structure
# Returns: Key-value pairs ready for formatting
#######################################
redis::status::collect_data() {
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
    
    # Basic status checks
    local installed="false"
    local running="false"
    local healthy="false"
    local container_status="not_found"
    local health_message="Unknown"
    
    if redis::common::container_exists; then
        installed="true"
        container_status=$(docker inspect --format='{{.State.Status}}' "${REDIS_CONTAINER_NAME}" 2>/dev/null || echo "unknown")
        
        if redis::common::is_running; then
            running="true"
            
            if redis::common::is_healthy; then
                healthy="true"
                health_message="Healthy - All systems operational"
            else
                health_message="Unhealthy - Service not responding"
            fi
        else
            health_message="Stopped - Container not running"
        fi
    else
        health_message="Not installed - Container does not exist"
    fi
    
    # Basic resource information
    status_data+=("name" "redis")
    status_data+=("category" "storage")
    status_data+=("description" "Redis cache and data structure server")
    status_data+=("installed" "$installed")
    status_data+=("running" "$running")
    status_data+=("healthy" "$healthy")
    status_data+=("health_message" "$health_message")
    status_data+=("container_name" "$REDIS_CONTAINER_NAME")
    status_data+=("container_status" "$container_status")
    status_data+=("port" "$REDIS_PORT")
    
    # Service endpoints
    status_data+=("redis_url" "redis://localhost:${REDIS_PORT}")
    status_data+=("cli_command" "redis-cli -p ${REDIS_PORT}")
    
    # Configuration details
    status_data+=("image" "$REDIS_IMAGE")
    status_data+=("max_memory" "$REDIS_MAX_MEMORY")
    status_data+=("persistence" "$REDIS_PERSISTENCE")
    status_data+=("databases" "$REDIS_DATABASES")
    
    # Runtime information (only if running and healthy)
    if [[ "$running" == "true" && "$healthy" == "true" ]]; then
        # Skip expensive redis info gathering in fast mode
        if [[ "$fast_mode" == "true" ]]; then
            status_data+=("memory_used" "N/A")
            status_data+=("connected_clients" "N/A")
            status_data+=("total_commands" "N/A")
            status_data+=("version" "N/A")
            status_data+=("uptime" "N/A")
            status_data+=("total_keys" "N/A")
            status_data+=("active_databases" "N/A")
        else
            local redis_info
            redis_info=$(timeout 3s redis::common::get_info 2>/dev/null)
        
        if [[ -n "$redis_info" ]]; then
            # Memory usage
            local memory_used
            memory_used=$(echo "$redis_info" | grep "used_memory_human:" | cut -d: -f2 | tr -d '\r' | xargs)
            status_data+=("memory_used" "${memory_used:-N/A}")
            
            # Connected clients
            local clients
            clients=$(echo "$redis_info" | grep "connected_clients:" | cut -d: -f2 | tr -d '\r' | xargs)
            status_data+=("connected_clients" "${clients:-0}")
            
            # Commands processed
            local commands
            commands=$(echo "$redis_info" | grep "total_commands_processed:" | cut -d: -f2 | tr -d '\r' | xargs)
            status_data+=("total_commands" "${commands:-0}")
            
            # Redis version
            local version
            version=$(echo "$redis_info" | grep "redis_version:" | cut -d: -f2 | tr -d '\r' | xargs)
            status_data+=("version" "${version:-unknown}")
            
            # Uptime
            local uptime_seconds
            uptime_seconds=$(echo "$redis_info" | grep "uptime_in_seconds:" | cut -d: -f2 | tr -d '\r' | xargs)
            if [[ -n "$uptime_seconds" ]]; then
                local uptime_human
                uptime_human=$(redis::status::format_uptime "$uptime_seconds")
                status_data+=("uptime" "$uptime_human")
            else
                status_data+=("uptime" "unknown")
            fi
            
            # Database key count
            local total_keys=0
            local active_dbs=0
            for i in $(seq 0 $((REDIS_DATABASES - 1))); do
                local db_info
                db_info=$(echo "$redis_info" | grep "^db${i}:")
                if [[ -n "$db_info" ]]; then
                    local keys
                    keys=$(echo "$db_info" | sed 's/.*keys=\([0-9]*\).*/\1/')
                    if [[ -n "$keys" && "$keys" -gt 0 ]]; then
                        active_dbs=$((active_dbs + 1))
                        total_keys=$((total_keys + keys))
                    fi
                fi
            done
            status_data+=("total_keys" "$total_keys")
            status_data+=("active_databases" "$active_dbs")
        fi
        fi
    fi
    
    # Return the collected data
    printf '%s\n' "${status_data[@]}"
}

#######################################
# Show Redis status using standardized format
# Args: [--format json|text] [--verbose]
#######################################
redis::status::show() {
    status::run_standard "redis" "redis::status::collect_data" "redis::status::display_text" "$@"
}

#######################################
# Display status in text format
#######################################
redis::status::display_text() {
    local -A data
    
    # Convert array to associative array
    for ((i=1; i<=$#; i+=2)); do
        local key="${!i}"
        local value_idx=$((i+1))
        local value="${!value_idx}"
        data["$key"]="$value"
    done
    
    # Header
    log::header "📊 Redis Status"
    echo
    
    # Basic status
    log::info "📊 Basic Status:"
    if [[ "${data[installed]:-false}" == "true" ]]; then
        log::success "   ✅ Installed: Yes"
    else
        log::error "   ❌ Installed: No"
        echo
        log::info "💡 Installation Required:"
        log::info "   To install Redis, run: ./manage.sh --action install"
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
    log::info "   🔗 Redis URL: ${data[redis_url]:-unknown}"
    log::info "   💻 CLI Command: ${data[cli_command]:-unknown}"
    echo
    
    # Configuration
    log::info "⚙️  Configuration:"
    log::info "   📶 Port: ${data[port]:-unknown}"
    log::info "   💾 Max Memory: ${data[max_memory]:-unknown}"
    log::info "   💿 Persistence: ${data[persistence]:-unknown}"
    log::info "   🗄️  Databases: ${data[databases]:-unknown}"
    echo
    
    # Runtime information (only if healthy)
    if [[ "${data[healthy]:-false}" == "true" ]]; then
        log::info "📈 Runtime Information:"
        log::info "   🔖 Version: ${data[version]:-unknown}"
        log::info "   ⏱️  Uptime: ${data[uptime]:-unknown}"
        log::info "   💾 Memory Used: ${data[memory_used]:-unknown}"
        log::info "   👥 Connected Clients: ${data[connected_clients]:-0}"
        log::info "   📊 Total Commands: ${data[total_commands]:-0}"
        log::info "   🗄️  Active Databases: ${data[active_databases]:-0}"
        log::info "   🔑 Total Keys: ${data[total_keys]:-0}"
    fi
}

#######################################
# Show detailed Redis information
#######################################
redis::status::show_detailed_info() {
    log::info "${MSG_STATS_HEADER}"
    
    # Get Redis info
    local redis_info
    redis_info=$(redis::common::get_info)
    
    if [[ -n "$redis_info" ]]; then
        # Memory usage
        local memory_used
        memory_used=$(echo "$redis_info" | grep "used_memory_human:" | cut -d: -f2 | tr -d '\r' | xargs)
        if [[ -n "$memory_used" ]]; then
            log::info "${MSG_MEMORY_USAGE}${memory_used}"
        fi
        
        # Connected clients
        local clients
        clients=$(echo "$redis_info" | grep "connected_clients:" | cut -d: -f2 | tr -d '\r' | xargs)
        if [[ -n "$clients" ]]; then
            log::info "${MSG_CONNECTED_CLIENTS}${clients}"
        fi
        
        # Total commands processed
        local commands
        commands=$(echo "$redis_info" | grep "total_commands_processed:" | cut -d: -f2 | tr -d '\r' | xargs)
        if [[ -n "$commands" ]]; then
            log::info "${MSG_TOTAL_COMMANDS}${commands}"
        fi
        
        # Operations per second
        local ops_per_sec
        ops_per_sec=$(echo "$redis_info" | grep "instantaneous_ops_per_sec:" | cut -d: -f2 | tr -d '\r' | xargs)
        if [[ -n "$ops_per_sec" ]]; then
            log::info "${MSG_OPS_PER_SECOND}${ops_per_sec}"
        fi
        
        # Redis version
        local redis_version
        redis_version=$(echo "$redis_info" | grep "redis_version:" | cut -d: -f2 | tr -d '\r' | xargs)
        if [[ -n "$redis_version" ]]; then
            log::info "🔖 Redis Version: ${redis_version}"
        fi
        
        # Uptime
        local uptime_seconds
        uptime_seconds=$(echo "$redis_info" | grep "uptime_in_seconds:" | cut -d: -f2 | tr -d '\r' | xargs)
        if [[ -n "$uptime_seconds" ]]; then
            local uptime_human
            uptime_human=$(redis::status::format_uptime "$uptime_seconds")
            log::info "⏱️  Uptime: ${uptime_human}"
        fi
        
        # Database info
        redis::status::show_database_info "$redis_info"
    fi
    
    # Show connection information
    redis::docker::show_connection_info
    
    # Show configuration summary
    redis::status::show_config_summary
    
    # Show repository information
    resources::show_repository_info
}

#######################################
# Show basic Redis information (when unhealthy)
#######################################
redis::status::show_basic_info() {
    log::info "🐳 Container: ${REDIS_CONTAINER_NAME}"
    log::info "🌐 Port: ${REDIS_PORT}"
    log::info "📁 Data Directory: ${REDIS_DATA_DIR}"
    
    # Show Docker container status
    local container_status
    container_status=$(docker inspect --format='{{.State.Status}}' "${REDIS_CONTAINER_NAME}" 2>/dev/null)
    if [[ -n "$container_status" ]]; then
        log::info "📊 Container Status: ${container_status}"
    fi
    
    # Show recent logs
    log::info "📄 Recent logs:"
    docker logs --tail 5 "${REDIS_CONTAINER_NAME}" 2>/dev/null | while read -r line; do
        log::info "   $line"
    done
}

#######################################
# Show database information
# Arguments:
#   $1 - Redis info output
#######################################
redis::status::show_database_info() {
    local redis_info="$1"
    
    # Count databases with keys
    local db_count=0
    local total_keys=0
    
    for i in $(seq 0 $((REDIS_DATABASES - 1))); do
        local db_info
        db_info=$(echo "$redis_info" | grep "^db${i}:")
        if [[ -n "$db_info" ]]; then
            local keys
            keys=$(echo "$db_info" | sed 's/.*keys=\([0-9]*\).*/\1/')
            if [[ -n "$keys" && "$keys" -gt 0 ]]; then
                db_count=$((db_count + 1))
                total_keys=$((total_keys + keys))
                log::info "🗄️  Database ${i}: ${keys} keys"
            fi
        fi
    done
    
    if [[ $db_count -eq 0 ]]; then
        log::info "🗄️  No databases with data"
    else
        log::info "📊 Total: ${db_count} databases, ${total_keys} keys"
    fi
}

#######################################
# Show configuration summary
#######################################
redis::status::show_config_summary() {
    log::info "⚙️  Configuration:"
    log::info "   Max Memory: ${REDIS_MAX_MEMORY}"
    log::info "   Memory Policy: ${REDIS_MAX_MEMORY_POLICY}"
    log::info "   Persistence: ${REDIS_PERSISTENCE}"
    log::info "   Databases: ${REDIS_DATABASES}"
    
    if [[ -n "$REDIS_PASSWORD" ]]; then
        log::info "   🔒 Password: Enabled"
    else
        log::info "   🔓 Password: Disabled"
    fi
}

#######################################
# Format uptime seconds to human readable
# Arguments:
#   $1 - uptime in seconds
# Returns: Human readable uptime
#######################################
redis::status::format_uptime() {
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
# Monitor Redis in real-time
# Arguments:
#   $1 - interval in seconds (optional)
#######################################
redis::status::monitor() {
    local interval="${1:-5}"
    
    if ! redis::common::is_running; then
        log::error "Redis is not running"
        return 1
    fi
    
    log::info "🔍 Monitoring Redis (interval: ${interval}s, press Ctrl+C to stop)"
    echo
    
    while true; do
        # Clear screen and show timestamp
        clear
        echo "Redis Monitor - $(date)"
        echo "================================"
        
        # Get current stats
        local redis_info
        redis_info=$(redis::common::get_info)
        
        if [[ -n "$redis_info" ]]; then
            # Extract key metrics
            local memory_used
            memory_used=$(echo "$redis_info" | grep "used_memory_human:" | cut -d: -f2 | tr -d '\r' | xargs)
            
            local memory_peak
            memory_peak=$(echo "$redis_info" | grep "used_memory_peak_human:" | cut -d: -f2 | tr -d '\r' | xargs)
            
            local clients
            clients=$(echo "$redis_info" | grep "connected_clients:" | cut -d: -f2 | tr -d '\r' | xargs)
            
            local commands
            commands=$(echo "$redis_info" | grep "total_commands_processed:" | cut -d: -f2 | tr -d '\r' | xargs)
            
            local ops_per_sec
            ops_per_sec=$(echo "$redis_info" | grep "instantaneous_ops_per_sec:" | cut -d: -f2 | tr -d '\r' | xargs)
            
            local keyspace_hits
            keyspace_hits=$(echo "$redis_info" | grep "keyspace_hits:" | cut -d: -f2 | tr -d '\r' | xargs)
            
            local keyspace_misses
            keyspace_misses=$(echo "$redis_info" | grep "keyspace_misses:" | cut -d: -f2 | tr -d '\r' | xargs)
            
            # Calculate hit rate
            local hit_rate="N/A"
            if [[ -n "$keyspace_hits" && -n "$keyspace_misses" && "$keyspace_hits" -gt 0 ]]; then
                local total_requests=$((keyspace_hits + keyspace_misses))
                if [[ $total_requests -gt 0 ]]; then
                    hit_rate=$(awk "BEGIN {printf \"%.2f%%\", ($keyspace_hits * 100.0) / $total_requests}")
                fi
            fi
            
            # Display metrics
            echo "Memory Usage:     ${memory_used:-N/A} (Peak: ${memory_peak:-N/A})"
            echo "Connected Clients: ${clients:-N/A}"
            echo "Commands Processed: ${commands:-N/A}"
            echo "Operations/sec:   ${ops_per_sec:-N/A}"
            echo "Cache Hit Rate:   ${hit_rate}"
            echo
            
            # Show database info
            local db_count=0
            local total_keys=0
            
            for i in $(seq 0 $((REDIS_DATABASES - 1))); do
                local db_info
                db_info=$(echo "$redis_info" | grep "^db${i}:")
                if [[ -n "$db_info" ]]; then
                    local keys
                    keys=$(echo "$db_info" | sed 's/.*keys=\([0-9]*\).*/\1/')
                    if [[ -n "$keys" && "$keys" -gt 0 ]]; then
                        db_count=$((db_count + 1))
                        total_keys=$((total_keys + keys))
                        echo "Database ${i}: ${keys} keys"
                    fi
                fi
            done
            
            if [[ $db_count -eq 0 ]]; then
                echo "No databases with data"
            else
                echo "Total: ${db_count} databases, ${total_keys} keys"
            fi
        else
            echo "❌ Unable to retrieve Redis information"
        fi
        
        echo
        echo "Next update in ${interval}s..."
        sleep "$interval"
    done
}

#######################################
# List all client instances
#######################################
redis::status::list_clients() {
    log::info "🤖 Client Redis Instances:"
    
    local found_clients=false
    
    # Find all client containers
    while IFS= read -r container; do
        if [[ -n "$container" ]]; then
            found_clients=true
            local client_id
            client_id=$(docker inspect --format='{{index .Config.Labels "vrooli.client"}}' "$container" 2>/dev/null)
            
            local port
            port=$(docker port "$container" "${REDIS_INTERNAL_PORT}/tcp" 2>/dev/null | cut -d: -f2)
            
            local status
            status=$(docker inspect --format='{{.State.Status}}' "$container" 2>/dev/null)
            
            local health
            health=$(docker inspect --format='{{.State.Health.Status}}' "$container" 2>/dev/null)
            if [[ "$health" == "<no value>" ]]; then
                health="N/A"
            fi
            
            echo "   Client: ${client_id:-unknown}"
            echo "     Container: ${container}"
            echo "     Port: ${port:-N/A}"
            echo "     Status: ${status:-unknown}"
            echo "     Health: ${health}"
            echo "     URL: redis://localhost:${port:-N/A}"
            echo
        fi
    done < <(docker ps -a --filter "label=vrooli.resource=redis" --format "{{.Names}}" | grep "^${REDIS_CLIENT_PREFIX}-")
    
    if [[ "$found_clients" == false ]]; then
        log::info "   No client instances found"
    fi
}