#!/usr/bin/env bash
# Redis Status Management
# Functions for checking and displaying Redis status information

#######################################
# Show Redis status
# Returns: 0 if healthy, 1 if unhealthy, 2 if not running
#######################################
redis::status::show() {
    log::info "${MSG_CHECKING_STATUS}"
    
    if ! redis::common::container_exists; then
        log::error "${MSG_STATUS_NOT_INSTALLED}"
        return 2
    fi
    
    if ! redis::common::is_running; then
        log::warn "${MSG_STATUS_STOPPED}"
        return 2
    fi
    
    if redis::common::is_healthy; then
        log::success "${MSG_STATUS_RUNNING}"
        redis::status::show_detailed_info
        return 0
    else
        log::warn "${MSG_STATUS_UNHEALTHY}"
        redis::status::show_basic_info
        return 1
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