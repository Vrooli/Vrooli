#!/usr/bin/env bash
# Agent S2 Docker Operations
# Docker container and image management

#######################################
# Build Agent S2 Docker image
# Returns: 0 if successful, 1 if failed
#######################################
agents2::docker_build() {
    log::info "$MSG_BUILDING_IMAGE"
    
    local build_dir="${SCRIPT_DIR}/docker"
    
    # Ensure Dockerfile exists
    if [[ ! -f "$build_dir/Dockerfile" ]]; then
        log::error "Dockerfile not found at: $build_dir/Dockerfile"
        return 1
    fi
    
    # Build with proper arguments
    local build_args=(
        --build-arg "VNC_PASSWORD=$AGENTS2_VNC_PASSWORD"
        --build-arg "USER_ID=$AGENTS2_USER_ID"
        --build-arg "GROUP_ID=$AGENTS2_GROUP_ID"
        -t "$AGENTS2_IMAGE_NAME"
        "$build_dir"
    )
    
    if docker build "${build_args[@]}"; then
        log::success "$MSG_BUILD_SUCCESS"
        return 0
    else
        log::error "$MSG_BUILD_FAILED"
        return 1
    fi
}

#######################################
# Create Docker network for Agent S2
# Returns: 0 if successful or exists, 1 if failed
#######################################
agents2::docker_create_network() {
    if docker network ls --format "{{.Name}}" | grep -q "^${AGENTS2_NETWORK_NAME}$"; then
        log::info "$MSG_NETWORK_EXISTS: $AGENTS2_NETWORK_NAME"
        return 0
    fi
    
    log::info "$MSG_CREATING_NETWORK"
    if docker network create "$AGENTS2_NETWORK_NAME"; then
        log::success "$MSG_NETWORK_CREATED"
        return 0
    else
        log::error "$MSG_NETWORK_FAILED"
        return 1
    fi
}

#######################################
# Start Agent S2 container
# Returns: 0 if successful, 1 if failed
#######################################
agents2::docker_start() {
    if agents2::is_running; then
        if [[ "$FORCE" != "yes" ]]; then
            log::info "$MSG_CONTAINER_RUNNING"
            return 0
        else
            log::info "Force specified, stopping existing container..."
            agents2::docker_stop
        fi
    fi
    
    # Remove existing stopped container
    if agents2::container_exists && ! agents2::is_running; then
        log::info "Removing existing stopped container..."
        docker rm "$AGENTS2_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    # Ensure network exists
    agents2::docker_create_network || return 1
    
    # Save environment configuration
    agents2::save_env_config || return 1
    
    log::info "$MSG_STARTING_CONTAINER"
    
    # Build Docker run command
    local docker_cmd=(
        docker run -d
        --name "$AGENTS2_CONTAINER_NAME"
        --restart unless-stopped
        --network "$AGENTS2_NETWORK_NAME"
        -p "${AGENTS2_PORT}:4113"
        -p "${AGENTS2_VNC_PORT}:5900"
        -e "AGENTS2_API_KEY=$AGENTS2_API_KEY"
        -e "AGENTS2_LLM_PROVIDER=$AGENTS2_LLM_PROVIDER"
        -e "AGENTS2_LLM_MODEL=$AGENTS2_LLM_MODEL"
        -e "DISPLAY=$AGENTS2_DISPLAY"
        -v "${AGENTS2_DATA_DIR}/logs:/var/log/supervisor:rw"
        -v "${AGENTS2_DATA_DIR}/cache:/home/agents2/.cache:rw"
        -v "${AGENTS2_DATA_DIR}/models:/home/agents2/.agent-s2/models:rw"
        --security-opt "$AGENTS2_SECURITY_OPT"
        --shm-size "$AGENTS2_SHM_SIZE"
        --memory "$AGENTS2_MEMORY_LIMIT"
        --cpus "$AGENTS2_CPU_LIMIT"
    )
    
    # Add host display access if enabled (security risk)
    if [[ "$AGENTS2_ENABLE_HOST_DISPLAY" == "yes" ]]; then
        log::warn "⚠️  Host display access enabled - this is a security risk!"
        docker_cmd+=(
            -v "/tmp/.X11-unix:/tmp/.X11-unix:rw"
            -e "DISPLAY=${DISPLAY:-:0}"
        )
    fi
    
    # Add the image name
    docker_cmd+=("$AGENTS2_IMAGE_NAME")
    
    if "${docker_cmd[@]}"; then
        log::success "$MSG_CONTAINER_STARTED"
        
        # Wait for service to be ready
        sleep "$AGENTS2_INITIALIZATION_WAIT"
        
        if agents2::wait_for_ready; then
            log::success "$MSG_SERVICE_HEALTHY"
            return 0
        else
            log::warn "$MSG_HEALTH_CHECK_FAILED"
            log::info "The service may still be initializing. Check logs with: $0 --action logs"
            return 0
        fi
    else
        log::error "Failed to start container"
        return 1
    fi
}

