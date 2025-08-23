#!/usr/bin/env bash
# Redis Common Functions
# Shared utility functions for Redis resource management

#######################################
# Check if Redis container exists
# Returns: 0 if exists, 1 if not
#######################################
redis::common::container_exists() {
    docker ps -a --format "{{.Names}}" | grep -q "^${REDIS_CONTAINER_NAME}$"
}

#######################################
# Check if Redis is running
# Returns: 0 if running, 1 if not
#######################################
redis::common::is_running() {
    docker ps --format "{{.Names}}" | grep -q "^${REDIS_CONTAINER_NAME}$"
}

#######################################
# Check if Redis is healthy
# Returns: 0 if healthy, 1 if not
#######################################
redis::common::is_healthy() {
    if ! redis::common::is_running; then
        return 1
    fi
    
    # Check Docker health status
    local health_status
    health_status=$(docker inspect --format='{{.State.Health.Status}}' "${REDIS_CONTAINER_NAME}" 2>/dev/null)
    
    if [[ "$health_status" == "healthy" ]]; then
        return 0
    fi
    
    # Fallback to Redis ping test
    redis::common::ping
}

#######################################
# Ping Redis server
# Returns: 0 if responsive, 1 if not
#######################################
redis::common::ping() {
    local cmd=(docker exec "${REDIS_CONTAINER_NAME}" redis-cli ping)
    
    if [[ -n "$REDIS_PASSWORD" ]]; then
        cmd=(docker exec "${REDIS_CONTAINER_NAME}" redis-cli -a "$REDIS_PASSWORD" ping)
    fi
    
    "${cmd[@]}" 2>/dev/null | grep -q "PONG"
}

#######################################
# Wait for Redis to be ready
# Arguments:
#   $1 - timeout in seconds (optional, default: REDIS_READY_TIMEOUT)
# Returns: 0 if ready, 1 if timeout
#######################################
redis::common::wait_for_ready() {
    local timeout="${1:-$REDIS_READY_TIMEOUT}"
    local elapsed=0
    
    log::debug "Waiting for Redis to be ready (timeout: ${timeout}s)"
    
    while [ $elapsed -lt "$timeout" ]; do
        if redis::common::ping; then
            log::debug "Redis is ready after ${elapsed}s"
            return 0
        fi
        
        sleep 1
        elapsed=$((elapsed + 1))
    done
    
    log::error "Redis failed to become ready within ${timeout}s"
    return 1
}

#######################################
# Create necessary directories
# Returns: 0 on success, 1 on failure
#######################################
redis::common::create_volumes() {
    # Create Docker volumes if they don't exist
    local volumes=(
        "$REDIS_VOLUME_NAME"
        "$REDIS_LOG_VOLUME_NAME"
    )
    
    for volume in "${volumes[@]}"; do
        if ! docker volume inspect "$volume" >/dev/null 2>&1; then
            if ! docker volume create "$volume" >/dev/null 2>&1; then
                log::error "Failed to create volume: $volume"
                return 1
            fi
            log::debug "Created volume: $volume"
        else
            log::debug "Volume already exists: $volume"
        fi
    done
    
    # Create temporary config directory
    if ! mkdir -p "$REDIS_TEMP_CONFIG_DIR"; then
        log::error "Failed to create temporary config directory: $REDIS_TEMP_CONFIG_DIR"
        return 1
    fi
    log::debug "Created temporary config directory: $REDIS_TEMP_CONFIG_DIR"
    
    return 0
}

