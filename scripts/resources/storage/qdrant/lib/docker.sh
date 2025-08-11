#!/usr/bin/env bash
# Qdrant Docker Management
# Functions for managing Qdrant Docker container

# Source required utilities
QDRANT_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
# shellcheck disable=SC1091
source "${QDRANT_LIB_DIR}/../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true

#######################################
# Create Docker network if it doesn't exist
# Returns: 0 on success, 1 on failure
#######################################
qdrant::docker::create_network() {
    if ! docker network ls | grep -q "${QDRANT_NETWORK_NAME}"; then
        log::debug "Creating Docker network: ${QDRANT_NETWORK_NAME}"
        if docker network create "${QDRANT_NETWORK_NAME}" >/dev/null 2>&1; then
            log::debug "Docker network created successfully"
            return 0
        else
            log::error "Failed to create Docker network"
            return 1
        fi
    else
        log::debug "Docker network already exists: ${QDRANT_NETWORK_NAME}"
        return 0
    fi
}

#######################################
# Pull Qdrant Docker image
# Returns: 0 on success, 1 on failure
#######################################
qdrant::docker::pull_image() {
    log::info "${MSG_PULLING_IMAGE}"
    
    if docker pull "${QDRANT_IMAGE}" 2>&1 | while read -r line; do
        if [[ "$line" =~ "Pulling from" ]] || [[ "$line" =~ "Status:" ]]; then
            log::debug "$line"
        fi
    done; then
        log::debug "Qdrant image pulled successfully"
        return 0
    else
        log::error "Failed to pull Qdrant image"
        return 1
    fi
}

#######################################
# Create necessary directories
# Returns: 0 on success, 1 on failure
#######################################
qdrant::docker::create_directories() {
    log::info "${MSG_CREATING_DIRECTORIES}"
    
    local directories=(
        "${QDRANT_DATA_DIR}"
        "${QDRANT_CONFIG_DIR}"
        "${QDRANT_SNAPSHOTS_DIR}"
    )
    
    for dir in "${directories[@]}"; do
        if ! mkdir -p "$dir" 2>/dev/null; then
            log::error "Failed to create directory: $dir"
            return 1
        fi
        
        # Fix Docker volume permissions if setup was run with sudo
        docker::fix_volume_permissions "$dir" 2>/dev/null || {
            log::debug "Could not fix Docker volume permissions for $dir, continuing..."
        }
        
        # Set appropriate permissions
        chmod 755 "$dir" 2>/dev/null || true
    done
    
    log::debug "Directories created successfully"
    return 0
}

#######################################
# Create and start Qdrant container
# Returns: 0 on success, 1 on failure
#######################################
qdrant::docker::create_container() {
    # Ensure network exists
    qdrant::docker::create_network || return 1
    
    # Ensure directories exist
    qdrant::docker::create_directories || return 1
    
    log::info "${MSG_STARTING_CONTAINER}"
    
    # Build environment variables
    local env_vars=(
        "-e" "QDRANT__SERVICE__HTTP_PORT=6333"
        "-e" "QDRANT__SERVICE__GRPC_PORT=6334"
        "-e" "QDRANT__STORAGE__OPTIMIZED_SEGMENT_SIZE=${QDRANT_STORAGE_OPTIMIZED_SEGMENT_SIZE}"
        "-e" "QDRANT__STORAGE__MEMMAP_THRESHOLD=${QDRANT_STORAGE_MEMMAP_THRESHOLD}"
        "-e" "QDRANT__STORAGE__INDEXING_THRESHOLD=${QDRANT_STORAGE_INDEXING_THRESHOLD}"
        "-e" "QDRANT__SERVICE__MAX_REQUEST_SIZE_MB=${QDRANT_MAX_REQUEST_SIZE_MB}"
    )
    
    # Add API key if provided
    if [[ -n "${QDRANT_API_KEY:-}" ]]; then
        env_vars+=("-e" "QDRANT__SERVICE__API_KEY=${QDRANT_API_KEY}")
    fi
    
    # Add worker configuration
    if [[ "${QDRANT_MAX_WORKERS}" -gt 0 ]]; then
        env_vars+=("-e" "QDRANT__SERVICE__MAX_WORKERS=${QDRANT_MAX_WORKERS}")
    fi
    
    # Build docker run command
    local docker_cmd=(
        "docker" "run" "-d"
        "--name" "${QDRANT_CONTAINER_NAME}"
        "--network" "${QDRANT_NETWORK_NAME}"
        "-p" "${QDRANT_PORT}:6333"
        "-p" "${QDRANT_GRPC_PORT}:6334"
        "-v" "${QDRANT_DATA_DIR}:/qdrant/storage"
        "-v" "${QDRANT_SNAPSHOTS_DIR}:/qdrant/snapshots"
        "${env_vars[@]}"
        "--restart" "unless-stopped"
        "--health-cmd" "timeout 2 bash -c 'exec 3<>/dev/tcp/localhost/6333 && echo -e \"GET / HTTP/1.0\\r\\n\\r\\n\" >&3 && head -1 <&3 | grep -q \"200 OK\"' || exit 1"
        "--health-interval" "30s"
        "--health-timeout" "10s"
        "--health-retries" "3"
        "${QDRANT_IMAGE}"
    )
    
    # Execute the command
    if "${docker_cmd[@]}" >/dev/null 2>&1; then
        log::debug "Qdrant container created successfully"
        return 0
    else
        log::error "Failed to create Qdrant container"
        return 1
    fi
}

