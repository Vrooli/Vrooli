#!/usr/bin/env bash
# Whisper Docker Management Functions
# Docker-specific operations for Whisper

#######################################
# Pull Whisper Docker image
# Arguments:
#   $1 - GPU enabled (yes/no)
# Returns: 0 if successful, 1 otherwise
#######################################
docker::pull_image() {
    local gpu_enabled="${1:-$WHISPER_GPU_ENABLED}"
    local image
    
    image=$(WHISPER_GPU_ENABLED="$gpu_enabled" docker::get_docker_image)
    
    log::info "${MSG_PULLING_IMAGE}"
    log::debug "Pulling image: $image"
    
    if ! docker pull "$image"; then
        log::error "Failed to pull Docker image: $image"
        return 1
    fi
    
    return 0
}

#######################################
# Start Whisper container
# Arguments:
#   $1 - model size (optional, defaults to WHISPER_DEFAULT_MODEL)
#   $2 - GPU enabled (optional, defaults to WHISPER_GPU_ENABLED)
# Returns: 0 if successful, 1 otherwise
#######################################
docker::start_container() {
    local model="${1:-$WHISPER_DEFAULT_MODEL}"
    local gpu_enabled="${2:-$WHISPER_GPU_ENABLED}"
    local image
    
    image=$(WHISPER_GPU_ENABLED="$gpu_enabled" docker::get_docker_image)
    
    log::info "${MSG_STARTING_CONTAINER}"
    log::debug "Starting container with model: $model"
    
    # Build Docker run arguments
    local docker_args=(
        "run"
        "-d"
        "--name" "$WHISPER_CONTAINER_NAME"
        "-p" "${WHISPER_PORT}:9000"
        "-v" "${WHISPER_MODELS_DIR}:/app/models"
        "-v" "${WHISPER_UPLOADS_DIR}:/app/uploads"
        "-e" "ASR_MODEL=$model"
        "-e" "ASR_ENGINE=openai_whisper"
        "--restart=unless-stopped"
    )
    
    # Add GPU support if enabled and available
    if [[ "$gpu_enabled" == "yes" ]] && docker::is_gpu_available; then
        docker_args+=(
            "--gpus" "all"
            "-e" "NVIDIA_VISIBLE_DEVICES=all"
        )
        log::debug "GPU support enabled"
    fi
    
    # Add the image
    docker_args+=("$image")
    
    # Execute docker run
    if ! docker "${docker_args[@]}" >/dev/null 2>&1; then
        log::error "${MSG_START_CONTAINER_FAILED}"
        return 1
    fi
    
    log::debug "${MSG_CONTAINER_STARTED}"
    return 0
}

#######################################
# Stop Whisper container
# Returns: 0 if successful, 1 otherwise
#######################################
docker::stop_container() {
    log::info "${MSG_STOPPING}"
    
    if ! docker stop "$WHISPER_CONTAINER_NAME" >/dev/null 2>&1; then
        log::error "${MSG_STOP_FAILED}"
        return 1
    fi
    
    return 0
}

#######################################
# Restart Whisper container
# Arguments:
#   $1 - model size (optional)
#   $2 - GPU enabled (optional)
# Returns: 0 if successful, 1 otherwise
#######################################
docker::restart_container() {
    local model="${1:-$WHISPER_DEFAULT_MODEL}"
    local gpu_enabled="${2:-$WHISPER_GPU_ENABLED}"
    
    log::info "${MSG_RESTARTING}"
    
    # Stop if running
    if docker::is_running; then
        if ! docker::stop_container; then
            return 1
        fi
        
        # Wait a moment for graceful shutdown
        sleep 2
    fi
    
    # Remove container if it exists
    if docker::container_exists; then
        if ! docker rm "$WHISPER_CONTAINER_NAME" >/dev/null 2>&1; then
            log::warn "Failed to remove existing container"
        fi
    fi
    
    # Start with new parameters
    docker::start_container "$model" "$gpu_enabled"
}

#######################################
# Get Whisper container logs
# Arguments:
#   $1 - number of lines (optional, defaults to 50)
#   $2 - follow logs (optional, yes/no, defaults to no)
# Returns: 0 if successful, 1 otherwise
#######################################
docker::show_logs() {
    local lines="${1:-50}"
    local follow="${2:-no}"
    
    if ! docker::container_exists; then
        log::error "${MSG_CONTAINER_NOT_EXISTS}"
        return 1
    fi
    
    local docker_args=("logs")
    
    if [[ "$follow" == "yes" ]]; then
        log::info "${MSG_SHOWING_LOGS}"
        docker_args+=("-f")
    fi
    
    docker_args+=("--tail" "$lines" "$WHISPER_CONTAINER_NAME")
    
    docker "${docker_args[@]}"
}

