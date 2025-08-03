#!/usr/bin/env bash
# Port Manager - Dynamic port allocation and management for test isolation
# Provides safe port allocation, tracking, and cleanup for parallel test execution

# Prevent duplicate loading
if [[ "${VROOLI_PORT_MANAGER_LOADED:-}" == "true" ]]; then
    return 0
fi
export VROOLI_PORT_MANAGER_LOADED="true"

# Load dependencies
SHARED_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SHARED_DIR/logging.bash"
source "$SHARED_DIR/utils.bash"

# Port allocation configuration
readonly PORT_RANGE_START="${VROOLI_PORT_RANGE_START:-30000}"
readonly PORT_RANGE_END="${VROOLI_PORT_RANGE_END:-32000}"
readonly PORT_LOCK_DIR="${VROOLI_PORT_LOCK_DIR:-/tmp/vrooli-port-locks}"
readonly PORT_ALLOCATION_TIMEOUT="${VROOLI_PORT_ALLOCATION_TIMEOUT:-30}"

# Port allocation tracking (for cleanup)
declare -a ALLOCATED_PORTS=()

#######################################
# Initialize port manager
# Creates lock directory and validates configuration
# Arguments: None
# Returns: 0 on success, 1 on failure
#######################################
vrooli_port_init() {
    vrooli_log_debug "PORT" "Initializing port manager"
    
    # Create lock directory if it doesn't exist
    if ! mkdir -p "$PORT_LOCK_DIR" 2>/dev/null; then
        vrooli_log_error "PORT" "Failed to create port lock directory: $PORT_LOCK_DIR"
        return 1
    fi
    
    # Clean up stale locks (older than 1 hour)
    find "$PORT_LOCK_DIR" -name "*.lock" -mmin +60 -delete 2>/dev/null || true
    
    vrooli_log_info "PORT" "Port manager initialized (range: $PORT_RANGE_START-$PORT_RANGE_END)"
    return 0
}

#######################################
# Allocate a free port within the configured range
# Uses file-based locking for parallel safety
# Arguments:
#   $1 - Service name (optional, for tracking)
# Returns: Allocated port number
# Outputs: Port number to stdout
#######################################
vrooli_port_allocate() {
    local service_name="${1:-test}"
    local port=""
    local attempts=0
    local max_attempts=100
    
    vrooli_log_debug "PORT" "Allocating port for service: $service_name"
    
    while [[ -z "$port" && $attempts -lt $max_attempts ]]; do
        # Generate random port in range
        local candidate=$((PORT_RANGE_START + RANDOM % (PORT_RANGE_END - PORT_RANGE_START)))
        local lock_file="$PORT_LOCK_DIR/$candidate.lock"
        
        # Try to acquire lock (atomic operation)
        if (set -C; echo "$$:$service_name:$(date +%s)" > "$lock_file") 2>/dev/null; then
            # Verify port is actually free
            if ! vrooli_port_is_in_use "$candidate"; then
                port=$candidate
                ALLOCATED_PORTS+=("$candidate")
                vrooli_log_info "PORT" "Allocated port $port for $service_name"
                echo "$port"
                return 0
            else
                # Port in use, release lock
                rm -f "$lock_file"
                vrooli_log_debug "PORT" "Port $candidate is in use, trying another"
            fi
        fi
        
        ((attempts++))
        sleep 0.1
    done
    
    vrooli_log_error "PORT" "Failed to allocate port after $max_attempts attempts"
    return 1
}

#######################################
# Check if a port is currently in use
# Arguments:
#   $1 - Port number
# Returns: 0 if in use, 1 if free
#######################################
vrooli_port_is_in_use() {
    local port="$1"
    
    # Check with nc (netcat)
    if command -v nc >/dev/null 2>&1; then
        nc -z -w1 localhost "$port" 2>/dev/null
        return $?
    fi
    
    # Fallback to lsof
    if command -v lsof >/dev/null 2>&1; then
        lsof -i:"$port" >/dev/null 2>&1
        return $?
    fi
    
    # Fallback to ss
    if command -v ss >/dev/null 2>&1; then
        ss -tln | grep -q ":$port "
        return $?
    fi
    
    # Fallback to /proc/net/tcp (Linux only)
    if [[ -r /proc/net/tcp ]]; then
        local hex_port
        hex_port=$(printf "%04X" "$port")
        grep -q ":$hex_port " /proc/net/tcp 2>/dev/null
        return $?
    fi
    
    # Conservative: assume in use if we can't check
    vrooli_log_warn "PORT" "Cannot determine if port $port is in use (no suitable tools)"
    return 0
}

#######################################
# Release a previously allocated port
# Arguments:
#   $1 - Port number
# Returns: 0 on success
#######################################
vrooli_port_release() {
    local port="$1"
    local lock_file="$PORT_LOCK_DIR/$port.lock"
    
    vrooli_log_debug "PORT" "Releasing port $port"
    
    # Remove from tracking array
    local new_ports=()
    for p in "${ALLOCATED_PORTS[@]}"; do
        if [[ "$p" != "$port" ]]; then
            new_ports+=("$p")
        fi
    done
    ALLOCATED_PORTS=("${new_ports[@]}")
    
    # Release lock
    if [[ -f "$lock_file" ]]; then
        rm -f "$lock_file"
        vrooli_log_info "PORT" "Released port $port"
    else
        vrooli_log_debug "Port $port was not locked"
    fi
    
    return 0
}

