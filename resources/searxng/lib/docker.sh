#!/usr/bin/env bash
# SearXNG Docker Operations - Ultra-Simplified
# Uses docker-resource-utils.sh for minimal boilerplate

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SEARXNG_LIB_DIR="${APP_ROOT}/resources/searxng/lib"

# Source shared libraries  
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/service/secrets.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/docker-resource-utils.sh"

#######################################
# Start SearXNG container using shared utilities
#######################################
searxng::start_container() {
    if searxng::is_running; then
        log::warn "SearXNG is already running"
        return 0
    fi
    
    # Check if container exists but is stopped, try to start it first
    if docker ps -a --format "{{.Names}}" | grep -q "^${SEARXNG_CONTAINER_NAME}$"; then
        log::info "Found existing SearXNG container, attempting to start it..."
        if docker start "$SEARXNG_CONTAINER_NAME" >/dev/null 2>&1; then
            if searxng::wait_for_health; then
                log::success "Existing SearXNG container started successfully"
                return 0
            fi
        fi
        # If starting the existing container failed, remove it and create a new one
        log::info "Removing existing container to create a fresh one..."
        docker rm -f "$SEARXNG_CONTAINER_NAME" >/dev/null 2>&1 || true
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
    
    # Health check - using wget with 127.0.0.1 since localhost doesn't resolve in container
    local health_cmd="wget --spider -q http://127.0.0.1:8080/ || exit 1"
    
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
        log::info "Redis caching enabled - starting Redis container"
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

#######################################
# Enable Redis caching for SearXNG
#######################################
searxng::enable_redis() {
    log::info "Enabling Redis caching for SearXNG..."
    
    # Check if Redis container exists
    if docker ps -a --format "{{.Names}}" | grep -q "searxng-redis"; then
        log::info "Redis container already exists"
        # Start it if not running
        if ! docker ps --format "{{.Names}}" | grep -q "searxng-redis"; then
            if docker start searxng-redis; then
                log::success "Redis container started"
            else
                log::error "Failed to start Redis container"
                return 1
            fi
        else
            log::info "Redis container is already running"
        fi
    else
        # Create and start Redis using docker-compose
        local compose_file="$SEARXNG_LIB_DIR/../docker/docker-compose.yml"
        
        log::info "Creating and starting Redis container..."
        if docker-compose -f "$compose_file" --profile redis up -d redis; then
            log::success "Redis container created and started"
        else
            log::error "Failed to create Redis container"
            return 1
        fi
    fi
    
    # Wait for Redis to be healthy
    log::info "Waiting for Redis to become healthy..."
    local attempts=0
    local max_attempts=30
    
    while [[ $attempts -lt $max_attempts ]]; do
        if docker exec searxng-redis redis-cli ping >/dev/null 2>&1; then
            log::success "✅ Redis is healthy and ready"
            break
        fi
        ((attempts++))
        sleep 1
    done
    
    if [[ $attempts -eq $max_attempts ]]; then
        log::warn "Redis may not be fully ready yet, but container is running"
    fi
    
    # Note: Redis is now enabled
    log::success "✅ Redis is healthy and ready"
    log::info "Redis caching enabled. Restart SearXNG to use caching:"
    log::info "  resource-searxng manage restart"
    
    return 0
}

#######################################
# Disable Redis caching for SearXNG
#######################################
searxng::disable_redis() {
    log::info "Disabling Redis caching for SearXNG..."
    
    # Stop Redis container if running
    if docker ps --format "{{.Names}}" | grep -q "searxng-redis"; then
        if docker stop searxng-redis; then
            log::success "Redis container stopped"
        else
            log::warn "Failed to stop Redis container"
        fi
    fi
    
    # Note: Redis is now disabled
    log::info "Redis caching disabled. Restart SearXNG to apply changes:"
    log::info "  resource-searxng manage restart"
    
    return 0
}

#######################################
# Check Redis status and show caching information
#######################################
searxng::redis_status() {
    log::info "Redis Caching Status:"
    
    if [[ "$SEARXNG_ENABLE_REDIS" == "yes" ]]; then
        echo "  Configuration: Enabled"
        
        # Check if Redis container exists and is running
        if docker ps --format "{{.Names}}" | grep -q "searxng-redis"; then
            echo "  Container: Running"
            
            # Test Redis connection
            if docker exec searxng-redis redis-cli ping >/dev/null 2>&1; then
                echo "  Health: Healthy"
                
                # Get Redis info
                local redis_info
                redis_info=$(docker exec searxng-redis redis-cli info keyspace 2>/dev/null | grep "^db" || echo "No cached data")
                echo "  Cache Status: $redis_info"
            else
                echo "  Health: Unhealthy"
            fi
        else
            echo "  Container: Not running"
        fi
    else
        echo "  Configuration: Disabled"
        echo "  Performance: Using in-memory caching only"
        echo "  Enable with: resource-searxng content execute --name enable-redis"
    fi
}