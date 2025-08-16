#!/usr/bin/env bash
# SearXNG Docker Operations - Ultra-Simplified
# Uses docker-resource-utils.sh for minimal boilerplate

SEARXNG_LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source shared libraries  
# shellcheck disable=SC1091
source "${SEARXNG_LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${SEARXNG_LIB_DIR}/../../../../lib/service/secrets.sh"
# shellcheck disable=SC1091
source "${SEARXNG_LIB_DIR}/../../../lib/docker-resource-utils.sh"

#######################################
# Pull SearXNG Docker image
#######################################
searxng::pull_image() {
    log::info "Pulling SearXNG Docker image: $SEARXNG_IMAGE"
    docker::pull_image "$SEARXNG_IMAGE"
}

#######################################
# Start SearXNG container using simplified API with enhanced configuration
#######################################
searxng::start_container() {
    if searxng::is_running; then
        searxng::message "warning" "MSG_SEARXNG_ALREADY_RUNNING"
        return 0
    fi
    
    log::info "Starting SearXNG container..."
    
    # Create network first
    docker::create_network "$SEARXNG_NETWORK_NAME"
    
    # Build enhanced docker run command with SearXNG-specific settings
    local cmd=("docker" "run" "-d")
    cmd+=("--name" "$SEARXNG_CONTAINER_NAME")
    cmd+=("--hostname" "searxng")
    cmd+=("--network" "$SEARXNG_NETWORK_NAME")
    cmd+=("--restart" "unless-stopped")
    
    # Port mapping with bind address support
    cmd+=("-p" "${SEARXNG_BIND_ADDRESS}:${SEARXNG_PORT}:8080")
    
    # Volume mounts
    cmd+=("-v" "${SEARXNG_DATA_DIR}:/etc/searxng:rw")
    
    # Environment variables for SearXNG
    cmd+=("-e" "SEARXNG_BASE_URL=${SEARXNG_BASE_URL}")
    cmd+=("-e" "SEARXNG_SECRET=${SEARXNG_SECRET_KEY}")
    
    # Security configuration - SearXNG specific capabilities
    cmd+=("--cap-drop=ALL")
    cmd+=("--cap-add=CHOWN")
    cmd+=("--cap-add=SETGID")
    cmd+=("--cap-add=SETUID")
    
    # Logging configuration
    cmd+=("--log-driver" "json-file")
    cmd+=("--log-opt" "max-size=1m")
    cmd+=("--log-opt" "max-file=3")
    
    # Health check configuration
    cmd+=("--health-cmd" "curl -f http://localhost:8080/stats || exit 1")
    cmd+=("--health-interval" "30s")
    cmd+=("--health-timeout" "10s")
    cmd+=("--health-retries" "3")
    cmd+=("--health-start-period" "60s")
    
    # Image
    cmd+=("$SEARXNG_IMAGE")
    
    # Execute the command
    if "${cmd[@]}" >/dev/null 2>&1; then
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
    if docker::stop_container "$SEARXNG_CONTAINER_NAME"; then
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
    
    if docker::restart_container "$SEARXNG_CONTAINER_NAME"; then
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
    log::info "Removing SearXNG container..."
    docker::remove_container "$SEARXNG_CONTAINER_NAME" "true"
}

#######################################
# Start SearXNG using Docker Compose (preserved as-is)
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
# Stop SearXNG using Docker Compose (preserved as-is)
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

#######################################
# Legacy compatibility functions (simplified)
#######################################
searxng::create_network() {
    docker::create_network "$SEARXNG_NETWORK_NAME"
}

searxng::remove_network() {
    if docker network inspect "$SEARXNG_NETWORK_NAME" >/dev/null 2>&1; then
        docker network rm "$SEARXNG_NETWORK_NAME" >/dev/null 2>&1 || true
    fi
}

# Removed: searxng::build_docker_command - replaced by docker_resource::create_service_with_command