#!/usr/bin/env bash
# Speaker Verification Docker Management - Ultra-Simplified
# Uses docker-resource-utils.sh for minimal boilerplate

# Source var.sh to get proper directory variables
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/lib/utils/var.sh"
_SPEAKER_VERIFICATION_DOCKER_DIR="${var_RESOURCES_DIR}/speaker-verification/lib"

# Source shared libraries
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/docker-resource-utils.sh"
# shellcheck disable=SC1091
source "${var_RESOURCES_DIR}/speaker-verification/config/defaults.sh"

# Export configuration
defaults::export_config

# Network name for Speaker Verification
readonly SPEAKER_VERIFICATION_NETWORK_NAME="speaker-verification-network"

#######################################
# Check if GPU is available
# Returns: 0 if available, 1 otherwise
#######################################
speaker_verification::docker::is_gpu_available() {
    system::host_inventory_bool "has_docker_addressable_nvidia_gpu"
}

#######################################
# Show Speaker Verification container logs
#######################################
speaker_verification::docker::show_logs() {
    local lines="${1:-50}" follow="${2:-no}"
    docker_resource::show_logs_with_follow "$SPEAKER_VERIFICATION_CONTAINER_NAME" "$lines" "$follow"
}

#######################################
# Show container resource statistics
#######################################
speaker_verification::docker::show_stats() {
    docker_resource::get_stats "$SPEAKER_VERIFICATION_CONTAINER_NAME"
}

#######################################
# Execute a command inside the container
#######################################
speaker_verification::docker::exec() {
    docker_resource::exec "$SPEAKER_VERIFICATION_CONTAINER_NAME" "$@"
}

#######################################
# Get container information
#######################################
speaker_verification::docker::container_info() {
    if ! docker::container_exists "$SPEAKER_VERIFICATION_CONTAINER_NAME"; then
        log::error "Speaker Verification container does not exist"
        return 1
    fi

    echo "Container Information:"
    docker inspect "$SPEAKER_VERIFICATION_CONTAINER_NAME" --format '
Name: {{.Name}}
Image: {{.Config.Image}}
Status: {{.State.Status}}
Created: {{.Created}}
Ports: {{range $p, $conf := .NetworkSettings.Ports}}{{$p}} -> {{(index $conf 0).HostPort}} {{end}}
Mounts: {{range .Mounts}}{{.Source}} -> {{.Destination}} {{end}}
Environment: {{range .Config.Env}}{{.}} {{end}}'
}

speaker_verification::docker::check_gpu_support() {
    if speaker_verification::docker::is_gpu_available; then
        echo "GPU support available"
    else
        echo "No GPU support"
        return 1
    fi
}

speaker_verification::docker::check_container_health() {
    docker::container_exists "$SPEAKER_VERIFICATION_CONTAINER_NAME" || { echo "Container does not exist"; return 1; }

    local health_status
    health_status=$(docker inspect "$SPEAKER_VERIFICATION_CONTAINER_NAME" --format '{{.State.Health.Status}}')

    if [[ "$health_status" == "healthy" ]]; then
        echo "Container is healthy"
    else
        echo "Container is not healthy: $health_status"
        return 1
    fi
}

#######################################
# Start Speaker Verification using docker-compose
# Arguments: $1 - use GPU compose file (yes/no)
# Returns: 0 if successful, 1 otherwise
#######################################
speaker_verification::compose_up() {
    local use_gpu="${1:-$SPEAKER_VERIFICATION_GPU_ENABLED}"
    local compose_file="${_SPEAKER_VERIFICATION_DOCKER_DIR}/../docker/docker-compose.yml"

    if [[ "$use_gpu" == "yes" ]] && speaker_verification::docker::is_gpu_available; then
        compose_file="${_SPEAKER_VERIFICATION_DOCKER_DIR}/../docker/docker-compose.gpu.yml"
        log::info "Using GPU-enabled compose configuration"
    fi

    docker_resource::compose_up "$compose_file"
}

#######################################
# Stop Speaker Verification using docker-compose
# Arguments: $1 - remove volumes (yes/no)
# Returns: 0 if successful, 1 otherwise
#######################################
speaker_verification::compose_down() {
    local remove_volumes="${1:-no}"
    local compose_file="${_SPEAKER_VERIFICATION_DOCKER_DIR}/../docker/docker-compose.yml"

    # Check for GPU compose if container was started with it
    if docker inspect "$SPEAKER_VERIFICATION_CONTAINER_NAME" 2>/dev/null | grep -q "nvidia"; then
        compose_file="${_SPEAKER_VERIFICATION_DOCKER_DIR}/../docker/docker-compose.gpu.yml"
    fi

    docker_resource::compose_down "$compose_file" "$remove_volumes"
}

# Export functions for subshell availability
export -f speaker_verification::docker::is_gpu_available
export -f speaker_verification::docker::show_logs speaker_verification::docker::show_stats
export -f speaker_verification::docker::exec speaker_verification::docker::container_info
export -f speaker_verification::docker::check_gpu_support
export -f speaker_verification::docker::check_container_health
export -f speaker_verification::compose_up speaker_verification::compose_down
