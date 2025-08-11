#!/usr/bin/env bash
# Qdrant Common Utilities
# Shared functions used across Qdrant management scripts

# Source var.sh for directory variables
# shellcheck disable=SC1091
source "$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_UTILS_DIR}/sudo.sh" 2>/dev/null || true

#######################################
# Check if Qdrant container exists
# Returns: 0 if exists, 1 if not
#######################################
qdrant::common::container_exists() {
    docker ps -a --format "table {{.Names}}" | grep -q "^${QDRANT_CONTAINER_NAME}$"
}

#######################################
# Check if Qdrant container is running
# Returns: 0 if running, 1 if not
#######################################
qdrant::common::is_running() {
    docker ps --format "table {{.Names}}" | grep -q "^${QDRANT_CONTAINER_NAME}$"
}

#######################################
# Check if a port is available
# Arguments:
#   $1 - Port number to check
# Returns: 0 if available, 1 if in use
#######################################
qdrant::common::is_port_available() {
    local port=$1
    
    if command -v lsof >/dev/null 2>&1; then
        ! lsof -Pi :"${port}" -sTCP:LISTEN -t >/dev/null 2>&1
    else
        ! netstat -tuln 2>/dev/null | grep -q ":${port} "
    fi
}

#######################################
# Check if both Qdrant ports are available
# Returns: 0 if both available, 1 if any in use
#######################################
qdrant::common::check_ports() {
    local rest_port_available=true
    local grpc_port_available=true
    
    if ! qdrant::common::is_port_available "${QDRANT_PORT}"; then
        rest_port_available=false
        log::warn "Port ${QDRANT_PORT} is already in use"
    fi
    
    if ! qdrant::common::is_port_available "${QDRANT_GRPC_PORT}"; then
        grpc_port_available=false
        log::warn "gRPC port ${QDRANT_GRPC_PORT} is already in use"
    fi
    
    if [[ "$rest_port_available" == "true" && "$grpc_port_available" == "true" ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Get Qdrant container status
# Outputs: Status description
# Returns: 0 if container exists, 1 if not
#######################################
qdrant::common::get_container_status() {
    if ! qdrant::common::container_exists; then
        echo "Container does not exist"
        return 1
    fi
    
    if qdrant::common::is_running; then
        echo "Running"
    else
        echo "Stopped"
    fi
}

#######################################
# Check if Qdrant API is responding
# Returns: 0 if healthy, 1 if not
#######################################
qdrant::common::is_api_healthy() {
    local auth_header=""
    if [[ -n "${QDRANT_API_KEY:-}" ]]; then
        auth_header="-H \"api-key: ${QDRANT_API_KEY}\""
    fi
    
    if eval "curl -f -s --max-time ${QDRANT_API_TIMEOUT} ${auth_header} ${QDRANT_BASE_URL}/ >/dev/null 2>&1"; then
        return 0
    else
        return 1
    fi
}

#######################################
# Check if gRPC API is responding
# Returns: 0 if responding, 1 if not
#######################################
qdrant::common::is_grpc_healthy() {
    # Check if gRPC port is accepting connections
    if command -v nc >/dev/null 2>&1; then
        nc -z localhost "${QDRANT_GRPC_PORT}" 2>/dev/null
    elif command -v telnet >/dev/null 2>&1; then
        timeout 3 telnet localhost "${QDRANT_GRPC_PORT}" </dev/null >/dev/null 2>&1
    else
        # Fallback: check if port is listening
        ! qdrant::common::is_port_available "${QDRANT_GRPC_PORT}"
    fi
}

#######################################
# Wait for Qdrant to start up
# Arguments:
#   $1 - Maximum wait time in seconds (optional, default from config)
# Returns: 0 if started successfully, 1 if timeout
#######################################
qdrant::common::wait_for_startup() {
    local max_wait="${1:-$QDRANT_STARTUP_MAX_WAIT}"
    local elapsed=0
    
    log::info "Waiting for Qdrant to start (max ${max_wait}s)..."
    
    while [[ $elapsed -lt $max_wait ]]; do
        if qdrant::common::is_api_healthy; then
            log::success "Qdrant is ready (${elapsed}s)"
            return 0
        fi
        
        sleep "${QDRANT_STARTUP_WAIT_INTERVAL}"
        elapsed=$((elapsed + QDRANT_STARTUP_WAIT_INTERVAL))
        
        # Show progress every 10 seconds
        if [[ $((elapsed % 10)) -eq 0 ]]; then
            log::info "Still waiting for Qdrant... (${elapsed}s/${max_wait}s)"
        fi
    done
    
    log::error "Qdrant failed to start within ${max_wait} seconds"
    return 1
}

#######################################
# Show Qdrant container logs
# Arguments:
#   $1 - Number of lines to show (optional, default: 50)
#######################################
qdrant::common::show_logs() {
    local lines="${1:-50}"
    
    if ! qdrant::common::container_exists; then
        log::error "Qdrant container does not exist"
        return 1
    fi
    
    log::info "Showing last ${lines} lines of Qdrant logs:"
    docker logs --tail "$lines" "${QDRANT_CONTAINER_NAME}" 2>&1
}

#######################################
# Get Qdrant version information
# Outputs: Version information
# Returns: 0 if successful, 1 if failed
#######################################
qdrant::common::get_version() {
    if ! qdrant::common::is_running; then
        echo "Qdrant is not running"
        return 1
    fi
    
    local auth_header=""
    if [[ -n "${QDRANT_API_KEY:-}" ]]; then
        auth_header="-H \"api-key: ${QDRANT_API_KEY}\""
    fi
    
    local version_info
    if version_info=$(eval "curl -s --max-time ${QDRANT_API_TIMEOUT} ${auth_header} ${QDRANT_BASE_URL}/ | jq -r '.version // \"unknown\"' 2>/dev/null"); then
        echo "$version_info"
        return 0
    else
        echo "Unable to retrieve version"
        return 1
    fi
}

#######################################
# Check disk space in Qdrant data directory
# Returns: 0 if sufficient space, 1 if low space
#######################################
qdrant::common::check_disk_space() {
    local data_dir="${QDRANT_DATA_DIR}"
    local min_space_gb="${QDRANT_MIN_DISK_SPACE_GB}"
    
    # Create directory if it doesn't exist
    if sudo::is_running_as_sudo && [[ "$data_dir" == "${HOME}/"* || "$data_dir" == "/home/"* ]]; then
        sudo::mkdir_as_user "$data_dir"
    else
        mkdir -p "$data_dir"
    fi
    
    # Check available space
    local available_kb
    available_kb=$(df "$data_dir" | awk 'NR==2 {print $4}')
    local available_gb=$((available_kb / 1024 / 1024))
    
    if [[ $available_gb -lt $min_space_gb ]]; then
        log::warn "Low disk space: ${available_gb}GB available (minimum: ${min_space_gb}GB)"
        return 1
    fi
    
    return 0
}

#######################################
# Clean up temporary files and resources
#######################################
qdrant::common::cleanup() {
    # Remove any temporary files created during operations
    trash::safe_remove /tmp/qdrant_*.json --temp 2>/dev/null || true
    trash::safe_remove /tmp/qdrant_*.tmp --temp 2>/dev/null || true
}

#######################################
# Get Qdrant process information
# Outputs: Process information if container is running
#######################################
qdrant::common::get_process_info() {
    if ! qdrant::common::is_running; then
        echo "Qdrant container is not running"
        return 1
    fi
    
    echo "Container Information:"
    docker inspect "${QDRANT_CONTAINER_NAME}" --format='{{json .State}}' | jq '.'
    echo
    echo "Resource Usage:"
    docker stats "${QDRANT_CONTAINER_NAME}" --no-stream --format="table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}"
}