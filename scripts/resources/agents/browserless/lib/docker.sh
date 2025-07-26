#!/usr/bin/env bash
# Browserless Docker Operations
# Container lifecycle management and Docker operations

#######################################
# Create Docker network for Browserless
#######################################
browserless::create_network() {
    if ! docker network ls | grep -q "$BROWSERLESS_NETWORK_NAME"; then
        log::info "${MSG_CREATING_NETWORK}"
        
        if docker network create "$BROWSERLESS_NETWORK_NAME" >/dev/null 2>&1; then
            log::success "${MSG_NETWORK_CREATED}"
            
            # Add rollback action
            resources::add_rollback_action \
                "Remove Docker network" \
                "docker network rm $BROWSERLESS_NETWORK_NAME 2>/dev/null || true" \
                5
        else
            log::warn "${MSG_NETWORK_CREATE_FAILED}"
        fi
    fi
}

#######################################
# Build Docker run command with all parameters
# Arguments:
#   $1 - Max browsers (optional, uses BROWSERLESS_MAX_BROWSERS)
#   $2 - Timeout (optional, uses BROWSERLESS_TIMEOUT)
#   $3 - Headless mode (optional, uses BROWSERLESS_HEADLESS)
# Returns: Outputs the complete Docker command
#######################################
browserless::build_docker_command() {
    local max_browsers="${1:-$BROWSERLESS_MAX_BROWSERS}"
    local timeout="${2:-$BROWSERLESS_TIMEOUT}"
    local headless="${3:-$BROWSERLESS_HEADLESS}"
    
    local docker_cmd="docker run -d"
    docker_cmd+=" --name $BROWSERLESS_CONTAINER_NAME"
    docker_cmd+=" --network $BROWSERLESS_NETWORK_NAME"
    docker_cmd+=" -p ${BROWSERLESS_PORT}:3000"
    docker_cmd+=" -v ${BROWSERLESS_DATA_DIR}:/workspace"
    docker_cmd+=" --restart unless-stopped"
    docker_cmd+=" --shm-size=${BROWSERLESS_DOCKER_SHM_SIZE}"
    
    # Environment variables (using new variable names)
    docker_cmd+=" -e CONCURRENT=${max_browsers}"
    docker_cmd+=" -e TIMEOUT=${timeout}"
    docker_cmd+=" -e ENABLE_DEBUGGER=false"
    docker_cmd+=" -e PREBOOT_CHROME=true"
    
    # Security settings
    docker_cmd+=" --cap-add=${BROWSERLESS_DOCKER_CAPS}"
    docker_cmd+=" --security-opt seccomp=${BROWSERLESS_DOCKER_SECCOMP}"
    
    # Image
    docker_cmd+=" $BROWSERLESS_IMAGE"
    
    echo "$docker_cmd"
}

#######################################
# Start Browserless Docker container
# Arguments:
#   $1 - Max browsers (optional)
#   $2 - Timeout (optional)
#   $3 - Headless mode (optional)
# Returns: 0 if successful, 1 if failed
#######################################
browserless::docker_run() {
    local max_browsers="${1:-$BROWSERLESS_MAX_BROWSERS}"
    local timeout="${2:-$BROWSERLESS_TIMEOUT}"
    local headless="${3:-$BROWSERLESS_HEADLESS}"
    
    log::info "${MSG_STARTING_CONTAINER}"
    
    # Build and execute Docker command
    local docker_cmd
    docker_cmd=$(browserless::build_docker_command "$max_browsers" "$timeout" "$headless")
    
    if eval "$docker_cmd" >/dev/null 2>&1; then
        log::success "${MSG_CONTAINER_STARTED}"
        
        # Add rollback action
        resources::add_rollback_action \
            "Stop and remove Browserless container" \
            "docker stop $BROWSERLESS_CONTAINER_NAME 2>/dev/null; docker rm $BROWSERLESS_CONTAINER_NAME 2>/dev/null || true" \
            25
        
        return 0
    else
        log::error "${MSG_START_CONTAINER_FAILED}"
        return 1
    fi
}

