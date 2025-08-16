#!/usr/bin/env bash
# Whisper Docker Management - Ultra-Simplified
# Uses docker-resource-utils.sh for minimal boilerplate

# Source var.sh to get proper directory variables
_WHISPER_DOCKER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck disable=SC1091
source "${_WHISPER_DOCKER_DIR}/../../../../lib/utils/var.sh"

# Source shared libraries
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"
# shellcheck disable=SC1091
source "${_WHISPER_DOCKER_DIR}/../config/defaults.sh"

# Export configuration
defaults::export_config

# Network name for Whisper
readonly WHISPER_NETWORK_NAME="whisper-network"

#######################################
# Get appropriate Docker image based on GPU availability
# Outputs: Docker image name
#######################################
whisper::docker::get_docker_image() {
    if [[ "$WHISPER_GPU_ENABLED" == "yes" ]] && whisper::docker::is_gpu_available; then
        echo "$WHISPER_IMAGE"
    else
        [[ "$WHISPER_GPU_ENABLED" == "yes" ]] && log::warn "GPU not available, falling back to CPU image"
        echo "$WHISPER_CPU_IMAGE"
    fi
}

#######################################
# Check if GPU is available
# Returns: 0 if available, 1 otherwise
#######################################
whisper::docker::is_gpu_available() {
    system::is_command "nvidia-smi" && nvidia-smi >/dev/null 2>&1 && docker info | grep -q nvidia
}

#######################################
# Pull Whisper Docker image
# Arguments: $1 - GPU enabled (yes/no)
# Returns: 0 if successful, 1 otherwise
#######################################
whisper::docker::pull_image() {
    local gpu_enabled="${1:-$WHISPER_GPU_ENABLED}"
    local image
    
    image=$(WHISPER_GPU_ENABLED="$gpu_enabled" whisper::docker::get_docker_image)
    log::info "Pulling Whisper Docker image..."
    log::debug "Pulling image: $image"
    
    if ! docker pull "$image"; then
        log::error "Failed to pull Whisper image: $image"
        return 1
    fi
    
    return 0
}

#######################################
# Start Whisper container with GPU support
# Arguments: $1 - model (optional), $2 - GPU enabled (optional)
# Returns: 0 if successful, 1 otherwise
#######################################
whisper::docker::start_container() {
    local model="${1:-$WHISPER_DEFAULT_MODEL}"
    local gpu_enabled="${2:-$WHISPER_GPU_ENABLED}"
    local image volumes
    
    # Ensure directories exist
    whisper::create_directories || return 1
    
    image=$(WHISPER_GPU_ENABLED="$gpu_enabled" whisper::docker::get_docker_image)
    volumes="${WHISPER_MODELS_DIR}:/app/models ${WHISPER_UPLOADS_DIR}:/app/uploads"
    
    log::info "Starting Whisper container..."
    log::debug "Starting container with model: $model"
    
    # Use simplified service creation with enhanced functionality
    local cmd=(docker run -d --name "$WHISPER_CONTAINER_NAME" --restart=unless-stopped)
    cmd+=("-p" "${WHISPER_PORT}:9000")
    
    # Add volumes
    for volume in $volumes; do
        cmd+=("-v" "$volume")
    done
    
    # Add environment variables
    cmd+=("-e" "ASR_MODEL=$model" "-e" "ASR_ENGINE=openai_whisper")
    
    # Add GPU support if enabled and available
    if [[ "$gpu_enabled" == "yes" ]] && whisper::docker::is_gpu_available; then
        cmd+=("--gpus" "all" "-e" "NVIDIA_VISIBLE_DEVICES=all")
        log::debug "GPU support enabled"
    fi
    
    # Create network and add to command
    docker::create_network "$WHISPER_NETWORK_NAME"
    cmd+=("--network" "$WHISPER_NETWORK_NAME" "$image")
    
    # Execute and wait for initialization
    if "${cmd[@]}" >/dev/null 2>&1; then
        log::debug "Whisper container started successfully"
        sleep "${WHISPER_INITIALIZATION_WAIT:-30}"
        return 0
    else
        log::error "Failed to start Whisper container"
        return 1
    fi
}

#######################################
# Stop Whisper container
#######################################
whisper::docker::stop_container() {
    log::info "Stopping Whisper container..."
    docker stop "$WHISPER_CONTAINER_NAME" >/dev/null 2>&1
}

