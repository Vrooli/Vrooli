#!/usr/bin/env bash
# Whisper Docker Management - Ultra-Simplified
# Uses docker-resource-utils.sh for minimal boilerplate

# Source var.sh to get proper directory variables
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
_WHISPER_DOCKER_DIR="${APP_ROOT}/resources/whisper/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

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
    docker::pull_image "$image"
}

#######################################
# Start Whisper container with GPU support
# Arguments: $1 - model (optional), $2 - GPU enabled (optional)
# Returns: 0 if successful, 1 otherwise
#######################################
whisper::docker::start_container() {
    local model="${1:-$WHISPER_DEFAULT_MODEL}"
    local gpu_enabled="${2:-$WHISPER_GPU_ENABLED}"
    local image
    
    # Ensure directories exist
    whisper::create_directories || return 1
    
    image=$(WHISPER_GPU_ENABLED="$gpu_enabled" whisper::docker::get_docker_image)
    
    log::info "Starting Whisper container..."
    
    # Pull image
    docker::pull_image "$image"
    
    # Prepare environment variables
    local env_vars=(
        "ASR_MODEL=$model"
        "ASR_ENGINE=openai_whisper"
    )
    
    # Prepare Docker options
    local docker_opts=()
    
    # Add GPU support if enabled and available
    if [[ "$gpu_enabled" == "yes" ]] && whisper::docker::is_gpu_available; then
        docker_opts+=("--gpus" "all")
        env_vars+=("NVIDIA_VISIBLE_DEVICES=all")
        log::debug "GPU support enabled"
    fi
    
    # Volumes
    local volumes="${WHISPER_MODELS_DIR}:/app/models ${WHISPER_UPLOADS_DIR}:/app/uploads"
    
    # Use advanced creation
    docker_resource::create_service_advanced \
        "$WHISPER_CONTAINER_NAME" \
        "$image" \
        "${WHISPER_PORT}:9000" \
        "$WHISPER_NETWORK_NAME" \
        "$volumes" \
        "env_vars" \
        "docker_opts" \
        "" \
        ""
    
    if [[ $? -eq 0 ]]; then
        sleep "${WHISPER_INITIALIZATION_WAIT:-30}"
        return 0
    else
        log::error "Failed to start Whisper container"
        return 1
    fi
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

whisper::docker::show_logs() {
    local lines="${1:-50}" follow="${2:-no}"
    docker_resource::show_logs_with_follow "$WHISPER_CONTAINER_NAME" "$lines" "$follow"
}

whisper::docker::show_stats() {
    docker_resource::get_stats "$WHISPER_CONTAINER_NAME"
}

whisper::docker::exec() {
    docker_resource::exec "$WHISPER_CONTAINER_NAME" "$@"
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



whisper::docker::check_gpu_support() {
    if whisper::docker::is_gpu_available; then
        echo "GPU support available"
    else
        echo "No GPU support"
        return 1
    fi
}

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

#######################################
# Start Whisper using docker-compose
# Arguments: $1 - use GPU compose file (yes/no)
# Returns: 0 if successful, 1 otherwise
#######################################
whisper::compose_up() {
    local use_gpu="${1:-$WHISPER_GPU_ENABLED}"
    local compose_file="${_WHISPER_DOCKER_DIR}/../docker/docker-compose.yml"
    
    if [[ "$use_gpu" == "yes" ]] && whisper::docker::is_gpu_available; then
        compose_file="${_WHISPER_DOCKER_DIR}/../docker/docker-compose.gpu.yml"
        log::info "Using GPU-enabled compose configuration"
    fi
    
    docker_resource::compose_up "$compose_file"
}

#######################################
# Stop Whisper using docker-compose
# Arguments: $1 - remove volumes (yes/no)
# Returns: 0 if successful, 1 otherwise
#######################################
whisper::compose_down() {
    local remove_volumes="${1:-no}"
    local compose_file="${_WHISPER_DOCKER_DIR}/../docker/docker-compose.yml"
    
    # Check for GPU compose if container was started with it
    if docker inspect "$WHISPER_CONTAINER_NAME" 2>/dev/null | grep -q "nvidia"; then
        compose_file="${_WHISPER_DOCKER_DIR}/../docker/docker-compose.gpu.yml"
    fi
    
    docker_resource::compose_down "$compose_file" "$remove_volumes"
}

# Export functions for subshell availability
export -f whisper::docker::pull_image whisper::docker::start_container
export -f whisper::docker::restart_container whisper::docker::show_logs whisper::docker::show_stats
export -f whisper::docker::exec whisper::docker::container_info
export -f whisper::docker::check_gpu_support
export -f whisper::docker::check_container_health whisper::docker::get_docker_image whisper::docker::is_gpu_available
export -f whisper::compose_up whisper::compose_down