#######################################
# Release all ports allocated by this process
# Call this in cleanup/teardown
# Arguments: None
# Returns: 0
#######################################
vrooli_port_release_all() {
    vrooli_log_info "Releasing all allocated ports"
    
    for port in "${ALLOCATED_PORTS[@]}"; do
        vrooli_port_release "$port"
    done
    
    ALLOCATED_PORTS=()
    return 0
}

#######################################
# Get the default port for a service from configuration
# Falls back to well-known ports if not configured
# Arguments:
#   $1 - Service name
# Returns: Port number
# Outputs: Port number to stdout
#######################################
vrooli_port_get_default() {
    local service="$1"
    
    # Try to get from configuration
    if declare -f vrooli_config_get_port >/dev/null 2>&1; then
        local port
        port=$(vrooli_config_get_port "$service" 2>/dev/null)
        if [[ -n "$port" && "$port" != "null" ]]; then
            echo "$port"
            return 0
        fi
    fi
    
    # Fallback to well-known ports
    case "$service" in
        postgres|postgresql) echo "5432" ;;
        redis) echo "6379" ;;
        ollama) echo "11434" ;;
        n8n) echo "5678" ;;
        comfyui) echo "8188" ;;
        whisper) echo "9000" ;;
        windmill) echo "8000" ;;
        huginn) echo "3000" ;;
        node-red) echo "1880" ;;
        searxng) echo "8080" ;;
        minio) echo "9000" ;;
        vault) echo "8200" ;;
        qdrant) echo "6333" ;;
        questdb) echo "9000" ;;
        judge0) echo "2358" ;;
        unstructured-io) echo "8000" ;;
        agent-s2) echo "8501" ;;
        browserless) echo "3000" ;;
        *) 
            vrooli_log_warn "Unknown service: $service, cannot determine default port"
            echo ""
            ;;
    esac
}

#######################################
# Wait for a service to be available on a port
# Arguments:
#   $1 - Port number
#   $2 - Timeout in seconds (optional, default 30)
# Returns: 0 if service available, 1 if timeout
#######################################
vrooli_port_wait_for_service() {
    local port="$1"
    local timeout="${2:-30}"
    local elapsed=0
    
    vrooli_log_info "Waiting for service on port $port (timeout: ${timeout}s)"
    
    while [[ $elapsed -lt $timeout ]]; do
        if vrooli_port_is_in_use "$port"; then
            vrooli_log_info "Service is available on port $port"
            return 0
        fi
        
        sleep 1
        ((elapsed++))
        
        # Progress indication every 5 seconds
        if [[ $((elapsed % 5)) -eq 0 ]]; then
            vrooli_log_debug "Still waiting for port $port... (${elapsed}s/${timeout}s)"
        fi
    done
    
    vrooli_log_error "Timeout waiting for service on port $port"
    return 1
}

#######################################
# Get all ports currently locked by this test run
# Useful for debugging and cleanup validation
# Arguments: None
# Returns: 0
# Outputs: List of ports to stdout
#######################################
vrooli_port_list_allocated() {
    if [[ ${#ALLOCATED_PORTS[@]} -eq 0 ]]; then
        vrooli_log_debug "No ports currently allocated"
        return 0
    fi
    
    vrooli_log_info "Currently allocated ports: ${ALLOCATED_PORTS[*]}"
    printf '%s\n' "${ALLOCATED_PORTS[@]}"
}

#######################################
# Clean up all stale port locks
# Removes locks from terminated processes
# Arguments: None
# Returns: 0
#######################################
vrooli_port_cleanup_stale() {
    vrooli_log_info "Cleaning up stale port locks"
    
    local cleaned=0
    for lock_file in "$PORT_LOCK_DIR"/*.lock; do
        [[ -f "$lock_file" ]] || continue
        
        # Extract PID from lock file
        local lock_info
        lock_info=$(cat "$lock_file" 2>/dev/null)
        local pid="${lock_info%%:*}"
        
        # Check if process is still running
        if [[ -n "$pid" ]] && ! kill -0 "$pid" 2>/dev/null; then
            rm -f "$lock_file"
            ((cleaned++))
            vrooli_log_debug "Removed stale lock: $(basename "$lock_file")"
        fi
    done
    
    vrooli_log_info "Cleaned up $cleaned stale port locks"
    return 0
}

#######################################
# Export functions for use in tests
#######################################
export -f vrooli_port_init vrooli_port_allocate vrooli_port_is_in_use
export -f vrooli_port_release vrooli_port_release_all vrooli_port_get_default
export -f vrooli_port_wait_for_service vrooli_port_list_allocated
export -f vrooli_port_cleanup_stale

# Initialize on load
vrooli_port_init

vrooli_log_info "PORT" "Port manager loaded successfully"