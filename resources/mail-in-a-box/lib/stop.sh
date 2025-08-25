#!/bin/bash

# Stop functions for Mail-in-a-Box resource

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
MAILINABOX_STOP_LIB_DIR="${APP_ROOT}/resources/mail-in-a-box/lib"

# Source dependencies
source "$MAILINABOX_STOP_LIB_DIR/core.sh"

# Stop Mail-in-a-Box
mailinabox_stop() {
    format_header "⏹️ Stopping Mail-in-a-Box"
    
    # Check if running
    if ! mailinabox_is_running; then
        format_info "Mail-in-a-Box is not running"
        return 0
    fi
    
    # Stop container
    format_info "Stopping Mail-in-a-Box container..."
    if docker stop "$MAILINABOX_CONTAINER_NAME" >/dev/null 2>&1; then
        format_success "Mail-in-a-Box stopped successfully"
        return 0
    else
        format_error "Failed to stop Mail-in-a-Box"
        return 1
    fi
}