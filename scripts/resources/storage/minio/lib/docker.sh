#!/usr/bin/env bash
# MinIO Docker Management
# Functions for managing MinIO Docker container

#######################################
# Create Docker network if it doesn't exist
# Returns: 0 on success, 1 on failure
#######################################
minio::docker::create_network() {
    if ! docker network ls | grep -q "^${MINIO_NETWORK_NAME}"; then
        log::debug "Creating Docker network: ${MINIO_NETWORK_NAME}"
        if docker network create "${MINIO_NETWORK_NAME}" >/dev/null 2>&1; then
            log::debug "Docker network created successfully"
            return 0
        else
            log::error "Failed to create Docker network"
            return 1
        fi
    else
        log::debug "Docker network already exists: ${MINIO_NETWORK_NAME}"
        return 0
    fi
}

#######################################
# Pull MinIO Docker image
# Returns: 0 on success, 1 on failure
#######################################
minio::docker::pull_image() {
    log::info "${MSG_PULLING_IMAGE}"
    
    if docker pull "${MINIO_IMAGE}" 2>&1 | while read -r line; do
        if [[ "$line" =~ "Pulling from" ]] || [[ "$line" =~ "Status:" ]]; then
            log::debug "$line"
        fi
    done; then
        log::debug "MinIO image pulled successfully"
        return 0
    else
        log::error "Failed to pull MinIO image"
        return 1
    fi
}

#######################################
# Create and start MinIO container
# Returns: 0 on success, 1 on failure
#######################################
minio::docker::create_container() {
    # Ensure network exists
    minio::docker::create_network || return 1
    
    # Ensure directories exist
    minio::common::create_directories
    
    # Generate credentials if needed
    minio::common::generate_credentials
    
    log::info "${MSG_STARTING_CONTAINER}"
    
    # Build docker run command
    local docker_cmd=(
        "docker" "run" "-d"
        "--name" "${MINIO_CONTAINER_NAME}"
        "--network" "${MINIO_NETWORK_NAME}"
        "-p" "${MINIO_PORT}:9000"
        "-p" "${MINIO_CONSOLE_PORT}:9001"
        "-v" "${MINIO_DATA_DIR}:/data"
        "-v" "${MINIO_CONFIG_DIR}:/root/.minio"
        "-e" "MINIO_ROOT_USER=${MINIO_ROOT_USER}"
        "-e" "MINIO_ROOT_PASSWORD=${MINIO_ROOT_PASSWORD}"
        "-e" "MINIO_BROWSER=${MINIO_BROWSER}"
        "-e" "MINIO_REGION=${MINIO_REGION}"
        "--restart" "unless-stopped"
        "--health-cmd" "curl -f http://localhost:9000/minio/health/live || exit 1"
        "--health-interval" "30s"
        "--health-timeout" "10s"
        "--health-retries" "3"
        "${MINIO_IMAGE}"
        "server" "/data"
        "--console-address" ":9001"
    )
    
    # Run the container
    if "${docker_cmd[@]}" >/dev/null 2>&1; then
        log::debug "MinIO container created successfully"
        
        # Wait a moment for container to initialize
        sleep "${MINIO_INITIALIZATION_WAIT}"
        
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
    if ! minio::common::container_exists; then
        log::error "MinIO container does not exist. Run install first."
        return 1
    fi
    
    if minio::common::is_running; then
        log::info "MinIO is already running"
        return 0
    fi
    
    log::info "Starting MinIO container..."
    
    if docker start "${MINIO_CONTAINER_NAME}" >/dev/null 2>&1; then
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
    if ! minio::common::container_exists; then
        log::warn "MinIO container does not exist"
        return 0
    fi
    
    if ! minio::common::is_running; then
        log::info "MinIO is not running"
        return 0
    fi
    
    log::info "Stopping MinIO container..."
    
    if docker stop "${MINIO_CONTAINER_NAME}" >/dev/null 2>&1; then
        log::success "${MSG_STOP_SUCCESS}"
        return 0
    else
        log::error "Failed to stop MinIO container"
        return 1
    fi
}

#######################################
# Restart MinIO container
# Returns: 0 on success, 1 on failure
#######################################
minio::docker::restart() {
    log::info "Restarting MinIO..."
    
    # Stop if running
    if minio::common::is_running; then
        minio::docker::stop || return 1
        sleep 2
    fi
    
    # Start again
    if minio::docker::start; then
        log::success "${MSG_RESTART_SUCCESS}"
        return 0
    else
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
    
    if ! minio::common::container_exists; then
        log::debug "MinIO container does not exist"
        return 0
    fi
    
    # Stop container if running
    if minio::common::is_running; then
        minio::docker::stop || {
            if [[ "$force" == "true" ]]; then
                log::warn "Force stopping container"
                docker kill "${MINIO_CONTAINER_NAME}" >/dev/null 2>&1 || true
            else
                return 1
            fi
        }
    fi
    
    log::info "Removing MinIO container..."
    
    if docker rm "${MINIO_CONTAINER_NAME}" >/dev/null 2>&1; then
        log::debug "MinIO container removed successfully"
        return 0
    else
        log::error "Failed to remove MinIO container"
        return 1
    fi
}

#######################################
# Execute command in MinIO container
# Arguments:
#   $@ - Command to execute
# Returns: Command exit code
#######################################
minio::docker::exec() {
    if ! minio::common::is_running; then
        log::error "MinIO container is not running"
        return 1
    fi
    
    docker exec "${MINIO_CONTAINER_NAME}" "$@"
}

#######################################
# Get container statistics
# Output: JSON with container stats
#######################################
minio::docker::stats() {
    if ! minio::common::is_running; then
        echo "{\"error\": \"Container not running\"}"
        return 1
    fi
    
    docker stats "${MINIO_CONTAINER_NAME}" --no-stream --format "json" 2>/dev/null || echo "{}"
}

#######################################
# Check Docker availability
# Returns: 0 if available, 1 if not
#######################################
minio::docker::check_docker() {
    if ! command -v docker >/dev/null 2>&1; then
        log::error "Docker is not installed"
        log::info "Please install Docker: https://docs.docker.com/get-docker/"
        return 1
    fi
    
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running"
        log::info "Please start Docker and try again"
        return 1
    fi
    
    log::debug "Docker is available and running"
    return 0
}