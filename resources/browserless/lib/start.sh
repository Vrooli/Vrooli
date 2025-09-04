#!/usr/bin/env bash
#
# Browserless start functions

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
BROWSERLESS_LIB_DIR="${APP_ROOT}/resources/browserless/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh" || { echo "FATAL: Failed to load variable definitions" >&2; exit 1; }
# shellcheck disable=SC1091
source "${var_LOG_FILE}" || { echo "FATAL: Failed to load logging library" >&2; exit 1; }
# Source browserless common.sh using explicit path to avoid conflicts
source "${BROWSERLESS_LIB_DIR}/common.sh"

function start_browserless() {
    log::header "🚀 Starting Browserless"
    
    # Check Docker daemon is running
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running. Please start Docker first."
        return 1
    fi
    
    # Check if already running
    if is_running; then
        log::warning "Browserless is already running"
        return 0
    fi
    
    # Check for existing stopped container and remove it
    local container_status
    container_status=$(get_container_status)
    if [[ "$container_status" == "exited" ]] || [[ "$container_status" == "created" ]] || [[ "$container_status" == "dead" ]]; then
        log::info "Removing existing stopped container"
        docker rm -f "$BROWSERLESS_CONTAINER_NAME" >/dev/null 2>&1 || true
    fi
    
    # Ensure directories exist
    ensure_directories
    
    # Prepare environment variables
    local docker_args=(
        "--name" "$BROWSERLESS_CONTAINER_NAME"
        "--restart" "unless-stopped"
        "-p" "${BROWSERLESS_PORT}:3000"
        "-v" "${BROWSERLESS_WORKSPACE_DIR}:/workspace"
        "--shm-size" "2gb"
        "-e" "CONNECTION_TIMEOUT=60000"
        "-e" "MAX_CONCURRENT_SESSIONS=${BROWSERLESS_MAX_CONCURRENT_SESSIONS}"
        "-e" "WORKSPACE_DIR=/workspace"
        "-e" "ENABLE_DEBUGGER=false"
        "-e" "PREBOOT_CHROME=true"
        "-e" "KEEP_ALIVE=true"
    )
    
    # Add token if configured
    if [[ -n "${BROWSERLESS_TOKEN}" ]]; then
        docker_args+=("-e" "TOKEN=${BROWSERLESS_TOKEN}")
    fi
    
    # Start container
    log::info "Starting Browserless container..."
    if docker run -d "${docker_args[@]}" "$BROWSERLESS_IMAGE" >/dev/null; then
        log::success "Container started successfully"
    else
        log::error "Failed to start container"
        return 1
    fi
    
    # Wait for service to be ready
    log::info "Waiting for service to be ready..."
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if [[ "$(check_health)" == "healthy" ]]; then
            log::success "Browserless is ready"
            log::info "Access at: http://localhost:${BROWSERLESS_PORT}"
            return 0
        fi
        sleep 2
        ((attempt++))
    done
    
    log::error "Service failed to become ready"
    return 1
}
