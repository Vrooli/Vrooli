#!/bin/bash

# Stop functions for Mail-in-a-Box resource

SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
MAILINABOX_STOP_LIB_DIR="${RESOURCE_DIR}/lib"

# Source dependencies
source "${REPO_ROOT}/scripts/lib/utils/log.sh" 2>/dev/null || true
source "$MAILINABOX_STOP_LIB_DIR/core.sh"

# Stop Mail-in-a-Box
mailinabox_stop() {
    log::header "⏹️ Stopping Mail-in-a-Box"
    
    # Stop using docker-compose if available
    if command -v docker-compose &>/dev/null && [[ -f "${REPO_ROOT}/resources/mail-in-a-box/docker-compose.yml" ]]; then
        log::info "Stopping Mail-in-a-Box services with docker-compose..."
        
        # Export environment variables for docker-compose
        export MAILINABOX_DATA_DIR="${MAILINABOX_DATA_DIR:-/var/lib/mailinabox}"
        export MAILINABOX_CONFIG_DIR="${MAILINABOX_CONFIG_DIR:-/var/lib/mailinabox}"
        
        cd "${REPO_ROOT}/resources/mail-in-a-box"
        if docker-compose stop >/dev/null 2>&1; then
            log::success "Mail-in-a-Box services stopped successfully"
            return 0
        else
            log::warning "Docker-compose stop failed, trying individual containers..."
        fi
    fi
    
    # Fallback: Stop individual containers
    log::info "Stopping Mail-in-a-Box containers..."
    local stopped_any=false
    
    for container in "$MAILINABOX_CONTAINER_NAME" "mailinabox-webmail" "mailinabox-caldav"; do
        if docker ps -q -f name="$container" | grep -q .; then
            if docker stop "$container" >/dev/null 2>&1; then
                log::info "Stopped $container"
                stopped_any=true
            fi
        fi
    done
    
    if [[ "$stopped_any" == "true" ]]; then
        log::success "Mail-in-a-Box stopped successfully"
        return 0
    else
        log::info "No Mail-in-a-Box containers were running"
        return 0
    fi
}