#######################################
# Restart Whisper container
# Arguments: $1 - model (optional), $2 - GPU enabled (optional)
#######################################
whisper::docker::restart_container() {
    local model="${1:-$WHISPER_DEFAULT_MODEL}"
    local gpu_enabled="${2:-$WHISPER_GPU_ENABLED}"
    
    log::info "Restarting Whisper container..."
    
    # Stop and remove if running
    docker::is_running "$WHISPER_CONTAINER_NAME" && docker stop "$WHISPER_CONTAINER_NAME" >/dev/null 2>&1 && sleep ${WHISPER_STOP_WAIT:-2}
    docker::container_exists "$WHISPER_CONTAINER_NAME" && docker rm -f "$WHISPER_CONTAINER_NAME" >/dev/null 2>&1
    
    # Start with new parameters
    whisper::docker::start_container "$model" "$gpu_enabled"
}

#######################################
# Get Whisper container logs
# Arguments: $1 - lines (optional), $2 - follow (optional)
#######################################
whisper::docker::show_logs() {
    local lines="${1:-50}" follow="${2:-no}"
    
    if ! docker::container_exists "$WHISPER_CONTAINER_NAME"; then
        log::error "Whisper container does not exist"
        return 1
    fi
    
    if [[ "$follow" == "yes" ]]; then
        log::info "Following Whisper logs..."
        docker logs -f --tail "$lines" "$WHISPER_CONTAINER_NAME"
    else
        docker::get_logs "$WHISPER_CONTAINER_NAME" "$lines"
    fi
}

#######################################
# Get Whisper container stats
#######################################
whisper::docker::show_stats() {
    docker::is_running "$WHISPER_CONTAINER_NAME" || { log::error "Whisper is not running"; return 1; }
    docker stats --no-stream "$WHISPER_CONTAINER_NAME"
}

#######################################
# Execute command in Whisper container
# Arguments: $@ - command to execute
#######################################
whisper::docker::exec() {
    docker::is_running "$WHISPER_CONTAINER_NAME" || { log::error "Whisper is not running"; return 1; }
    docker exec "$WHISPER_CONTAINER_NAME" "$@"
}

#######################################
# Get container information
#######################################
whisper::docker::container_info() {
    if ! docker::container_exists "$WHISPER_CONTAINER_NAME"; then
        log::error "Whisper container does not exist"
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

#######################################
# Remove container
#######################################
whisper::docker::remove_container() {
    docker::container_exists "$WHISPER_CONTAINER_NAME" || { echo "Container does not exist"; return 0; }
    log::info "Removing Whisper container..."
    docker rm -f "$WHISPER_CONTAINER_NAME" >/dev/null 2>&1
}

#######################################
# Get container logs (alias for show_logs)
#######################################
whisper::docker::get_logs() {
    whisper::docker::show_logs 50 "${1:-no}"
}

#######################################
# Inspect container
#######################################
whisper::docker::inspect_container() {
    docker::container_exists "$WHISPER_CONTAINER_NAME" || { echo "Container does not exist"; return 1; }
    docker inspect "$WHISPER_CONTAINER_NAME"
}

#######################################
# Check GPU support
#######################################
whisper::docker::check_gpu_support() {
    if whisper::docker::is_gpu_available; then
        echo "GPU support available"
    else
        echo "No GPU support"
        return 1
    fi
}

#######################################
# Check container health
#######################################
whisper::docker::check_container_health() {
    docker::container_exists "$WHISPER_CONTAINER_NAME" || { echo "Container does not exist"; return 1; }
    
    local health_status
    health_status=$(docker inspect "$WHISPER_CONTAINER_NAME" --format '{{.State.Health.Status}}')
    
    if [[ "$health_status" == "healthy" ]]; then
        echo "Container is healthy"
    else
        echo "Container is not healthy: $health_status"
        return 1
    fi
}

# Export functions for subshell availability
export -f whisper::docker::pull_image whisper::docker::start_container whisper::docker::stop_container
export -f whisper::docker::restart_container whisper::docker::show_logs whisper::docker::show_stats
export -f whisper::docker::exec whisper::docker::container_info whisper::docker::remove_container
export -f whisper::docker::get_logs whisper::docker::inspect_container whisper::docker::check_gpu_support
export -f whisper::docker::check_container_health whisper::docker::get_docker_image whisper::docker::is_gpu_available