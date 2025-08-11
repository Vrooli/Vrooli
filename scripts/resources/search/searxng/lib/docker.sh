#!/usr/bin/env bash
# SearXNG Docker Operations
# All Docker-related operations for SearXNG containers

SEARXNG_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

#######################################
# Create SearXNG Docker network
#######################################
searxng::create_network() {
    if docker network inspect "$SEARXNG_NETWORK_NAME" >/dev/null 2>&1; then
        log::info "SearXNG network already exists"
        return 0
    fi
    
    log::info "Creating SearXNG Docker network: $SEARXNG_NETWORK_NAME"
    if docker network create "$SEARXNG_NETWORK_NAME" >/dev/null 2>&1; then
        log::success "SearXNG network created successfully"
        return 0
    else
        log::error "Failed to create SearXNG network"
        return 1
    fi
}

#######################################
# Remove SearXNG Docker network
#######################################
searxng::remove_network() {
    if ! docker network inspect "$SEARXNG_NETWORK_NAME" >/dev/null 2>&1; then
        log::info "SearXNG network does not exist"
        return 0
    fi
    
    log::info "Removing SearXNG Docker network: $SEARXNG_NETWORK_NAME"
    if docker network rm "$SEARXNG_NETWORK_NAME" >/dev/null 2>&1; then
        log::success "SearXNG network removed successfully"
        return 0
    else
        log::warn "Failed to remove SearXNG network (may have containers attached)"
        return 1
    fi
}

#######################################
# Pull SearXNG Docker image
#######################################
searxng::pull_image() {
    log::info "Pulling SearXNG Docker image: $SEARXNG_IMAGE"
    
    if docker pull "$SEARXNG_IMAGE"; then
        log::success "SearXNG image pulled successfully"
        return 0
    else
        log::error "Failed to pull SearXNG image"
        return 1
    fi
}

#######################################
# Build SearXNG Docker command for running container
#######################################
searxng::build_docker_command() {
    local docker_cmd="docker run -d"
    
    # Basic container configuration
    docker_cmd+=" --name $SEARXNG_CONTAINER_NAME"
    docker_cmd+=" --hostname searxng"
    docker_cmd+=" --network $SEARXNG_NETWORK_NAME"
    
    # Port mapping
    docker_cmd+=" -p ${SEARXNG_BIND_ADDRESS}:${SEARXNG_PORT}:8080"
    
    # User mapping - SearXNG needs to run as its own user (977:977)
    # We'll handle file permissions by setting proper ownership on the volume
    
    # Volume mounts
    docker_cmd+=" -v ${SEARXNG_DATA_DIR}:/etc/searxng:rw"
    
    # Environment variables
    docker_cmd+=" -e SEARXNG_BASE_URL=${SEARXNG_BASE_URL}"
    docker_cmd+=" -e SEARXNG_SECRET=${SEARXNG_SECRET_KEY}"
    
    # Security configuration
    docker_cmd+=" --cap-drop=ALL"
    docker_cmd+=" --cap-add=CHOWN"
    docker_cmd+=" --cap-add=SETGID" 
    docker_cmd+=" --cap-add=SETUID"
    
    # Restart policy
    docker_cmd+=" --restart unless-stopped"
    
    # Logging configuration
    docker_cmd+=" --log-driver json-file"
    docker_cmd+=" --log-opt max-size=1m"
    docker_cmd+=" --log-opt max-file=3"
    
    # Health check (if not built into image)
    docker_cmd+=" --health-cmd='curl -f http://localhost:8080/stats || exit 1'"
    docker_cmd+=" --health-interval=30s"
    docker_cmd+=" --health-timeout=10s"
    docker_cmd+=" --health-retries=3"
    docker_cmd+=" --health-start-period=60s"
    
    # Image
    docker_cmd+=" $SEARXNG_IMAGE"
    
    echo "$docker_cmd"
}

#######################################
# Start SearXNG container
#######################################
searxng::start_container() {
    if searxng::is_running; then
        searxng::message "warning" "MSG_SEARXNG_ALREADY_RUNNING"
        return 0
    fi
    
    # Ensure network exists
    if ! searxng::create_network; then
        return 1
    fi
    
    # Build and execute Docker command
    local docker_cmd
    docker_cmd=$(searxng::build_docker_command)
    
    log::info "Starting SearXNG container..."
    if eval "$docker_cmd" >/dev/null 2>&1; then
        searxng::message "success" "MSG_SEARXNG_START_SUCCESS"
        
        # Wait for container to become healthy
        if searxng::wait_for_health; then
            searxng::message "info" "MSG_SEARXNG_ACCESS_INFO"
            return 0
        else
            log::warn "SearXNG started but may not be fully healthy yet"
            return 0
        fi
    else
        searxng::message "error" "MSG_SEARXNG_START_FAILED"
        return 1
    fi
}

