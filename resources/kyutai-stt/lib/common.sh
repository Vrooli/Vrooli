#!/usr/bin/env bash
# Kyutai STT Common Utility Functions
# Shared utilities used across all modules

# Resolve repo paths from this library location
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"

# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/lib/utils/var.sh"

# Source defaults first (messages reference config variables like KYUTAI_STT_PORT)
source "${var_RESOURCES_DIR}/kyutai-stt/config/defaults.sh"
defaults::export_config

# Source messages
source "${var_RESOURCES_DIR}/kyutai-stt/config/messages.sh"
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
# Check if Kyutai STT container exists
# Returns: 0 if exists, 1 otherwise
#######################################
common::container_exists() {
    docker ps -a --format '{{.Names}}' | grep -q "^${KYUTAI_STT_CONTAINER_NAME}$"
}

#######################################
# Check if Kyutai STT is running
# Returns: 0 if running, 1 otherwise
#######################################
common::is_running() {
    docker ps --format '{{.Names}}' | grep -q "^${KYUTAI_STT_CONTAINER_NAME}$"
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
# Create Kyutai STT data directories
# Returns: 0 if successful, 1 otherwise
#######################################
kyutai_stt::create_directories() {
    log::info "${MSG_CREATING_DIRS}"

    if ! mkdir -p "$KYUTAI_STT_DATA_DIR" "$KYUTAI_STT_MODELS_DIR"; then
        log::error "${MSG_CREATE_DIRS_FAILED}"
        return 1
    fi

    # The server runs in-container as a non-host UID, so the bind-mounted HF
    # cache must be writable by that UID/GID as well.
    if ! chmod 0777 "$KYUTAI_STT_DATA_DIR" "$KYUTAI_STT_MODELS_DIR"; then
        log::error "Failed to update Kyutai STT data directory permissions"
        return 1
    fi

    log::debug "${MSG_DIRECTORIES_CREATED}"
    return 0
}

#######################################
# Wait for Kyutai STT to be healthy
# Returns: 0 if healthy, 1 on timeout
#######################################
kyutai_stt::wait_for_health() {
    log::info "${MSG_WAITING_STARTUP}"

    local max_attempts=$((KYUTAI_STT_STARTUP_MAX_WAIT / KYUTAI_STT_STARTUP_WAIT_INTERVAL))
    local attempt=1

    while [[ $attempt -le $max_attempts ]]; do
        if kyutai_stt::is_healthy; then
            log::debug "${MSG_HEALTHY}"
            return 0
        fi

        log::debug "Health check attempt $attempt/$max_attempts failed, waiting..."
        sleep "$KYUTAI_STT_STARTUP_WAIT_INTERVAL"
        ((attempt++))
    done

    log::error "${MSG_STARTUP_TIMEOUT}"
    return 1
}

#######################################
# Check if Kyutai STT API is healthy
# Returns: 0 if healthy, 1 otherwise
#######################################
kyutai_stt::is_healthy() {
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" \
        "$KYUTAI_STT_BASE_URL/health" \
        --max-time "$KYUTAI_STT_API_TIMEOUT" 2>/dev/null)

    [[ "$response" == "200" ]]
}

#######################################
# Check if the loaded model reports ready (model_loaded == true)
# Returns: 0 if model loaded, 1 otherwise
#######################################
kyutai_stt::model_loaded() {
    local loaded
    loaded=$(curl -s "$KYUTAI_STT_BASE_URL/health" \
        --max-time "$KYUTAI_STT_API_TIMEOUT" 2>/dev/null | jq -r '.model_loaded // false' 2>/dev/null)

    [[ "$loaded" == "true" ]]
}

#######################################
# Check if GPU is available
# Returns: 0 if available, 1 otherwise
#######################################
kyutai_stt::is_gpu_available() {
    if system::is_command "nvidia-smi"; then
        nvidia-smi >/dev/null 2>&1
        return $?
    fi

    return 1
}

#######################################
# Remove Kyutai STT container and cleanup
# Returns: 0 if successful, 1 otherwise
#######################################
kyutai_stt::cleanup() {
    local success=true

    if common::container_exists; then
        log::info "${MSG_REMOVING_CONTAINER}"
        if ! docker rm -f "$KYUTAI_STT_CONTAINER_NAME" >/dev/null 2>&1; then
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
export -f kyutai_stt::create_directories
export -f kyutai_stt::wait_for_health
export -f kyutai_stt::is_healthy
export -f kyutai_stt::model_loaded
export -f kyutai_stt::is_gpu_available
export -f kyutai_stt::cleanup