#######################################
# Get Whisper container stats
# Returns: 0 if successful, 1 otherwise
#######################################
docker::show_stats() {
    if ! docker::is_running; then
        log::error "${MSG_NOT_RUNNING}"
        return 1
    fi
    
    docker stats --no-stream "$WHISPER_CONTAINER_NAME"
}

#######################################
# Execute command in Whisper container
# Arguments:
#   $@ - command to execute
# Returns: exit code of the command
#######################################
docker::exec() {
    if ! docker::is_running; then
        log::error "${MSG_NOT_RUNNING}"
        return 1
    fi
    
    docker exec "$WHISPER_CONTAINER_NAME" "$@"
}

#######################################
# Get container information
# Returns: 0 if successful, 1 otherwise
#######################################
docker::container_info() {
    if ! docker::container_exists; then
        log::error "${MSG_CONTAINER_NOT_EXISTS}"
        return 1
    fi
    
    echo "Container Information:"
    docker inspect "$WHISPER_CONTAINER_NAME" --format '
Name: {{.Name}}
Image: {{.Config.Image}}
Status: {{.State.Status}}
Created: {{.Created}}
Ports: {{range $p, $conf := .NetworkSettings.Ports}}{{$p}} -> {{(index $conf 0).HostPort}} {{end}}
Mounts: {{range .Mounts}}{{.Source}} -> {{.Destination}} {{end}}
Environment: {{range .Config.Env}}{{.}} {{end}}'
}

# GPU availability check is provided by common.sh

#######################################
# Remove container
#######################################
docker::remove_container() {
    if ! docker::container_exists; then
        echo "Container does not exist"
        return 0
    fi
    
    log::info "Removing Whisper container..."
    docker rm -f "$WHISPER_CONTAINER_NAME"
}

#######################################
# Get container logs (alias for show_logs)
#######################################
docker::get_logs() {
    local follow="${1:-no}"
    if [[ "$follow" == "yes" ]]; then
        docker logs -f "$WHISPER_CONTAINER_NAME"
    else
        docker logs "$WHISPER_CONTAINER_NAME"
    fi
}

#######################################
# Inspect container
#######################################
docker::inspect_container() {
    if ! docker::container_exists; then
        echo "Container does not exist"
        return 1
    fi
    
    docker inspect "$WHISPER_CONTAINER_NAME"
}

#######################################
# Check GPU support
#######################################
docker::check_gpu_support() {
    if docker::is_gpu_available; then
        echo "GPU support available"
        return 0
    else
        echo "No GPU support"
        return 1
    fi
}

# Docker image selection is provided by common.sh

#######################################
# Check container health
#######################################
docker::check_container_health() {
    if ! docker::container_exists; then
        echo "Container does not exist"
        return 1
    fi
    
    local health_status
    health_status=$(docker inspect "$WHISPER_CONTAINER_NAME" --format '{{.State.Health.Status}}')
    
    if [[ "$health_status" == "healthy" ]]; then
        echo "Container is healthy"
        return 0
    else
        echo "Container is not healthy: $health_status"
        return 1
    fi
}

#######################################
# Setup container volumes
#######################################
docker::setup_volumes() {
    echo "Setting up Whisper volumes"
    # Volume setup logic here
    return 0
}

#######################################
# Setup container networking
#######################################
docker::setup_network() {
    echo "Setting up Whisper networking"
    # Network setup logic here
    return 0
}

#######################################
# Setup container environment
#######################################
docker::setup_environment() {
    echo "Setting up Whisper environment"
    # Environment setup logic here
    return 0
}

#######################################
# Set resource limits
#######################################
docker::set_resource_limits() {
    echo "Setting Whisper resource limits"
    # Resource limits logic here
    return 0
}

#######################################
# Cleanup container and resources
#######################################
docker::cleanup_container() {
    echo "Cleaning up Whisper container"
    docker::remove_container
    return 0
}

# Export functions for subshell availability
export -f docker::pull_image
export -f docker::start_container
export -f docker::stop_container
export -f docker::restart_container
export -f docker::show_logs
export -f docker::show_stats
export -f docker::exec
export -f docker::container_info
export -f docker::remove_container
export -f docker::get_logs
export -f docker::inspect_container
export -f docker::check_gpu_support
export -f docker::check_container_health
export -f docker::setup_volumes
export -f docker::setup_network
export -f docker::setup_environment
export -f docker::set_resource_limits
export -f docker::cleanup_container