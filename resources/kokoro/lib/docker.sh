#!/usr/bin/env bash
# Kokoro Docker Management - Ultra-Simplified
# Uses docker-resource-utils.sh for minimal boilerplate

# Source var.sh to get proper directory variables
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
_KOKORO_DOCKER_DIR="${var_RESOURCES_DIR}/kokoro/lib"

# Source shared libraries
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"
# shellcheck disable=SC1091
source "${var_RESOURCES_DIR}/kokoro/config/defaults.sh"

# Export configuration
defaults::export_config

# Network name for Kokoro
readonly KOKORO_NETWORK_NAME="kokoro-network"

#######################################
# Get appropriate Docker image based on GPU availability
# Outputs: Docker image name
#######################################
kokoro::docker::get_docker_image() {
    if [[ "$KOKORO_GPU_ENABLED" == "yes" ]] && kokoro::docker::is_gpu_available; then
        echo "$KOKORO_IMAGE"
    else
        [[ "$KOKORO_GPU_ENABLED" == "yes" ]] && log::warn "GPU not available, falling back to CPU image"
        echo "$KOKORO_CPU_IMAGE"
    fi
}

#######################################
# Check if GPU is available
# Returns: 0 if available, 1 otherwise
#######################################
kokoro::docker::is_gpu_available() {
    system::is_command "nvidia-smi" && nvidia-smi >/dev/null 2>&1 && docker info | grep -q nvidia
}

#######################################
# Pull Kokoro Docker image
# Arguments: $1 - GPU enabled (yes/no)
# Returns: 0 if successful, 1 otherwise
#######################################
kokoro::docker::pull_image() {
    local gpu_enabled="${1:-$KOKORO_GPU_ENABLED}"
    local image

    if [[ "$gpu_enabled" == "yes" ]] && kokoro::docker::is_gpu_available; then
        image="$KOKORO_IMAGE"
    else
        image="$KOKORO_CPU_IMAGE"
    fi
    docker::pull_image "$image"
}

#######################################
# Start Kokoro container with GPU support
# Arguments: $1 - GPU enabled (optional)
# Returns: 0 if successful, 1 otherwise
#######################################
kokoro::docker::start_container() {
    local gpu_enabled="${1:-$KOKORO_GPU_ENABLED}"
    local image

    # Ensure directories exist
    kokoro::create_directories || return 1

    if [[ "$gpu_enabled" == "yes" ]] && kokoro::docker::is_gpu_available; then
        image="$KOKORO_IMAGE"
    else
        image="$KOKORO_CPU_IMAGE"
    fi

    log::info "Starting Kokoro container..."

    # Pull image
    docker::pull_image "$image"

    # Prepare environment variables
    local env_vars=()

    # Prepare Docker options
    local docker_opts=()

    # Add GPU support if enabled and available
    if [[ "$gpu_enabled" == "yes" ]] && kokoro::docker::is_gpu_available; then
        docker_opts+=("--gpus" "all")
        env_vars+=("NVIDIA_VISIBLE_DEVICES=all")
        log::debug "GPU support enabled"
    fi

    # Only mount a host voice directory when custom voices exist locally.
    # An empty bind mount hides the image's bundled default voices.
    local volumes=""
    if find "${KOKORO_VOICES_DIR}" -type f \( -name '*.pt' -o -name '*.onnx' -o -name '*.bin' \) -print -quit 2>/dev/null | grep -q .; then
        volumes="${KOKORO_VOICES_DIR}:/app/api/src/voices"
        log::debug "Using host-provided Kokoro voices from ${KOKORO_VOICES_DIR}"
    else
        log::info "No custom Kokoro voices found; using bundled in-image voices"
    fi

    # Use advanced creation
    docker_resource::create_service_advanced \
        "$KOKORO_CONTAINER_NAME" \
        "$image" \
        "${KOKORO_PORT}:8880" \
        "$KOKORO_NETWORK_NAME" \
        "$volumes" \
        "env_vars" \
        "docker_opts" \
        "" \
        ""

    if [[ $? -eq 0 ]]; then
        sleep "${KOKORO_INITIALIZATION_WAIT:-30}"
        return 0
    else
        log::error "Failed to start Kokoro container"
        return 1
    fi
}

