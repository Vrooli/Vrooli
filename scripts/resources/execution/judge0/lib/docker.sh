#!/usr/bin/env bash
# Judge0 Docker Management Module
# Handles Docker container lifecycle operations

#######################################
# Start Judge0 services
#######################################
judge0::docker::start() {
    if ! judge0::is_installed; then
        log::error "$JUDGE0_MSG_STATUS_NOT_INSTALLED"
        return 1
    fi
    
    if judge0::is_running; then
        log::info "Judge0 is already running"
        return 0
    fi
    
    log::info "$JUDGE0_MSG_START"
    
    local compose_file="${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    
    if docker compose -f "$compose_file" start >/dev/null 2>&1; then
        if judge0::wait_for_health; then
            log::success "Judge0 started successfully"
            return 0
        else
            log::error "Judge0 started but health check failed"
            return 1
        fi
    else
        log::error "Failed to start Judge0"
        return 1
    fi
}

#######################################
# Stop Judge0 services
#######################################
judge0::docker::stop() {
    if ! judge0::is_running; then
        log::info "Judge0 is not running"
        return 0
    fi
    
    log::info "$JUDGE0_MSG_STOP"
    
    local compose_file="${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    
    if docker compose -f "$compose_file" stop >/dev/null 2>&1; then
        log::success "Judge0 stopped successfully"
        return 0
    else
        log::error "Failed to stop Judge0"
        return 1
    fi
}

#######################################
# Restart Judge0 services
#######################################
judge0::docker::restart() {
    log::info "$JUDGE0_MSG_RESTART"
    
    if judge0::docker::stop; then
        sleep 2
        judge0::docker::start
    else
        return 1
    fi
}

#######################################
# Show Judge0 logs
#######################################
judge0::docker::logs() {
    if ! judge0::is_installed; then
        log::error "$JUDGE0_MSG_STATUS_NOT_INSTALLED"
        return 1
    fi
    
    log::info "$JUDGE0_MSG_LOGS"
    
    local compose_file="${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    
    # Show logs from all services
    docker compose -f "$compose_file" logs --tail=100 --follow
}

#######################################
# Uninstall Judge0
# Arguments:
#   $1 - Force uninstall (yes/no)
#######################################
judge0::docker::uninstall() {
    local force="${1:-no}"
    
    if ! judge0::is_installed; then
        log::info "Judge0 is not installed"
        return 0
    fi
    
    if [[ "$force" != "yes" ]]; then
        log::warning "This will remove Judge0 and all its data"
        read -p "Are you sure? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            log::info "Uninstall cancelled"
            return 0
        fi
    fi
    
    log::info "$JUDGE0_MSG_UNINSTALL"
    
    # Stop services
    judge0::docker::stop
    
    local compose_file="${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    
    # Remove containers and volumes
    if [[ -f "$compose_file" ]]; then
        docker compose -f "$compose_file" down -v >/dev/null 2>&1
    fi
    
    # Remove network
    docker::remove_network "$JUDGE0_NETWORK_NAME"
    
    # Remove main volume
    docker volume rm "$JUDGE0_VOLUME_NAME" >/dev/null 2>&1
    
    # Clean up data
    judge0::cleanup_data "$force"
    
    log::success "Judge0 uninstalled successfully"
}

#######################################
# Update Judge0 to latest version
#######################################
judge0::docker::update() {
    log::info "Checking for Judge0 updates..."
    
    # Get current version
    local current_version=$(judge0::get_version)
    
    # Pull latest image
    local latest_image="${JUDGE0_IMAGE}:latest"
    if ! docker pull "$latest_image" >/dev/null 2>&1; then
        log::error "Failed to pull latest Judge0 image"
        return 1
    fi
    
    # Get latest version
    local latest_version=$(docker inspect "$latest_image" --format '{{.Config.Labels.version}}' 2>/dev/null || echo "unknown")
    
    if [[ "$current_version" == "$latest_version" ]]; then
        log::info "Judge0 is already up to date (version: $current_version)"
        return 0
    fi
    
    log::info "Updating Judge0 from $current_version to $latest_version..."
    
    # Stop current services
    judge0::docker::stop
    
    # Update compose file with new version
    local compose_file="${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    sed -i "s/${JUDGE0_IMAGE}:${JUDGE0_VERSION}/${JUDGE0_IMAGE}:${latest_version}/g" "$compose_file"
    
    # Update version in config
    export JUDGE0_VERSION="$latest_version"
    
    # Start updated services
    judge0::docker::start
    
    log::success "Judge0 updated to version $latest_version"
}

#######################################
# Execute command in Judge0 container
# Arguments:
#   $@ - Command to execute
#######################################
judge0::docker::exec() {
    if ! judge0::is_running; then
        log::error "Judge0 is not running"
        return 1
    fi
    
    docker exec -it "$JUDGE0_CONTAINER_NAME" "$@"
}

#######################################
# Show Judge0 container resource usage
#######################################
judge0::docker::stats() {
    if ! judge0::is_running; then
        log::error "Judge0 is not running"
        return 1
    fi
    
    log::info "Judge0 Container Statistics:"
    
    # Server stats
    echo "📊 Server:"
    docker stats "$JUDGE0_CONTAINER_NAME" --no-stream
    
    # Worker stats
    echo
    echo "👷 Workers:"
    for i in $(seq 1 "$JUDGE0_WORKERS_COUNT"); do
        local worker_name="${JUDGE0_WORKERS_NAME}-${i}"
        if docker::is_running "$worker_name"; then
            docker stats "$worker_name" --no-stream
        fi
    done
    
    # Database stats
    echo
    echo "🗄️  Database:"
    docker stats "${JUDGE0_CONTAINER_NAME}-db" --no-stream
    
    # Redis stats
    echo
    echo "💾 Redis:"
    docker stats "${JUDGE0_CONTAINER_NAME}-redis" --no-stream
}

#######################################
# Scale Judge0 workers
# Arguments:
#   $1 - Number of workers
#######################################
judge0::docker::scale_workers() {
    local worker_count="${1:-$JUDGE0_WORKERS_COUNT}"
    
    if ! judge0::is_running; then
        log::error "Judge0 is not running"
        return 1
    fi
    
    log::info "Scaling Judge0 workers to $worker_count..."
    
    local compose_file="${JUDGE0_CONFIG_DIR}/docker-compose.yml"
    
    if docker compose -f "$compose_file" up -d --scale judge0-workers="$worker_count" >/dev/null 2>&1; then
        log::success "Successfully scaled workers to $worker_count"
        export JUDGE0_WORKERS_COUNT="$worker_count"
    else
        log::error "Failed to scale workers"
        return 1
    fi
}

#######################################
# Check if Judge0 container exists
# Returns: 0 if exists, 1 if not
#######################################
judge0::docker::container_exists() {
    docker ps -a --format "{{.Names}}" | grep -q "^${JUDGE0_CONTAINER_NAME}$"
}

#######################################
# Check if Judge0 container is running
# Returns: 0 if running, 1 if not
#######################################
judge0::docker::is_running() {
    docker ps --format "{{.Names}}" | grep -q "^${JUDGE0_CONTAINER_NAME}$"
}

# Export functions for use by other modules
export -f judge0::docker::container_exists
export -f judge0::docker::is_running