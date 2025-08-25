#!/bin/bash
# LiteLLM Docker management functionality

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
LITELLM_DOCKER_DIR="${APP_ROOT}/resources/litellm/lib"

# Source dependencies
source "${LITELLM_DOCKER_DIR}/core.sh"

# Start LiteLLM proxy server
litellm::start() {
    local verbose="${1:-false}"
    local wait="${2:-true}"
    
    [[ "$verbose" == "true" ]] && log::info "Starting LiteLLM proxy server"
    
    # Initialize configuration
    litellm::init "$verbose"
    
    # Load provider keys
    litellm::load_provider_keys "$verbose"
    
    # Check if already running
    if litellm::is_running; then
        [[ "$verbose" == "true" ]] && log::info "LiteLLM is already running"
        return 0
    fi
    
    # Create Docker network if it doesn't exist
    if ! docker network ls --format '{{.Name}}' | grep -q "^${LITELLM_NETWORK}$"; then
        [[ "$verbose" == "true" ]] && log::info "Creating Docker network: $LITELLM_NETWORK"
        docker network create "$LITELLM_NETWORK" >/dev/null 2>&1 || {
            log::error "Failed to create Docker network"
            return 1
        }
    fi
    
    # Remove existing container if it exists
    if docker ps -a --format '{{.Names}}' | grep -q "^${LITELLM_CONTAINER_NAME}$"; then
        [[ "$verbose" == "true" ]] && log::info "Removing existing container"
        docker rm -f "$LITELLM_CONTAINER_NAME" >/dev/null 2>&1
    fi
    
    # Build Docker run command
    local docker_cmd=(
        "docker" "run" "-d"
        "--name" "$LITELLM_CONTAINER_NAME"
        "--hostname" "$LITELLM_HOSTNAME"
        "--network" "$LITELLM_NETWORK"
        "-p" "${LITELLM_PORT}:${LITELLM_INTERNAL_PORT}"
        "-v" "${LITELLM_CONFIG_FILE}:/app/config.yaml:ro"
        "-v" "${LITELLM_ENV_FILE}:/app/.env:ro"
        "-v" "${LITELLM_DATA_DIR}:/app/data"
        "-v" "${LITELLM_LOG_DIR}:/app/logs"
        "--restart" "unless-stopped"
        "$LITELLM_IMAGE"
        "--config" "/app/config.yaml"
        "--port" "$LITELLM_INTERNAL_PORT"
        "--detailed_debug"
    )
    
    # Add environment variables
    if [[ -f "$LITELLM_ENV_FILE" ]]; then
        docker_cmd+=("--env-file" "/app/.env")
    fi
    
    [[ "$verbose" == "true" ]] && log::info "Starting container: ${LITELLM_CONTAINER_NAME}"
    
    # Start the container
    if ! "${docker_cmd[@]}" >/dev/null 2>&1; then
        log::error "Failed to start LiteLLM container"
        return 1
    fi
    
    # Wait for startup if requested
    if [[ "$wait" == "true" ]]; then
        [[ "$verbose" == "true" ]] && log::info "Waiting for LiteLLM to start..."
        
        local count=0
        local max_attempts=$((LITELLM_STARTUP_TIMEOUT / 2))
        
        while [[ $count -lt $max_attempts ]]; do
            if litellm::test_connection 5 false; then
                [[ "$verbose" == "true" ]] && log::info "LiteLLM started successfully"
                return 0
            fi
            
            sleep 2
            count=$((count + 1))
            
            [[ "$verbose" == "true" ]] && echo -n "."
        done
        
        [[ "$verbose" == "true" ]] && echo
        log::error "LiteLLM failed to start within ${LITELLM_STARTUP_TIMEOUT} seconds"
        
        # Show container logs for debugging
        if [[ "$verbose" == "true" ]]; then
            log::info "Container logs:"
            docker logs "$LITELLM_CONTAINER_NAME" 2>&1 | tail -20
        fi
        
        return 1
    fi
    
    [[ "$verbose" == "true" ]] && log::info "LiteLLM container started"
    return 0
}