#######################################
# Restart Kokoro container
# Arguments: $1 - GPU enabled (optional)
#######################################
kokoro::docker::restart_container() {
    local gpu_enabled="${1:-$KOKORO_GPU_ENABLED}"

    log::info "Restarting Kokoro container..."

    # Stop and remove if running
    docker::is_running "$KOKORO_CONTAINER_NAME" && docker stop "$KOKORO_CONTAINER_NAME" >/dev/null 2>&1 && sleep ${KOKORO_STOP_WAIT:-2}
    docker::container_exists "$KOKORO_CONTAINER_NAME" && docker rm -f "$KOKORO_CONTAINER_NAME" >/dev/null 2>&1

    # Start with new parameters
    kokoro::docker::start_container "$gpu_enabled"
}

kokoro::docker::show_logs() {
    local lines="${1:-50}" follow="${2:-no}"
    docker_resource::show_logs_with_follow "$KOKORO_CONTAINER_NAME" "$lines" "$follow"
}

kokoro::docker::show_stats() {
    docker_resource::get_stats "$KOKORO_CONTAINER_NAME"
}

kokoro::docker::exec() {
    docker_resource::exec "$KOKORO_CONTAINER_NAME" "$@"
}

#######################################
# Get container information
#######################################
kokoro::docker::container_info() {
    if ! docker::container_exists "$KOKORO_CONTAINER_NAME"; then
        log::error "Kokoro container does not exist"
        return 1
    fi

    echo "Container Information:"
    docker inspect "$KOKORO_CONTAINER_NAME" --format '
Name: {{.Name}}
Image: {{.Config.Image}}
Status: {{.State.Status}}
Created: {{.Created}}
Ports: {{range $p, $conf := .NetworkSettings.Ports}}{{$p}} -> {{(index $conf 0).HostPort}} {{end}}
Mounts: {{range .Mounts}}{{.Source}} -> {{.Destination}} {{end}}
Environment: {{range .Config.Env}}{{.}} {{end}}'
}

kokoro::docker::check_gpu_support() {
    if kokoro::docker::is_gpu_available; then
        echo "GPU support available"
    else
        echo "No GPU support"
        return 1
    fi
}

kokoro::docker::check_container_health() {
    docker::container_exists "$KOKORO_CONTAINER_NAME" || { echo "Container does not exist"; return 1; }

    local health_status
    health_status=$(docker inspect "$KOKORO_CONTAINER_NAME" --format '{{.State.Health.Status}}')

    if [[ "$health_status" == "healthy" ]]; then
        echo "Container is healthy"
    else
        echo "Container is not healthy: $health_status"
        return 1
    fi
}

#######################################
# Start Kokoro using docker-compose
# Arguments: $1 - use GPU compose file (yes/no)
# Returns: 0 if successful, 1 otherwise
#######################################
kokoro::compose_up() {
    local use_gpu="${1:-$KOKORO_GPU_ENABLED}"
    local compose_file="${_KOKORO_DOCKER_DIR}/../docker/docker-compose.yml"

    if [[ "$use_gpu" == "yes" ]] && kokoro::docker::is_gpu_available; then
        compose_file="${_KOKORO_DOCKER_DIR}/../docker/docker-compose.gpu.yml"
        log::info "Using GPU-enabled compose configuration"
    fi

    docker_resource::compose_up "$compose_file"
}

#######################################
# Stop Kokoro using docker-compose
# Arguments: $1 - remove volumes (yes/no)
# Returns: 0 if successful, 1 otherwise
#######################################
kokoro::compose_down() {
    local remove_volumes="${1:-no}"
    local compose_file="${_KOKORO_DOCKER_DIR}/../docker/docker-compose.yml"

    # Check for GPU compose if container was started with it
    if docker inspect "$KOKORO_CONTAINER_NAME" 2>/dev/null | grep -q "nvidia"; then
        compose_file="${_KOKORO_DOCKER_DIR}/../docker/docker-compose.gpu.yml"
    fi

    docker_resource::compose_down "$compose_file" "$remove_volumes"
}

# Export functions for subshell availability
export -f kokoro::docker::pull_image kokoro::docker::start_container
export -f kokoro::docker::restart_container kokoro::docker::show_logs kokoro::docker::show_stats
export -f kokoro::docker::exec kokoro::docker::container_info
export -f kokoro::docker::check_gpu_support
export -f kokoro::docker::check_container_health kokoro::docker::get_docker_image kokoro::docker::is_gpu_available
export -f kokoro::compose_up kokoro::compose_down