#######################################
# Stop SearXNG container
#######################################
searxng::stop_container() {
    if ! searxng::is_running; then
        searxng::message "warning" "MSG_SEARXNG_NOT_RUNNING"
        return 0
    fi
    
    log::info "Stopping SearXNG container..."
    if docker stop "$SEARXNG_CONTAINER_NAME" >/dev/null 2>&1; then
        searxng::message "success" "MSG_SEARXNG_STOP_SUCCESS"
        return 0
    else
        searxng::message "error" "MSG_SEARXNG_STOP_FAILED"
        return 1
    fi
}

#######################################
# Restart SearXNG container
#######################################
searxng::restart_container() {
    log::info "Restarting SearXNG container..."
    
    # Stop container if running
    if searxng::is_running; then
        if ! searxng::stop_container; then
            return 1
        fi
        
        # Wait a moment for clean shutdown
        sleep 2
    fi
    
    # Start container
    if searxng::start_container; then
        searxng::message "success" "MSG_SEARXNG_RESTART_SUCCESS"
        return 0
    else
        return 1
    fi
}

#######################################
# Remove SearXNG container
#######################################
searxng::remove_container() {
    # Stop container first if running
    if searxng::is_running; then
        searxng::stop_container
    fi
    
    if searxng::is_installed; then
        log::info "Removing SearXNG container..."
        if docker rm "$SEARXNG_CONTAINER_NAME" >/dev/null 2>&1; then
            log::success "SearXNG container removed"
            return 0
        else
            log::error "Failed to remove SearXNG container"
            return 1
        fi
    fi
    
    return 0
}

#######################################
# Start SearXNG using Docker Compose
#######################################
searxng::compose_up() {
    local compose_file="$SEARXNG_LIB_DIR/../docker/docker-compose.yml"
    local redis_profile=""
    
    if [[ ! -f "$compose_file" ]]; then
        log::error "Docker Compose file not found: $compose_file"
        return 1
    fi
    
    # Enable Redis profile if configured
    if [[ "$SEARXNG_ENABLE_REDIS" == "yes" ]]; then
        redis_profile="--profile redis"
    fi
    
    log::info "Starting SearXNG with Docker Compose..."
    
    # Export environment variables for docker-compose
    export SEARXNG_PORT SEARXNG_BASE_URL SEARXNG_SECRET_KEY SEARXNG_DATA_DIR
    export SEARXNG_CONTAINER_NAME SEARXNG_IMAGE SEARXNG_NETWORK_NAME SEARXNG_BIND_ADDRESS
    
    if docker compose -f "$compose_file" up -d $redis_profile; then
        searxng::message "success" "MSG_SEARXNG_START_SUCCESS"
        
        # Wait for health check
        if searxng::wait_for_health; then
            searxng::message "info" "MSG_SEARXNG_ACCESS_INFO"
            return 0
        else
            log::warn "SearXNG started but may not be fully healthy yet"
            return 0
        fi
    else
        searxng::message "error" "MSG_SEARXNG_START_FAILED"
        return 1
    fi
}

#######################################
# Stop SearXNG using Docker Compose
#######################################
searxng::compose_down() {
    local compose_file="$SEARXNG_LIB_DIR/../docker/docker-compose.yml"
    
    if [[ ! -f "$compose_file" ]]; then
        log::error "Docker Compose file not found: $compose_file"
        return 1
    fi
    
    log::info "Stopping SearXNG with Docker Compose..."
    
    # Export environment variables for docker-compose
    export SEARXNG_NETWORK_NAME
    
    if docker compose -f "$compose_file" down; then
        searxng::message "success" "MSG_SEARXNG_STOP_SUCCESS"
        return 0
    else
        searxng::message "error" "MSG_SEARXNG_STOP_FAILED"
        return 1
    fi
}

#######################################
# Get SearXNG container resource usage
#######################################
searxng::get_resource_usage() {
    if ! searxng::is_running; then
        echo "Container not running"
        return 1
    fi
    
    echo "SearXNG Resource Usage:"
    docker stats --no-stream --format "table {{.CPUPerc}}\t{{.MemUsage}}\t{{.MemPerc}}\t{{.NetIO}}\t{{.BlockIO}}" "$SEARXNG_CONTAINER_NAME"
}

#######################################
# Execute command in SearXNG container
#######################################
searxng::exec_command() {
    local command="$1"
    
    if ! searxng::is_running; then
        log::error "SearXNG container is not running"
        return 1
    fi
    
    docker exec -it "$SEARXNG_CONTAINER_NAME" "$command"
}