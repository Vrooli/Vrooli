#!/usr/bin/env bash
# K6 Docker Functions

# Start K6 container (for cloud/dashboard features)
k6::docker::start() {
    log::info "Starting K6 container..."
    
    # Initialize directories
    k6::core::init
    
    # Check if already running
    if k6::core::is_running; then
        log::warn "K6 container already running"
        return 0
    fi
    
    # Pull latest image
    log::info "Pulling K6 Docker image..."
    docker pull "$K6_IMAGE"
    
    # Start container with correct user and entrypoint
    docker run -d \
        --name "$K6_CONTAINER_NAME" \
        --restart unless-stopped \
        --user "$(id -u):$(id -g)" \
        -v "$K6_SCRIPTS_DIR:/scripts:ro" \
        -v "$K6_RESULTS_DIR:/results:rw" \
        -v "$K6_DATA_DIR:/data:ro" \
        -p "${K6_PORT}:6565" \
        --entrypoint sleep \
        "$K6_IMAGE" \
        infinity
    
    # Wait for container to be ready
    sleep 2
    
    if k6::core::is_running; then
        log::success "K6 container started"
    else
        log::error "Failed to start K6 container"
        docker logs "$K6_CONTAINER_NAME" --tail 20
        return 1
    fi
}

# Stop K6 container
k6::docker::stop() {
    log::info "Stopping K6 container..."
    
    if ! k6::core::is_running; then
        log::warn "K6 container not running"
        return 0
    fi
    
    docker stop "$K6_CONTAINER_NAME"
    docker rm "$K6_CONTAINER_NAME"
    
    log::success "K6 container stopped"
}

# Restart K6 container
k6::docker::restart() {
    k6::docker::stop
    k6::docker::start
}

# Show K6 logs
k6::docker::logs() {
    local lines="${1:-50}"
    
    if k6::core::is_running; then
        docker logs "$K6_CONTAINER_NAME" --tail "$lines"
    else
        log::warn "K6 container not running"
    fi
}