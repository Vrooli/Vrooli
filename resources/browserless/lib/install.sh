#!/usr/bin/env bash
#
# Browserless installation functions

set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
BROWSERLESS_LIB_DIR="${RESOURCE_DIR}/lib"
# shellcheck disable=SC1091
source "${REPO_ROOT}/scripts/lib/utils/var.sh" || { echo "FATAL: Failed to load variable definitions" >&2; exit 1; }
# shellcheck disable=SC1091
source "${var_LOG_FILE}" || { echo "FATAL: Failed to load logging library" >&2; exit 1; }
source "$BROWSERLESS_LIB_DIR/common.sh"

function install_browserless() {
    log::header "📦 Installing Browserless"
    
    # Check Docker daemon is running
    if ! docker info >/dev/null 2>&1; then
        log::error "Docker daemon is not running. Please start Docker first."
        return 1
    fi
    
    # Create directories
    ensure_directories
    log::info "Created data directories"
    
    # Pull Docker image
    log::info "Pulling Browserless image..."
    if docker pull "$BROWSERLESS_IMAGE"; then
        log::success "Successfully pulled $BROWSERLESS_IMAGE"
    else
        log::error "Failed to pull Docker image"
        return 1
    fi
    
    # Start the service
    log::info "Starting Browserless..."
    source "$BROWSERLESS_LIB_DIR/start.sh"
    if start_browserless; then
        log::success "Browserless installed and started successfully"
    else
        log::error "Failed to start Browserless"
        return 1
    fi
    
    # Run initial health check
    sleep 5
    if [[ "$(check_health)" == "healthy" ]]; then
        log::success "Health check passed"
    else
        log::warning "Service started but health check failed"
    fi
    
    log::info "Access Browserless at: http://localhost:${BROWSERLESS_PORT}"
    log::info "API Documentation: http://localhost:${BROWSERLESS_PORT}/docs"
}
