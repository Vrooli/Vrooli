#\!/usr/bin/env bash
#
# Browserless start functions

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../../../lib/utils/format.sh"
source "$SCRIPT_DIR/common.sh"

function start_browserless() {
    format_section "🚀 Starting Browserless"
    
    # Check if already running
    if is_running; then
        format_warning "Browserless is already running"
        return 0
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
    format_info "Starting Browserless container..."
    if docker run -d "${docker_args[@]}" "$BROWSERLESS_IMAGE" >/dev/null; then
        format_success "Container started successfully"
    else
        format_error "Failed to start container"
        return 1
    fi
    
    # Wait for service to be ready
    format_info "Waiting for service to be ready..."
    local max_attempts=30
    local attempt=0
    
    while [ $attempt -lt $max_attempts ]; do
        if [[ "$(check_health)" == "healthy" ]]; then
            format_success "Browserless is ready"
            format_info "Access at: http://localhost:${BROWSERLESS_PORT}"
            return 0
        fi
        sleep 2
        ((attempt++))
    done
    
    format_error "Service failed to become ready"
    return 1
}
