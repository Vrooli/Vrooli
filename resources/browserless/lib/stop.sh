#!/usr/bin/env bash
#
# Browserless stop functions

set -euo pipefail

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
SCRIPT_DIR="${RESOURCE_DIR}/lib"
source "${REPO_ROOT}/scripts/lib/utils/format.sh"
source "${REPO_ROOT}/scripts/lib/utils/log.sh"
source "$SCRIPT_DIR/common.sh"

function stop_browserless() {
    log::subheader "🛑 Stopping Browserless"
    
    if ! is_running; then
        log::warning "Browserless is not running"
        return 0
    fi
    
    log::info "Stopping Browserless container..."
    if docker stop "$BROWSERLESS_CONTAINER_NAME" >/dev/null 2>&1; then
        log::success "Container stopped"
    else
        log::warning "Failed to stop container gracefully"
    fi
    
    log::info "Removing container..."
    if docker rm "$BROWSERLESS_CONTAINER_NAME" >/dev/null 2>&1; then
        log::success "Container removed"
    else
        log::warning "Failed to remove container"
    fi
    
    log::success "Browserless stopped successfully"
}