# Stop LiteLLM proxy server
litellm::stop() {
    local verbose="${1:-false}"
    local force="${2:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Stopping LiteLLM proxy server"
    
    # Check if running
    if ! litellm::is_running; then
        [[ "$verbose" == "true" ]] && log::info "LiteLLM is not running"
        return 0
    fi
    
    # Stop the container
    if [[ "$force" == "true" ]]; then
        [[ "$verbose" == "true" ]] && log::info "Force stopping container"
        docker kill "$LITELLM_CONTAINER_NAME" >/dev/null 2>&1
    else
        [[ "$verbose" == "true" ]] && log::info "Gracefully stopping container"
        docker stop "$LITELLM_CONTAINER_NAME" >/dev/null 2>&1
    fi
    
    # Remove the container
    docker rm "$LITELLM_CONTAINER_NAME" >/dev/null 2>&1
    
    [[ "$verbose" == "true" ]] && log::info "LiteLLM stopped"
    return 0
}

# Restart LiteLLM proxy server
litellm::restart() {
    local verbose="${1:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Restarting LiteLLM proxy server"
    
    litellm::stop "$verbose"
    sleep 2
    litellm::start "$verbose"
}

# Get container logs
litellm::logs() {
    local lines="${1:-50}"
    local follow="${2:-false}"
    
    if ! litellm::is_running; then
        echo "LiteLLM container is not running"
        return 1
    fi
    
    if [[ "$follow" == "true" ]]; then
        docker logs -f --tail "$lines" "$LITELLM_CONTAINER_NAME"
    else
        docker logs --tail "$lines" "$LITELLM_CONTAINER_NAME"
    fi
}

# Execute command in container
litellm::exec() {
    local cmd="$1"
    
    if ! litellm::is_running; then
        echo "LiteLLM container is not running"
        return 1
    fi
    
    docker exec -it "$LITELLM_CONTAINER_NAME" $cmd
}

# Get container resource usage
litellm::stats() {
    if ! litellm::is_running; then
        echo "LiteLLM container is not running"
        return 1
    fi
    
    docker stats --no-stream "$LITELLM_CONTAINER_NAME"
}

# Inspect container configuration
litellm::inspect() {
    if ! docker ps -a --format '{{.Names}}' | grep -q "^${LITELLM_CONTAINER_NAME}$"; then
        echo "LiteLLM container does not exist"
        return 1
    fi
    
    docker inspect "$LITELLM_CONTAINER_NAME"
}

# Update LiteLLM image
litellm::update() {
    local verbose="${1:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Updating LiteLLM image"
    
    # Pull latest image
    docker pull "$LITELLM_IMAGE" || {
        log::error "Failed to pull latest LiteLLM image"
        return 1
    }
    
    # Restart with new image
    if litellm::is_running; then
        litellm::restart "$verbose"
    fi
    
    [[ "$verbose" == "true" ]] && log::info "LiteLLM updated successfully"
    return 0
}

# Backup configuration and data
litellm::backup() {
    local backup_dir="${1:-${var_ROOT_DIR}/backups/litellm-$(date +%Y%m%d-%H%M%S)}"
    local verbose="${2:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Creating backup at: $backup_dir"
    
    mkdir -p "$backup_dir"
    
    # Backup configuration
    if [[ -d "$LITELLM_CONFIG_DIR" ]]; then
        cp -r "$LITELLM_CONFIG_DIR" "$backup_dir/config"
    fi
    
    # Backup data
    if [[ -d "$LITELLM_DATA_DIR" ]]; then
        cp -r "$LITELLM_DATA_DIR" "$backup_dir/data"
    fi
    
    # Backup logs
    if [[ -d "$LITELLM_LOG_DIR" ]]; then
        cp -r "$LITELLM_LOG_DIR" "$backup_dir/logs"
    fi
    
    # Create metadata
    cat > "$backup_dir/metadata.json" <<EOF
{
    "timestamp": "$(date -Iseconds)",
    "image": "$LITELLM_IMAGE",
    "container_name": "$LITELLM_CONTAINER_NAME",
    "config_dir": "$LITELLM_CONFIG_DIR",
    "data_dir": "$LITELLM_DATA_DIR",
    "log_dir": "$LITELLM_LOG_DIR"
}
EOF
    
    [[ "$verbose" == "true" ]] && log::info "Backup created successfully"
    echo "$backup_dir"
    return 0
}