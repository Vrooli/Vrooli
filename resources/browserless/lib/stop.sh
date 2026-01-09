#!/usr/bin/env bash
#
# Browserless stop functions

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/browserless/lib"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "$SCRIPT_DIR/common.sh"

function stop_browserless() {
    log::subheader "🛑 Stopping Browserless"
    
    # Stop auto-scaler first if running
    if [[ -f "${BROWSERLESS_DATA_DIR}/autoscaler.pid" ]]; then
        log::info "Stopping browser pool auto-scaler..."
        # Source pool manager if not already loaded
        if ! declare -f pool::stop_autoscaler >/dev/null 2>&1; then
            BROWSERLESS_LIB_DIR="${APP_ROOT}/resources/browserless/lib"
            source "${BROWSERLESS_LIB_DIR}/pool-manager.sh"
        fi
        pool::stop_autoscaler
    fi
    
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