#######################################
# Generate Redis configuration file
# Returns: 0 on success, 1 on failure
#######################################
redis::common::generate_config() {
    local config_file="$REDIS_CONFIG_FILE"
    
    log::debug "Generating Redis configuration: $config_file"
    
    # Generate configuration directly
    cat > "$config_file" << EOF
# Redis Configuration for Vrooli Resources
# Generated automatically - do not edit directly

# Network Configuration
bind ${REDIS_BIND}
port ${REDIS_INTERNAL_PORT}
protected-mode ${REDIS_PROTECTED_MODE}
tcp-backlog ${REDIS_TCP_BACKLOG}
timeout ${REDIS_TIMEOUT}
tcp-keepalive ${REDIS_TCP_KEEPALIVE}

# General Configuration
daemonize no
supervised no
pidfile /var/run/redis_6379.pid
loglevel ${REDIS_LOGLEVEL}
# logfile disabled - Redis will log to stdout for Docker
databases ${REDIS_DATABASES}

# Memory Management
maxmemory ${REDIS_MAX_MEMORY}
maxmemory-policy ${REDIS_MAX_MEMORY_POLICY}

EOF

    # Add persistence configuration based on mode
    if [[ "$REDIS_PERSISTENCE" == "rdb" || "$REDIS_PERSISTENCE" == "both" ]]; then
        cat >> "$config_file" << EOF
# RDB Persistence
save ${REDIS_SAVE_INTERVALS}
stop-writes-on-bgsave-error yes
rdbcompression yes
rdbchecksum yes
dbfilename dump.rdb
dir /data

EOF
    else
        cat >> "$config_file" << EOF
# RDB Persistence Disabled
save ""

EOF
    fi

    # Add AOF configuration
    if [[ "$REDIS_PERSISTENCE" == "aof" || "$REDIS_PERSISTENCE" == "both" ]]; then
        cat >> "$config_file" << EOF
# AOF Persistence
appendonly yes
appendfilename "appendonly.aof"
appendfsync everysec
no-appendfsync-on-rewrite no
auto-aof-rewrite-percentage 100
auto-aof-rewrite-min-size 64mb
aof-load-truncated yes
aof-use-rdb-preamble yes

EOF
    else
        cat >> "$config_file" << EOF
# AOF Persistence Disabled
appendonly no

EOF
    fi

    # Add password if configured
    if [[ -n "$REDIS_PASSWORD" ]]; then
        cat >> "$config_file" << EOF
# Security
requirepass ${REDIS_PASSWORD}

EOF
    fi

    # Add remaining configuration
    cat >> "$config_file" << EOF
# Performance Configuration
slowlog-log-slower-than 10000
slowlog-max-len 128
latency-monitor-threshold 100

# Advanced Configuration
hash-max-ziplist-entries 512
hash-max-ziplist-value 64
list-max-ziplist-size -2
list-compress-depth 0
set-max-intset-entries 512
zset-max-ziplist-entries 128
zset-max-ziplist-value 64
hll-sparse-max-bytes 3000
stream-node-max-bytes 4096
stream-node-max-entries 100
activerehashing yes
client-output-buffer-limit normal 0 0 0
client-output-buffer-limit replica 256mb 64mb 60
client-output-buffer-limit pubsub 32mb 8mb 60

# Lua Scripting
lua-time-limit 5000

# Memory Usage Optimization
rdb-save-incremental-fsync yes
dynamic-hz yes
jemalloc-bg-thread yes

# Keyspace Notifications (disabled by default)
notify-keyspace-events ""
EOF

    if [[ -f "$config_file" ]]; then
        log::debug "Redis configuration generated successfully"
        return 0
    else
        log::error "Failed to write Redis configuration file"
        return 1
    fi
}

#######################################
# Get Redis server info
# Returns: Redis info output
#######################################
redis::common::get_info() {
    local cmd=(docker exec "${REDIS_CONTAINER_NAME}" redis-cli info)
    
    if [[ -n "$REDIS_PASSWORD" ]]; then
        cmd=(docker exec "${REDIS_CONTAINER_NAME}" redis-cli -a "$REDIS_PASSWORD" info)
    fi
    
    "${cmd[@]}" 2>/dev/null
}

#######################################
# Get Redis memory usage
# Returns: Memory usage in bytes
#######################################
redis::common::get_memory_usage() {
    redis::common::get_info | grep "used_memory:" | cut -d: -f2 | tr -d '\r'
}

#######################################
# Get connected clients count
# Returns: Number of connected clients
#######################################
redis::common::get_connected_clients() {
    redis::common::get_info | grep "connected_clients:" | cut -d: -f2 | tr -d '\r'
}

#######################################
# Get total commands processed
# Returns: Total commands processed
#######################################
redis::common::get_total_commands() {
    redis::common::get_info | grep "total_commands_processed:" | cut -d: -f2 | tr -d '\r'
}

#######################################
# Format bytes to human readable
# Arguments:
#   $1 - bytes
# Returns: Human readable size
#######################################
redis::common::format_bytes() {
    local bytes=$1
    local units=('B' 'KB' 'MB' 'GB' 'TB')
    local unit_index=0
    
    while (( bytes > 1024 && unit_index < ${#units[@]} - 1 )); do
        bytes=$((bytes / 1024))
        unit_index=$((unit_index + 1))
    done
    
    echo "${bytes}${units[$unit_index]}"
}

#######################################
# Check if port is available
# Arguments:
#   $1 - port number
# Returns: 0 if available, 1 if in use
#######################################
redis::common::is_port_available() {
    local port=$1
    ! netstat -tuln 2>/dev/null | grep -q ":${port} " && ! ss -tuln 2>/dev/null | grep -q ":${port} "
}

#######################################
# Find next available port in range
# Arguments:
#   $1 - start port
#   $2 - end port
# Returns: Available port number or empty if none found
#######################################
redis::common::find_available_port() {
    local start_port=$1
    local end_port=$2
    
    for ((port=start_port; port<=end_port; port++)); do
        if redis::common::is_port_available "$port"; then
            echo "$port"
            return 0
        fi
    done
    
    return 1
}