#######################################
# Stop Agent S2 container
# Returns: 0 if successful, 1 if failed
#######################################
agents2::docker_stop() {
    if ! agents2::is_running; then
        log::info "$MSG_CONTAINER_NOT_RUNNING"
        return 0
    fi
    
    log::info "$MSG_STOPPING_CONTAINER"
    if docker stop "$AGENTS2_CONTAINER_NAME" >/dev/null 2>&1; then
        log::success "$MSG_CONTAINER_STOPPED"
        return 0
    else
        log::error "Failed to stop container"
        return 1
    fi
}

#######################################
# Restart Agent S2 container
# Returns: 0 if successful, 1 if failed
#######################################
agents2::docker_restart() {
    log::info "Restarting Agent S2..."
    agents2::docker_stop
    sleep 2
    agents2::docker_start
}

#######################################
# Show container logs
# Arguments:
#   $1 - number of lines to show (optional, default: follow)
#######################################
agents2::docker_logs() {
    if ! agents2::container_exists; then
        log::error "Agent S2 container does not exist"
        return 1
    fi
    
    if [[ -n "${1:-}" ]]; then
        docker logs --tail "$1" "$AGENTS2_CONTAINER_NAME"
    else
        log::info "Showing Agent S2 logs (Ctrl+C to exit)..."
        docker logs -f "$AGENTS2_CONTAINER_NAME"
    fi
}

#######################################
# Execute command in container
# Arguments:
#   $@ - command to execute
# Returns: exit code from command
#######################################
agents2::docker_exec() {
    if ! agents2::is_running; then
        log::error "Agent S2 container is not running"
        return 1
    fi
    
    docker exec -it "$AGENTS2_CONTAINER_NAME" "$@"
}

#######################################
# Get container statistics
# Returns: container stats as string
#######################################
agents2::docker_stats() {
    if agents2::is_running; then
        docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}" "$AGENTS2_CONTAINER_NAME"
    else
        echo "Container not running"
    fi
}

#######################################
# Remove Agent S2 container and network
# Returns: 0 if successful, 1 if failed
#######################################
agents2::docker_cleanup() {
    local success=true
    
    # Stop container if running
    if agents2::is_running; then
        agents2::docker_stop || success=false
    fi
    
    # Remove container
    if agents2::container_exists; then
        log::info "Removing container..."
        docker rm "$AGENTS2_CONTAINER_NAME" >/dev/null 2>&1 || success=false
    fi
    
    # Remove network (only if no other containers are using it)
    if docker network ls --format "{{.Name}}" | grep -q "^${AGENTS2_NETWORK_NAME}$"; then
        if ! docker network inspect "$AGENTS2_NETWORK_NAME" --format '{{len .Containers}}' | grep -q '^0$'; then
            log::info "Network has other containers, skipping removal"
        else
            log::info "Removing network..."
            docker network rm "$AGENTS2_NETWORK_NAME" >/dev/null 2>&1 || success=false
        fi
    fi
    
    # Remove image if requested
    if [[ "${REMOVE_IMAGE:-no}" == "yes" ]] && agents2::image_exists; then
        log::info "Removing Docker image..."
        docker rmi "$AGENTS2_IMAGE_NAME" >/dev/null 2>&1 || success=false
    fi
    
    $success && return 0 || return 1
}