#######################################
# Stop Browserless Docker container
# Returns: 0 if successful, 1 if failed
#######################################
browserless::docker_stop() {
    if ! browserless::is_running; then
        log::info "${MSG_NOT_RUNNING}"
        return 0
    fi
    
    log::info "${MSG_STOPPING}"
    
    if docker stop "$BROWSERLESS_CONTAINER_NAME" >/dev/null 2>&1; then
        log::success "${MSG_STOP_SUCCESS}"
        return 0
    else
        log::error "${MSG_STOP_FAILED}"
        return 1
    fi
}

#######################################
# Start existing Browserless Docker container
# Returns: 0 if successful, 1 if failed
#######################################
browserless::docker_start() {
    if browserless::is_running && [[ "$FORCE" != "yes" ]]; then
        log::info "${MSG_ALREADY_RUNNING}"
        return 0
    fi
    
    log::info "${MSG_STARTING}"
    
    # Check if container exists
    if ! browserless::container_exists; then
        log::error "${MSG_CONTAINER_NOT_EXISTS}. Run install first."
        return 1
    fi
    
    # Start Browserless container
    if docker start "$BROWSERLESS_CONTAINER_NAME" >/dev/null 2>&1; then
        log::success "${MSG_START_SUCCESS}"
        
        # Wait for service to be ready
        if resources::wait_for_service "Browserless" "$BROWSERLESS_PORT" 30; then
            log::success "✅ Browserless is running on port $BROWSERLESS_PORT"
            log::info "Access Browserless at: $BROWSERLESS_BASE_URL"
        else
            log::warn "Browserless started but may not be fully ready yet"
        fi
        return 0
    else
        log::error "${MSG_START_FAILED}"
        return 1
    fi
}

#######################################
# Restart Browserless Docker container
# Returns: 0 if successful, 1 if failed
#######################################
browserless::docker_restart() {
    log::info "${MSG_RESTARTING}"
    browserless::docker_stop
    sleep 2
    browserless::docker_start
}

#######################################
# Remove Browserless Docker container and network
# Arguments:
#   $1 - Whether to remove data directory (yes/no, optional)
# Returns: 0 if successful, 1 if failed
#######################################
browserless::docker_remove() {
    local remove_data="${1:-no}"
    
    # Stop and remove Browserless container
    if browserless::container_exists; then
        log::info "${MSG_REMOVING_CONTAINER}"
        docker stop "$BROWSERLESS_CONTAINER_NAME" 2>/dev/null || true
        docker rm "$BROWSERLESS_CONTAINER_NAME" 2>/dev/null || true
        log::success "${MSG_CONTAINER_REMOVED}"
    fi
    
    # Remove Docker network
    if docker network ls | grep -q "$BROWSERLESS_NETWORK_NAME"; then
        log::info "${MSG_REMOVING_NETWORK}"
        docker network rm "$BROWSERLESS_NETWORK_NAME" 2>/dev/null || true
    fi
    
    # Remove data directory if requested
    if [[ "$remove_data" == "yes" && -d "$BROWSERLESS_DATA_DIR" ]]; then
        rm -rf "$BROWSERLESS_DATA_DIR" 2>/dev/null || true
        log::info "Data directory removed"
    fi
}

#######################################
# Show Browserless Docker logs
# Returns: 0 if successful, 1 if failed
#######################################
browserless::docker_logs() {
    if ! browserless::container_exists; then
        log::error "${MSG_CONTAINER_NOT_EXISTS}"
        return 1
    fi
    
    log::info "${MSG_SHOWING_LOGS}"
    docker logs -f "$BROWSERLESS_CONTAINER_NAME"
}

#######################################
# Pull latest Browserless Docker image
# Returns: 0 if successful, 1 if failed
#######################################
browserless::docker_pull() {
    log::info "${MSG_PULLING_IMAGE}"
    
    if docker pull "$BROWSERLESS_IMAGE" >/dev/null 2>&1; then
        log::success "Latest Browserless image pulled"
        return 0
    else
        log::error "Failed to pull Browserless image"
        return 1
    fi
}