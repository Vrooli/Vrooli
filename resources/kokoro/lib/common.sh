#!/usr/bin/env bash
# Kokoro Common Utility Functions
# Shared utilities used across all modules

# Resolve repo paths from this library location
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"

# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/lib/utils/var.sh"

# Source defaults first (messages reference config variables like KOKORO_PORT)
source "${var_RESOURCES_DIR}/kokoro/config/defaults.sh"
defaults::export_config

# Source messages
source "${var_RESOURCES_DIR}/kokoro/config/messages.sh"
messages::export_messages

#######################################
# Check if Docker is installed and configured
# Returns: 0 if installed, 1 otherwise
#######################################
common::check_docker() {
    if ! system::is_command "docker"; then
        log::error "${MSG_DOCKER_NOT_FOUND}"
        log::info "${MSG_DOCKER_INSTALL_HINT}"
        return 1
    fi

    # Check if Docker daemon is running
    if ! docker info >/dev/null 2>&1; then
        log::error "${MSG_DOCKER_NOT_RUNNING}"
        log::info "${MSG_DOCKER_START_HINT}"
        return 1
    fi

    # Check if user has permissions
    if ! docker ps >/dev/null 2>&1; then
        log::error "${MSG_DOCKER_NO_PERMISSIONS}"
        log::info "${MSG_DOCKER_PERMISSIONS_HINT}"
        log::info "${MSG_DOCKER_LOGOUT_HINT}"
        return 1
    fi

    return 0
}

#######################################
# Check if Kokoro container exists
# Returns: 0 if exists, 1 otherwise
#######################################
common::container_exists() {
    docker ps -a --format '{{.Names}}' | grep -q "^${KOKORO_CONTAINER_NAME}$"
}

#######################################
# Check if Kokoro is running
# Returns: 0 if running, 1 otherwise
#######################################
common::is_running() {
    docker ps --format '{{.Names}}' | grep -q "^${KOKORO_CONTAINER_NAME}$"
}

#######################################
# Check if port is available
# Arguments:
#   $1 - port number
# Returns: 0 if available, 1 if in use
#######################################
common::is_port_available() {
    local port="$1"

    if system::is_port_in_use "$port"; then
        log::error "${MSG_PORT_IN_USE}"
        return 1
    fi

    return 0
}

#######################################
# Create Kokoro data directories
# Returns: 0 if successful, 1 otherwise
#######################################
kokoro::create_directories() {
    log::info "${MSG_CREATING_DIRS}"

    if ! mkdir -p "$KOKORO_DATA_DIR" "$KOKORO_VOICES_DIR"; then
        log::error "${MSG_CREATE_DIRS_FAILED}"
        return 1
    fi

    # Kokoro runs in-container as "appuser", so the bind-mounted voice cache
    # must be writable by a non-host UID/GID as well.
    if ! chmod 0777 "$KOKORO_DATA_DIR" "$KOKORO_VOICES_DIR"; then
        log::error "Failed to update Kokoro data directory permissions"
        return 1
    fi

    log::debug "${MSG_DIRECTORIES_CREATED}"
    return 0
}

#######################################
# Wait for Kokoro to be healthy
# Returns: 0 if healthy, 1 on timeout
#######################################
kokoro::wait_for_health() {
    log::info "${MSG_WAITING_STARTUP}"

    local max_attempts=$((KOKORO_STARTUP_MAX_WAIT / KOKORO_STARTUP_WAIT_INTERVAL))
    local attempt=1

    while [[ $attempt -le $max_attempts ]]; do
        if kokoro::is_healthy; then
            log::debug "${MSG_HEALTHY}"
            return 0
        fi

        log::debug "Health check attempt $attempt/$max_attempts failed, waiting..."
        sleep "$KOKORO_STARTUP_WAIT_INTERVAL"
        ((attempt++))
    done

    log::error "${MSG_STARTUP_TIMEOUT}"
    return 1
}

#######################################
# Check if Kokoro API is healthy
# Returns: 0 if healthy, 1 otherwise
#######################################
kokoro::is_healthy() {
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" \
        "$KOKORO_BASE_URL/v1/audio/voices" \
        --max-time "$KOKORO_API_TIMEOUT" 2>/dev/null)

    [[ "$response" == "200" ]]
}

#######################################
# Check if GPU is available
# Returns: 0 if available, 1 otherwise
#######################################
kokoro::is_gpu_available() {
    system::host_inventory_bool "has_docker_addressable_nvidia_gpu"
}

#######################################
# Get appropriate Docker image based on GPU availability
# Outputs: Docker image name
#######################################
kokoro::get_docker_image() {
    if [[ "$KOKORO_GPU_ENABLED" == "yes" ]] && kokoro::is_gpu_available; then
        echo "$KOKORO_IMAGE"
    else
        if [[ "$KOKORO_GPU_ENABLED" == "yes" ]]; then
            log::warn "${MSG_GPU_NOT_AVAILABLE}"
        fi
        echo "$KOKORO_CPU_IMAGE"
    fi
}

#######################################
# Remove Kokoro container and cleanup
# Returns: 0 if successful, 1 otherwise
#######################################
kokoro::cleanup() {
    local success=true

    if common::container_exists; then
        log::info "${MSG_REMOVING_CONTAINER}"
        if ! docker rm -f "$KOKORO_CONTAINER_NAME" >/dev/null 2>&1; then
            log::warn "Failed to remove container"
            success=false
        fi
    fi

    $success
}

# Export functions for subshell availability
export -f common::check_docker
export -f common::container_exists
export -f common::is_running
export -f common::is_port_available
export -f kokoro::create_directories
export -f kokoro::wait_for_health
export -f kokoro::is_healthy
export -f kokoro::is_gpu_available
export -f kokoro::get_docker_image
export -f kokoro::cleanup
