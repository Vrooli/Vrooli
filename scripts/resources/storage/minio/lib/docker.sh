#!/usr/bin/env bash
# MinIO Docker Management - Ultra-Simplified
# Uses docker-resource-utils.sh for minimal boilerplate

# Source var.sh to get proper directory variables
_MINIO_DOCKER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${_MINIO_DOCKER_DIR}/../../../../lib/utils/var.sh"

# Source shared libraries
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"

#######################################
# Create Docker network if it doesn't exist
# Returns: 0 on success, 1 on failure
#######################################
minio::docker::create_network() {
    docker::create_network "${MINIO_NETWORK_NAME}"
}


#######################################
# Create and start MinIO container
# Returns: 0 on success, 1 on failure
#######################################
minio::docker::create_container() {
    # Ensure directories exist
    minio::common::create_directories || return 1
    
    # Generate credentials if needed
    minio::common::generate_credentials || return 1
    
    # Pull image if needed
    log::info "${MSG_PULLING_IMAGE}"
    docker::pull_image "${MINIO_IMAGE}"
    
    log::info "${MSG_STARTING_CONTAINER}"
    
    # Validate required environment variables
    docker_resource::validate_env_vars "MINIO_ROOT_USER" "MINIO_ROOT_PASSWORD" "MINIO_IMAGE" || return 1
    
    # Prepare environment variables array
    local env_vars=(
        "MINIO_ROOT_USER=${MINIO_ROOT_USER}"
        "MINIO_ROOT_PASSWORD=${MINIO_ROOT_PASSWORD}"
        "MINIO_BROWSER=${MINIO_BROWSER:-on}"
        "MINIO_REGION=${MINIO_REGION:-us-east-1}"
    )
    
    # Port mappings for dual-port MinIO setup
    local port_mappings="${MINIO_PORT}:9000 ${MINIO_CONSOLE_PORT}:9001"
    
    # Volumes
    local volumes="${MINIO_DATA_DIR}:/data ${MINIO_CONFIG_DIR}:/root/.minio"
    
    # Health check command with MinIO-specific timeout
    local health_cmd="curl -f http://localhost:9000/minio/health/live || exit 1"
    export DOCKER_HEALTH_TIMEOUT="10s"  # Override default for MinIO
    
    # MinIO server command arguments
    local entrypoint_cmd=(
        "server" "/data" "--console-address" ":9001"
    )
    
    # Use advanced creation function
    if docker_resource::create_service_advanced \
        "$MINIO_CONTAINER_NAME" \
        "$MINIO_IMAGE" \
        "$port_mappings" \
        "$MINIO_NETWORK_NAME" \
        "$volumes" \
        "env_vars" \
        "" \
        "$health_cmd" \
        "entrypoint_cmd"; then
        
        log::debug "MinIO container created successfully"
        
        # Wait for container to initialize
        sleep "${MINIO_INITIALIZATION_WAIT:-10}"
        return 0
    else
        log::error "Failed to create MinIO container"
        return 1
    fi
}

#######################################
# Start MinIO container
# Returns: 0 on success, 1 on failure
#######################################
minio::docker::start() {
    if ! docker::container_exists "$MINIO_CONTAINER_NAME"; then
        log::error "MinIO container does not exist. Run install first."
        return 1
    fi
    
    log::info "Starting MinIO container..."
    
    if docker::start_container "$MINIO_CONTAINER_NAME"; then
        # Wait for container to be ready
        if minio::common::wait_for_ready; then
            log::success "${MSG_START_SUCCESS}"
            log::info "${MSG_CONSOLE_AVAILABLE}: ${MINIO_CONSOLE_URL}"
            return 0
        else
            log::error "${MSG_START_FAILED}"
            return 1
        fi
    else
        log::error "Failed to start MinIO container"
        return 1
    fi
}

#######################################
# Stop MinIO container
# Returns: 0 on success, 1 on failure
#######################################
minio::docker::stop() {
    if ! docker::container_exists "$MINIO_CONTAINER_NAME"; then
        log::warn "MinIO container does not exist"
        return 0
    fi
    
    log::info "Stopping MinIO container..."
    
    if docker::stop_container "$MINIO_CONTAINER_NAME"; then
        log::success "${MSG_STOP_SUCCESS}"
        return 0
    else
        log::error "${MSG_STOP_FAILED:-Failed to stop MinIO container}"
        return 1
    fi
}

#######################################
# Restart MinIO container
# Returns: 0 on success, 1 on failure
#######################################
minio::docker::restart() {
    log::info "Restarting MinIO..."
    
    if docker::restart_container "$MINIO_CONTAINER_NAME"; then
        # Wait for container to be ready
        if minio::common::wait_for_ready; then
            log::success "${MSG_RESTART_SUCCESS}"
            log::info "${MSG_CONSOLE_AVAILABLE}: ${MINIO_CONSOLE_URL}"
            return 0
        else
            log::error "MinIO restarted but failed to become ready"
            return 1
        fi
    else
        log::error "Failed to restart MinIO"
        return 1
    fi
}

#######################################
# Remove MinIO container
# Arguments:
#   $1 - Force removal (optional, default: false)
# Returns: 0 on success, 1 on failure
#######################################
minio::docker::remove() {
    local force=${1:-false}
    
    log::info "Removing MinIO container..."
    
    if docker::remove_container "$MINIO_CONTAINER_NAME" "$force"; then
        log::debug "MinIO container removed successfully"
        return 0
    else
        log::error "Failed to remove MinIO container"
        return 1
    fi
}
