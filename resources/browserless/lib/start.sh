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
    
    # Prepare Docker arguments with conditional host networking
    local docker_args=(
        "--name" "$BROWSERLESS_CONTAINER_NAME"
        "--restart" "unless-stopped"
        "-v" "${BROWSERLESS_WORKSPACE_DIR}:/workspace"
        "--shm-size" "${BROWSERLESS_DOCKER_SHM_SIZE:-2gb}"
        "--cap-add" "${BROWSERLESS_DOCKER_CAPS:-SYS_ADMIN}"
        "--security-opt" "seccomp=${BROWSERLESS_DOCKER_SECCOMP:-unconfined}"
        "--memory=${BROWSERLESS_DOCKER_MEMORY:-2g}"
        "--cpus=${BROWSERLESS_DOCKER_CPUS:-2}"
        "--pids-limit=${BROWSERLESS_DOCKER_PIDS_LIMIT:-512}"
        "-e" "CONCURRENT=${BROWSERLESS_MAX_BROWSERS:-6}"
        "-e" "MAX_CONCURRENT_SESSIONS=${BROWSERLESS_MAX_CONCURRENT_SESSIONS:-6}"
        "-e" "MAX_QUEUE_LENGTH=${BROWSERLESS_MAX_QUEUE_LENGTH:-10}"
        "-e" "TIMEOUT=${BROWSERLESS_TIMEOUT:-30000}"
        "-e" "CONNECTION_TIMEOUT=${BROWSERLESS_CONNECTION_TIMEOUT:-15000}"
        "-e" "WORKER_TIMEOUT=${BROWSERLESS_WORKER_TIMEOUT:-90000}"
        "-e" "CHROME_REFRESH_TIME=${BROWSERLESS_CHROME_REFRESH_MS:-600000}"
        "-e" "SOCKET_CLOSE_TIMEOUT=${BROWSERLESS_SOCKET_CLOSE_TIMEOUT:-5000}"
        "-e" "WORKSPACE_DIR=/workspace"
        "-e" "ENABLE_DEBUGGER=false"
        "-e" "PREBOOT_CHROME=${BROWSERLESS_PREBOOT_CHROME:-true}"
        "-e" "KEEP_ALIVE=${BROWSERLESS_KEEP_ALIVE:-true}"
        "-e" "PRE_REQUEST_HEALTH_CHECK=false"
        "-e" "FUNCTION_ENABLE_INCOGNITO_MODE=true"
        "-e" "WORKSPACE_DELETE_EXPIRED=true"
        "-e" "WORKSPACE_EXPIRE_DAYS=7"
        "-e" "EXIT_ON_HEALTH_FAILURE=true"
        "-e" "HEALTH_CHECK_INTERVAL=10000"
    )
    
    # Configure networking - use host networking for localhost access
    if [[ "${BROWSERLESS_USE_HOST_NETWORK:-yes}" == "yes" ]]; then
        docker_args+=("--network" "host")
        docker_args+=("-e" "PORT=${BROWSERLESS_PORT}")
        log::info "Using host networking for browserless - can access localhost services on port ${BROWSERLESS_PORT}"
    else
        docker_args+=("-p" "${BROWSERLESS_PORT}:3000")
        log::info "Using bridge networking for browserless - localhost services not accessible"
    fi
    
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
            
            # Start auto-scaler if enabled
            if [[ "${BROWSERLESS_ENABLE_AUTOSCALING:-false}" == "true" ]]; then
                log::info "Starting browser pool auto-scaler..."
                # Source pool manager if not already loaded
                if ! declare -f pool::start_autoscaler >/dev/null 2>&1; then
                    source "${BROWSERLESS_LIB_DIR}/pool-manager.sh"
                fi
                pool::start_autoscaler >/dev/null
            fi
            
            # Pre-warm browser pool if enabled
            if [[ "${BROWSERLESS_ENABLE_PREWARM:-true}" == "true" ]]; then
                log::info "Pre-warming browser pool for faster initial response..."
                # Source pool manager if not already loaded
                if ! declare -f pool::prewarm >/dev/null 2>&1; then
                    source "${BROWSERLESS_LIB_DIR}/pool-manager.sh"
                fi
                # Pre-warm in background to not delay startup
                (sleep 5 && pool::prewarm "${BROWSERLESS_PREWARM_COUNT:-2}" >/dev/null 2>&1) &
            fi
            
            return 0
        fi
        sleep 2
        ((attempt++))
    done
    
    log::error "Service failed to become ready"
    return 1
}
