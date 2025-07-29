#!/usr/bin/env bash
# Whisper Docker Management Functions
# Docker-specific operations for Whisper

#######################################
# Pull Whisper Docker image
# Arguments:
#   $1 - GPU enabled (yes/no)
# Returns: 0 if successful, 1 otherwise
#######################################
whisper::pull_image() {
    local gpu_enabled="${1:-$WHISPER_GPU_ENABLED}"
    local image
    
    image=$(WHISPER_GPU_ENABLED="$gpu_enabled" whisper::get_docker_image)
    
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
whisper::start_container() {
    local model="${1:-$WHISPER_DEFAULT_MODEL}"
    local gpu_enabled="${2:-$WHISPER_GPU_ENABLED}"
    local image
    
    image=$(WHISPER_GPU_ENABLED="$gpu_enabled" whisper::get_docker_image)
    
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
    if [[ "$gpu_enabled" == "yes" ]] && whisper::is_gpu_available; then
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
whisper::stop_container() {
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
whisper::restart_container() {
    local model="${1:-$WHISPER_DEFAULT_MODEL}"
    local gpu_enabled="${2:-$WHISPER_GPU_ENABLED}"
    
    log::info "${MSG_RESTARTING}"
    
    # Stop if running
    if whisper::is_running; then
        if ! whisper::stop_container; then
            return 1
        fi
        
        # Wait a moment for graceful shutdown
        sleep 2
    fi
    
    # Remove container if it exists
    if whisper::container_exists; then
        if ! docker rm "$WHISPER_CONTAINER_NAME" >/dev/null 2>&1; then
            log::warn "Failed to remove existing container"
        fi
    fi
    
    # Start with new parameters
    whisper::start_container "$model" "$gpu_enabled"
}

#######################################
# Get Whisper container logs
# Arguments:
#   $1 - number of lines (optional, defaults to 50)
#   $2 - follow logs (optional, yes/no, defaults to no)
# Returns: 0 if successful, 1 otherwise
#######################################
whisper::show_logs() {
    local lines="${1:-50}"
    local follow="${2:-no}"
    
    if ! whisper::container_exists; then
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
whisper::show_stats() {
    if ! whisper::is_running; then
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
whisper::exec() {
    if ! whisper::is_running; then
        log::error "${MSG_NOT_RUNNING}"
        return 1
    fi
    
    docker exec "$WHISPER_CONTAINER_NAME" "$@"
}

#######################################
# Get container information
# Returns: 0 if successful, 1 otherwise
#######################################
whisper::container_info() {
    if ! whisper::container_exists; then
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