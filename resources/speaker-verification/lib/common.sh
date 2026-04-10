#!/usr/bin/env bash
# Speaker Verification - Common Utilities

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
SV_LIB_DIR="${APP_ROOT}/resources/speaker-verification/lib"
SV_CONFIG_DIR="${APP_ROOT}/resources/speaker-verification/config"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${SV_CONFIG_DIR}/defaults.sh"
speaker_verification::export_config 2>/dev/null || true
# shellcheck disable=SC1091
source "${SV_CONFIG_DIR}/messages.sh" 2>/dev/null || true
messages::export_messages 2>/dev/null || true

#######################################
# Check Docker is available and running
# Returns: 0 if Docker is available, 1 otherwise
#######################################
common::check_docker() {
    if ! command -v docker &>/dev/null; then
        log::error "$MSG_DOCKER_NOT_FOUND"
        return 1
    fi
    if ! docker info &>/dev/null; then
        log::error "Docker daemon is not running or current user lacks permissions"
        return 1
    fi
    return 0
}
export -f common::check_docker

#######################################
# Check if the container exists (running or stopped)
# Returns: 0 if exists, 1 if not
#######################################
common::container_exists() {
    docker ps -a --format '{{.Names}}' | grep -q "^${SPEAKER_VERIFICATION_CONTAINER_NAME}$"
}
export -f common::container_exists

#######################################
# Check if the container is running
# Returns: 0 if running, 1 if not
#######################################
common::is_running() {
    docker ps --format '{{.Names}}' | grep -q "^${SPEAKER_VERIFICATION_CONTAINER_NAME}$"
}
export -f common::is_running

#######################################
# Check if a port is available
# Arguments: port number
# Returns: 0 if available, 1 if in use
#######################################
common::is_port_available() {
    local port="${1:?Port required}"
    if command -v system::is_port_in_use &>/dev/null; then
        ! system::is_port_in_use "$port"
    else
        ! ss -tlnp 2>/dev/null | grep -q ":${port} " && \
        ! netstat -tlnp 2>/dev/null | grep -q ":${port} "
    fi
}
export -f common::is_port_available

#######################################
# Create required data directories
#######################################
speaker_verification::create_directories() {
    mkdir -p "${SPEAKER_VERIFICATION_DATA_DIR}"
    mkdir -p "${SPEAKER_VERIFICATION_PROFILES_DIR}"
    mkdir -p "${SPEAKER_VERIFICATION_CACHE_DIR}"
    mkdir -p "${SPEAKER_VERIFICATION_LOG_DIR}"
}
export -f speaker_verification::create_directories

#######################################
# Wait for the service to become healthy
# Arguments: max_wait (optional, default from config)
# Returns: 0 if healthy, 1 if timeout
#######################################
speaker_verification::wait_for_health() {
    local max_wait="${1:-$SPEAKER_VERIFICATION_STARTUP_MAX_WAIT}"
    local interval="${SPEAKER_VERIFICATION_STARTUP_WAIT_INTERVAL}"
    local attempts=$((max_wait / interval))
    local attempt=0

    log::info "$MSG_WAITING_HEALTH"

    while [[ $attempt -lt $attempts ]]; do
        if timeout 5 curl -sf "${SPEAKER_VERIFICATION_BASE_URL}/health" &>/dev/null; then
            log::info "Service is alive, checking readiness..."
            local ready_response
            if ready_response=$(timeout 10 curl -sf "${SPEAKER_VERIFICATION_BASE_URL}/ready" 2>/dev/null) && \
               echo "$ready_response" | jq -e '.status == "ready"' &>/dev/null; then
                log::success "Speaker verification is healthy and ready"
                return 0
            fi
        fi
        ((attempt++))
        sleep "$interval"
    done

    log::error "Speaker verification did not become healthy within ${max_wait}s"
    return 1
}
export -f speaker_verification::wait_for_health

#######################################
# Check if the service is healthy (quick check)
# Returns: 0 if healthy, 1 if not
#######################################
speaker_verification::is_healthy() {
    timeout 5 curl -sf "${SPEAKER_VERIFICATION_BASE_URL}/health" &>/dev/null
}
export -f speaker_verification::is_healthy

#######################################
# Check if the service is ready (model loaded)
# Returns: 0 if ready, 1 if not
#######################################
speaker_verification::is_ready() {
    local ready_response
    ready_response=$(timeout 10 curl -sf "${SPEAKER_VERIFICATION_BASE_URL}/ready" 2>/dev/null) || return 1
    echo "$ready_response" | jq -e '.status == "ready"' &>/dev/null
}
export -f speaker_verification::is_ready

#######################################
# Check if GPU is available
# Returns: 0 if available, 1 if not
#######################################
speaker_verification::is_gpu_available() {
    command -v nvidia-smi &>/dev/null && nvidia-smi &>/dev/null
}
export -f speaker_verification::is_gpu_available

#######################################
# Get the appropriate Docker image
# Returns: image name on stdout
#######################################
speaker_verification::get_docker_image() {
    echo "$SPEAKER_VERIFICATION_IMAGE"
}
export -f speaker_verification::get_docker_image

#######################################
# Clean up container
#######################################
speaker_verification::cleanup() {
    docker rm -f "$SPEAKER_VERIFICATION_CONTAINER_NAME" &>/dev/null || true
}
export -f speaker_verification::cleanup
