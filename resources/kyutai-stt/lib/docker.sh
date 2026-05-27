#!/usr/bin/env bash
# Kyutai STT Docker Management
# Uses docker-resource-utils.sh for minimal boilerplate. Unlike prebuilt-image
# resources, Kyutai STT builds a local image from docker/Dockerfile via compose.

# Source var.sh to get proper directory variables
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/lib/utils/var.sh"
_KYUTAI_STT_DOCKER_DIR="${var_RESOURCES_DIR}/kyutai-stt/lib"

# Source shared libraries
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"
# shellcheck disable=SC1091
source "${var_RESOURCES_DIR}/kyutai-stt/config/defaults.sh"

# Export configuration
defaults::export_config

# Network name for Kyutai STT (declared in the compose files; kept here for
# parity with the resource template and consumers that source this lib).
# shellcheck disable=SC2034
readonly KYUTAI_STT_NETWORK_NAME="kyutai-stt-network"

#######################################
# Check if GPU is available
# Returns: 0 if available, 1 otherwise
#######################################
kyutai_stt::docker::is_gpu_available() {
    system::is_command "nvidia-smi" && nvidia-smi >/dev/null 2>&1 && docker info | grep -q nvidia
}

#######################################
# Resolve the compose file to use based on GPU availability
# Outputs: absolute path to the compose file
#######################################
kyutai_stt::docker::compose_file() {
    local use_gpu="${1:-$KYUTAI_STT_GPU_ENABLED}"
    if [[ "$use_gpu" == "yes" ]] && kyutai_stt::docker::is_gpu_available; then
        echo "${_KYUTAI_STT_DOCKER_DIR}/../docker/docker-compose.gpu.yml"
    else
        echo "${_KYUTAI_STT_DOCKER_DIR}/../docker/docker-compose.yml"
    fi
}

#######################################
# Start Kyutai STT via docker-compose (builds the image on first run).
# Arguments: $1 - GPU enabled (optional)
# Returns: 0 if successful, 1 otherwise
#######################################
kyutai_stt::docker::start_container() {
    local gpu_enabled="${1:-$KYUTAI_STT_GPU_ENABLED}"

    # Ensure directories exist (HF cache bind mount target)
    kyutai_stt::create_directories || return 1

    if [[ "$gpu_enabled" != "yes" ]] || ! kyutai_stt::docker::is_gpu_available; then
        log::warn "${MSG_GPU_NOT_AVAILABLE}"
    fi

    local compose_file
    compose_file="$(kyutai_stt::docker::compose_file "$gpu_enabled")"

    log::info "${MSG_STARTING_CONTAINER}"
    if docker_resource::compose_up "$compose_file"; then
        sleep "${KYUTAI_STT_INITIALIZATION_WAIT:-30}"
        return 0
    fi

    log::error "${MSG_START_CONTAINER_FAILED}"
    return 1
}

#######################################
# Restart Kyutai STT container
# Arguments: $1 - GPU enabled (optional)
#######################################
kyutai_stt::docker::restart_container() {
    local gpu_enabled="${1:-$KYUTAI_STT_GPU_ENABLED}"

    log::info "${MSG_RESTARTING}"

    local compose_file
    compose_file="$(kyutai_stt::docker::compose_file "$gpu_enabled")"
    docker_resource::compose_down "$compose_file" "false" || true

    kyutai_stt::docker::start_container "$gpu_enabled"
}

kyutai_stt::docker::show_logs() {
    local lines="${1:-50}" follow="${2:-no}"
    docker_resource::show_logs_with_follow "$KYUTAI_STT_CONTAINER_NAME" "$lines" "$follow"
}

kyutai_stt::docker::show_stats() {
    docker_resource::get_stats "$KYUTAI_STT_CONTAINER_NAME"
}

kyutai_stt::docker::exec() {
    docker_resource::exec "$KYUTAI_STT_CONTAINER_NAME" "$@"
}

#######################################
# Get container information
#######################################
kyutai_stt::docker::container_info() {
    if ! docker::container_exists "$KYUTAI_STT_CONTAINER_NAME"; then
        log::error "${MSG_CONTAINER_NOT_EXISTS}"
        return 1
    fi

    echo "Container Information:"
    docker inspect "$KYUTAI_STT_CONTAINER_NAME" --format '
Name: {{.Name}}
Image: {{.Config.Image}}
Status: {{.State.Status}}
Created: {{.Created}}
Ports: {{range $p, $conf := .NetworkSettings.Ports}}{{$p}} -> {{(index $conf 0).HostPort}} {{end}}
Mounts: {{range .Mounts}}{{.Source}} -> {{.Destination}} {{end}}
Environment: {{range .Config.Env}}{{.}} {{end}}'
}

kyutai_stt::docker::check_gpu_support() {
    if kyutai_stt::docker::is_gpu_available; then
        echo "GPU support available"
    else
        echo "No GPU support"
        return 1
    fi
}

kyutai_stt::docker::check_container_health() {
    docker::container_exists "$KYUTAI_STT_CONTAINER_NAME" || { echo "Container does not exist"; return 1; }

    local health_status
    health_status=$(docker inspect "$KYUTAI_STT_CONTAINER_NAME" --format '{{.State.Health.Status}}')

    if [[ "$health_status" == "healthy" ]]; then
        echo "Container is healthy"
    else
        echo "Container is not healthy: $health_status"
        return 1
    fi
}

#######################################
# Start Kyutai STT using docker-compose
# Arguments: $1 - use GPU compose file (yes/no)
# Returns: 0 if successful, 1 otherwise
#######################################
kyutai_stt::compose_up() {
    local use_gpu="${1:-$KYUTAI_STT_GPU_ENABLED}"
    local compose_file
    compose_file="$(kyutai_stt::docker::compose_file "$use_gpu")"

    if [[ "$compose_file" == *docker-compose.gpu.yml ]]; then
        log::info "Using GPU-enabled compose configuration"
    fi

    docker_resource::compose_up "$compose_file"
}

#######################################
# Stop Kyutai STT using docker-compose
# Arguments: $1 - remove volumes (yes/no)
# Returns: 0 if successful, 1 otherwise
#######################################
kyutai_stt::compose_down() {
    local remove_volumes="${1:-no}"
    local compose_file="${_KYUTAI_STT_DOCKER_DIR}/../docker/docker-compose.yml"

    # Use the GPU compose if the container was started with the nvidia runtime.
    if docker inspect "$KYUTAI_STT_CONTAINER_NAME" 2>/dev/null | grep -q "nvidia"; then
        compose_file="${_KYUTAI_STT_DOCKER_DIR}/../docker/docker-compose.gpu.yml"
    fi

    docker_resource::compose_down "$compose_file" "$remove_volumes"
}

# Export functions for subshell availability
export -f kyutai_stt::docker::is_gpu_available kyutai_stt::docker::compose_file
export -f kyutai_stt::docker::start_container kyutai_stt::docker::restart_container
export -f kyutai_stt::docker::show_logs kyutai_stt::docker::show_stats
export -f kyutai_stt::docker::exec kyutai_stt::docker::container_info
export -f kyutai_stt::docker::check_gpu_support kyutai_stt::docker::check_container_health
export -f kyutai_stt::compose_up kyutai_stt::compose_down
