#\!/usr/bin/env bash
#
# Browserless stop functions

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/../../../lib/utils/format.sh"
source "$SCRIPT_DIR/common.sh"

function stop_browserless() {
    format_section "🛑 Stopping Browserless"
    
    if \! is_running; then
        format_warning "Browserless is not running"
        return 0
    fi
    
    format_info "Stopping Browserless container..."
    if docker stop "$BROWSERLESS_CONTAINER_NAME" >/dev/null 2>&1; then
        format_success "Container stopped"
    else
        format_warning "Failed to stop container gracefully"
    fi
    
    format_info "Removing container..."
    if docker rm "$BROWSERLESS_CONTAINER_NAME" >/dev/null 2>&1; then
        format_success "Container removed"
    else
        format_warning "Failed to remove container"
    fi
    
    format_success "Browserless stopped successfully"
}