#######################################
# Start existing Qdrant container
# Returns: 0 on success, 1 on failure
#######################################
qdrant::docker::start() {
    if ! qdrant::common::container_exists; then
        log::error "Qdrant container does not exist. Use 'install' action first."
        return 1
    fi
    
    if qdrant::common::is_running; then
        log::info "Qdrant is already running"
        return 0
    fi
    
    log::info "Starting Qdrant container..."
    
    if docker start "${QDRANT_CONTAINER_NAME}" >/dev/null 2>&1; then
        if qdrant::common::wait_for_startup; then
            log::success "${MSG_START_SUCCESS}"
            qdrant::docker::show_connection_info
            return 0
        else
            log::error "${MSG_START_FAILED}"
            return 1
        fi
    else
        log::error "${MSG_START_FAILED}"
        return 1
    fi
}

#######################################
# Stop Qdrant container
# Returns: 0 on success, 1 on failure
#######################################
qdrant::docker::stop() {
    if ! qdrant::common::container_exists; then
        log::warn "Qdrant container does not exist"
        return 0
    fi
    
    if ! qdrant::common::is_running; then
        log::info "Qdrant is already stopped"
        return 0
    fi
    
    log::info "Stopping Qdrant container..."
    
    if docker stop "${QDRANT_CONTAINER_NAME}" >/dev/null 2>&1; then
        log::success "${MSG_STOP_SUCCESS}"
        return 0
    else
        log::error "Failed to stop Qdrant container"
        return 1
    fi
}

#######################################
# Restart Qdrant container
# Returns: 0 on success, 1 on failure
#######################################
qdrant::docker::restart() {
    log::info "Restarting Qdrant..."
    
    if qdrant::docker::stop && qdrant::docker::start; then
        log::success "${MSG_RESTART_SUCCESS}"
        return 0
    else
        log::error "Failed to restart Qdrant"
        return 1
    fi
}

#######################################
# Remove Qdrant container
# Arguments:
#   $1 - Whether to preserve data (true/false)
# Returns: 0 on success, 1 on failure
#######################################
qdrant::docker::remove_container() {
    local preserve_data="${1:-true}"
    
    # Stop container if running
    if qdrant::common::is_running; then
        qdrant::docker::stop || return 1
    fi
    
    # Remove container if exists
    if qdrant::common::container_exists; then
        log::info "Removing Qdrant container..."
        if docker rm "${QDRANT_CONTAINER_NAME}" >/dev/null 2>&1; then
            log::debug "Container removed successfully"
        else
            log::error "Failed to remove container"
            return 1
        fi
    fi
    
    # Remove data if requested
    if [[ "$preserve_data" == "false" ]]; then
        log::info "Removing Qdrant data..."
        trash::safe_remove "${QDRANT_DATA_DIR}" --production 2>/dev/null || true
        trash::safe_remove "${QDRANT_CONFIG_DIR}" --production 2>/dev/null || true
        trash::safe_remove "${QDRANT_SNAPSHOTS_DIR}" --production 2>/dev/null || true
        log::debug "Data directories removed"
    fi
    
    # Remove network if no other containers are using it
    if docker network ls | grep -q "${QDRANT_NETWORK_NAME}"; then
        local containers_using_network
        containers_using_network=$(docker network inspect "${QDRANT_NETWORK_NAME}" --format='{{len .Containers}}' 2>/dev/null || echo "1")
        
        if [[ "$containers_using_network" == "0" ]]; then
            log::debug "Removing Docker network: ${QDRANT_NETWORK_NAME}"
            docker network rm "${QDRANT_NETWORK_NAME}" >/dev/null 2>&1 || true
        fi
    fi
    
    return 0
}

#######################################
# Show connection information
#######################################
qdrant::docker::show_connection_info() {
    echo
    log::info "${MSG_WEB_UI_AVAILABLE} ${QDRANT_BASE_URL}/dashboard"
    log::info "${MSG_HELP_API}"
    log::info "${MSG_HELP_GRPC}"
    
    if [[ -n "${QDRANT_API_KEY:-}" ]]; then
        log::info "Authentication: API key configured"
    else
        log::warn "${MSG_NO_API_KEY}"
    fi
    
    echo
    log::info "${MSG_HELP_COLLECTIONS}"
    echo
}

#######################################
# Get container resource usage
# Outputs: Resource usage statistics
#######################################
qdrant::docker::get_resource_usage() {
    if ! qdrant::common::is_running; then
        echo "Container is not running"
        return 1
    fi
    
    docker stats "${QDRANT_CONTAINER_NAME}" --no-stream --format="table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}\t{{.BlockIO}}"
}

#######################################
# Check if Docker is available and running
# Returns: 0 if available, 1 if not
#######################################
qdrant::docker::check_docker() {
    if ! command -v docker >/dev/null 2>&1; then
        log::error "${MSG_DOCKER_NOT_FOUND}"
        return 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running"
        return 1
    fi
    
    return 0
}