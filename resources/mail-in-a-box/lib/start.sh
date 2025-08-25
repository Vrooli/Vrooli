#!/bin/bash

# Start functions for Mail-in-a-Box resource

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
MAILINABOX_START_LIB_DIR="${APP_ROOT}/resources/mail-in-a-box/lib"

# Source dependencies
source "$MAILINABOX_START_LIB_DIR/core.sh"
source "$MAILINABOX_START_LIB_DIR/install.sh"

# Start Mail-in-a-Box
mailinabox_start() {
    format_header "▶️ Starting Mail-in-a-Box"
    
    # Check if installed
    if ! mailinabox_is_installed; then
        format_warning "Mail-in-a-Box is not installed. Installing now..."
        if ! mailinabox_install; then
            format_error "Failed to install Mail-in-a-Box"
            return 1
        fi
    fi
    
    # Check if already running
    if mailinabox_is_running; then
        format_info "Mail-in-a-Box is already running"
        return 0
    fi
    
    # Start container
    format_info "Starting Mail-in-a-Box container..."
    if docker start "$MAILINABOX_CONTAINER_NAME"; then
        # Wait for service to be ready
        format_info "Waiting for Mail-in-a-Box to be ready..."
        local max_attempts=30
        local attempt=0
        
        while [[ $attempt -lt $max_attempts ]]; do
            if mailinabox_is_running && [[ "$(mailinabox_get_health)" == "healthy" ]]; then
                format_success "Mail-in-a-Box started successfully"
                format_info "Admin Panel: https://${MAILINABOX_BIND_ADDRESS}:${MAILINABOX_PORT_ADMIN}/admin"
                format_info "Webmail: https://${MAILINABOX_BIND_ADDRESS}/mail"
                format_info "Default admin: ${MAILINABOX_ADMIN_EMAIL}"
                return 0
            fi
            sleep 2
            ((attempt++))
        done
        
        format_warning "Mail-in-a-Box started but may not be fully ready"
        return 0
    else
        format_error "Failed to start Mail-in-a-Box"
        return 1
    fi
}