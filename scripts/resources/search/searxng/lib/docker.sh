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
# Start SearXNG container using shared utilities
#######################################
searxng::start_container() {
    if searxng::is_running; then
        log::warn "SearXNG is already running"
        return 0
    fi
    
    log::info "Starting SearXNG container..."
    
    # Pull image if needed
    docker::pull_image "$SEARXNG_IMAGE"
    
    # Prepare environment variables
    local env_vars=(
        "SEARXNG_BASE_URL=${SEARXNG_BASE_URL}"
        "SEARXNG_SECRET=${SEARXNG_SECRET_KEY}"
    )
    
    # Docker options for security and logging
    local docker_opts=(
        "--hostname" "searxng"
        "--cap-drop=ALL"
        "--cap-add=CHOWN"
        "--cap-add=SETGID"
        "--cap-add=SETUID"
        "--log-driver" "json-file"
        "--log-opt" "max-size=1m"
        "--log-opt" "max-file=3"
        "--health-start-period" "60s"
    )
    
    # Port mapping with bind address
    local port_mappings="${SEARXNG_BIND_ADDRESS}:${SEARXNG_PORT}:8080"
    
    # Volumes
    local volumes="${SEARXNG_DATA_DIR}:/etc/searxng:rw"
    
    # Health check
    local health_cmd="curl -f http://localhost:8080/stats || exit 1"
    
    # Use advanced creation with custom health intervals
    DOCKER_HEALTH_INTERVAL="30s" \
    DOCKER_HEALTH_TIMEOUT="10s" \
    DOCKER_HEALTH_RETRIES="3" \
    docker_resource::create_service_advanced \
        "$SEARXNG_CONTAINER_NAME" \
        "$SEARXNG_IMAGE" \
        "$port_mappings" \
        "$SEARXNG_NETWORK_NAME" \
        "$volumes" \
        "env_vars" \
        "docker_opts" \
        "$health_cmd" \
        ""
    
    if [[ $? -eq 0 ]]; then
        log::success "SearXNG container started successfully"
        
        # Wait for container to become healthy
        if searxng::wait_for_health; then
            log::info "SearXNG is accessible at http://localhost:${SEARXNG_PORT}"
            return 0
        else
            log::warn "SearXNG started but may not be fully healthy yet"
            return 0
        fi
    else
        log::error "Failed to start SearXNG container"
        return 1
    fi
}

searxng::stop_container() {
    if ! searxng::is_running; then
        log::warn "SearXNG is not running"
        return 0
    fi
    
    log::info "Stopping SearXNG container..."
    if docker::stop_container "$SEARXNG_CONTAINER_NAME"; then
        log::success "SearXNG stopped successfully"
        return 0
    else
        log::error "Failed to stop SearXNG"
        return 1
    fi
}

searxng::restart_container() {
    log::info "Restarting SearXNG container..."
    
    if docker::restart_container "$SEARXNG_CONTAINER_NAME"; then
        log::success "SearXNG restarted successfully"
        return 0
    else
        return 1
    fi
}

searxng::remove_container() {
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
        log::success "SearXNG started successfully"
        
        # Wait for health check
        if searxng::wait_for_health; then
            log::info "SearXNG is accessible at http://localhost:${SEARXNG_PORT}"
            return 0
        else
            log::warn "SearXNG started but may not be fully healthy yet"
            return 0
        fi
    else
        log::error "Failed to start SearXNG container"
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
        log::success "SearXNG stopped successfully"
        return 0
    else
        log::error "Failed to stop SearXNG"
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
        # Remove network only if empty
        docker::cleanup_network_if_empty "$SEARXNG_NETWORK_NAME"
    fi
}

# Removed: searxng::build_docker_command - replaced by docker_resource::create_service_with_command