#!/bin/bash

# Start functions for Mail-in-a-Box resource

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
MAILINABOX_START_LIB_DIR="${APP_ROOT}/resources/mail-in-a-box/lib"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true
source "$MAILINABOX_START_LIB_DIR/core.sh"
source "$MAILINABOX_START_LIB_DIR/install.sh"

# Start Mail-in-a-Box
mailinabox_start() {
    log::header "▶️ Starting Mail-in-a-Box"
    
    # Check if installed
    if ! mailinabox_is_installed; then
        log::warning "Mail-in-a-Box is not installed. Installing now..."
        if ! mailinabox_install; then
            log::error "Failed to install Mail-in-a-Box"
            return 1
        fi
    fi
    
    # Create directories for Radicale if needed
    mkdir -p "${MAILINABOX_DATA_DIR:-/var/lib/mailinabox}/radicale/data"
    mkdir -p "${MAILINABOX_CONFIG_DIR:-/var/lib/mailinabox}/radicale/config"
    
    # Check if already running
    if mailinabox_is_running; then
        log::info "Mail-in-a-Box is already running"
        return 0
    fi
    
    # Start containers using docker-compose if available
    if command -v docker-compose &>/dev/null && [[ -f "$APP_ROOT/resources/mail-in-a-box/docker-compose.yml" ]]; then
        log::info "Starting Mail-in-a-Box with docker-compose (includes webmail and CalDAV)..."
        
        # Export environment variables for docker-compose
        export MAILINABOX_DATA_DIR
        export MAILINABOX_CONFIG_DIR
        
        cd "$APP_ROOT/resources/mail-in-a-box"
        if docker-compose up -d; then
            log::info "Waiting for services to be ready..."
            local max_attempts=30
            local attempt=0
            
            while [[ $attempt -lt $max_attempts ]]; do
                if mailinabox_is_running && [[ "$(mailinabox_get_health)" == "healthy" ]]; then
                    log::success "Mail-in-a-Box started successfully"
                    log::info "Email Server: SMTP port ${MAILINABOX_PORT_SMTP}, IMAP port ${MAILINABOX_PORT_IMAPS}"
                    log::info "Webmail: http://${MAILINABOX_BIND_ADDRESS}:8080"
                    log::info "CalDAV/CardDAV: http://${MAILINABOX_BIND_ADDRESS}:5232"
                    log::info "Default admin: ${MAILINABOX_ADMIN_EMAIL}"
                    return 0
                fi
                sleep 2
                ((attempt++))
            done
            
            log::warning "Services started but may not be fully ready"
            return 0
        else
            log::warning "Docker-compose failed, trying basic start..."
        fi
    fi
    
    # Fallback to basic docker start
    log::info "Starting Mail-in-a-Box container (basic mode)..."
    if docker start "$MAILINABOX_CONTAINER_NAME"; then
        # Wait for service to be ready
        log::info "Waiting for Mail-in-a-Box to be ready..."
        local max_attempts=30
        local attempt=0
        
        while [[ $attempt -lt $max_attempts ]]; do
            if mailinabox_is_running && [[ "$(mailinabox_get_health)" == "healthy" ]]; then
                log::success "Mail-in-a-Box started successfully"
                log::info "Email Server: SMTP port ${MAILINABOX_PORT_SMTP}, IMAP port ${MAILINABOX_PORT_IMAPS}"
                log::info "Default admin: ${MAILINABOX_ADMIN_EMAIL}"
                return 0
            fi
            sleep 2
            ((attempt++))
        done
        
        log::warning "Mail-in-a-Box started but may not be fully ready"
        return 0
    else
        log::error "Failed to start Mail-in-a-Box"
        return 1
    fi
}