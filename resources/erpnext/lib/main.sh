#!/bin/bash
# ERPNext Resource Library - Main functions

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
ERPNEXT_LIB_DIR="${APP_ROOT}/resources/erpnext/lib"
ERPNEXT_RESOURCE_DIR="${APP_ROOT}/resources/erpnext"

# Source utilities and dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh" || return 1
source "${var_LIB_UTILS_DIR}/format.sh" || return 1
source "${var_LIB_UTILS_DIR}/log.sh" || return 1

# Source ERPNext specific libraries
source "$ERPNEXT_LIB_DIR/config.sh" || return 1
source "$ERPNEXT_LIB_DIR/docker.sh" || return 1
source "$ERPNEXT_LIB_DIR/status.sh" || return 1
source "$ERPNEXT_LIB_DIR/inject.sh" || return 1

# Start ERPNext
erpnext::start() {
    log::info "Starting ERPNext..."
    
    if erpnext::is_running; then
        log::warn "ERPNext is already running"
        return 0
    fi
    
    # Start using Docker Compose
    erpnext::docker::start || {
        log::error "Failed to start ERPNext"
        return 1
    }
    
    # Wait for health
    local max_wait=60
    local waited=0
    while [ $waited -lt $max_wait ]; do
        if erpnext::is_healthy; then
            log::success "ERPNext started successfully"
            return 0
        fi
        sleep 2
        ((waited+=2))
    done
    
    log::error "ERPNext failed to become healthy within ${max_wait} seconds"
    return 1
}

# Stop ERPNext
erpnext::stop() {
    log::info "Stopping ERPNext..."
    
    if ! erpnext::is_running; then
        log::warn "ERPNext is not running"
        return 0
    fi
    
    erpnext::docker::stop || {
        log::error "Failed to stop ERPNext"
        return 1
    }
    
    log::success "ERPNext stopped successfully"
    return 0
}

# Restart ERPNext
erpnext::restart() {
    log::info "Restarting ERPNext..."
    erpnext::stop || return 1
    sleep 2
    erpnext::start || return 1
    return 0
}

# Install ERPNext
erpnext::install() {
    log::info "Installing ERPNext..."
    
    # Check if already installed
    if erpnext::is_installed; then
        log::warn "ERPNext is already installed"
        return 0
    fi
    
    # Create directories
    local data_dir="${HOME}/.erpnext"
    mkdir -p "$data_dir/apps" "$data_dir/doctypes" "$data_dir/scripts" || {
        log::error "Failed to create data directories"
        return 1
    }
    
    # Pull Docker images
    erpnext::docker::pull || {
        log::error "Failed to pull Docker images"
        return 1
    }
    
    log::success "ERPNext installed successfully"
    return 0
}

# Uninstall ERPNext
erpnext::uninstall() {
    log::info "Uninstalling ERPNext..."
    
    # Stop if running
    if erpnext::is_running; then
        erpnext::stop || {
            log::warn "Failed to stop ERPNext"
        }
    fi
    
    # Remove Docker containers and volumes
    erpnext::docker::remove || {
        log::warn "Failed to remove Docker resources"
    }
    
    log::success "ERPNext uninstalled"
    return 0
}

# Check if installed
erpnext::is_installed() {
    # Check for Docker images - include tags in the search
    docker images --format "{{.Repository}}" | grep -q "^frappe/erpnext$" 2>/dev/null
}

# Check if running
erpnext::is_running() {
    docker ps --format "{{.Names}}" | grep -q "erpnext" 2>/dev/null
}

# Check if healthy
erpnext::is_healthy() {
    # Check if the container is running and bench is responsive
    docker exec erpnext-app bench --version >/dev/null 2>&1
}

# List injected items
erpnext::list() {
    local format="${1:-text}"
    local data_dir="${HOME}/.erpnext"
    
    if [ "$format" = "json" ]; then
        echo '{'
        echo '  "apps": ['
        if [ -d "$data_dir/apps" ]; then
            local first=true
            for app in "$data_dir/apps"/*; do
                if [ -f "$app" ]; then
                    [ "$first" = true ] && first=false || echo ','
                    echo -n "    \"$(basename "$app")\""
                fi
            done
        fi
        echo ''
        echo '  ],'
        echo '  "doctypes": ['
        if [ -d "$data_dir/doctypes" ]; then
            local first=true
            for doctype in "$data_dir/doctypes"/*.json; do
                if [ -f "$doctype" ]; then
                    [ "$first" = true ] && first=false || echo ','
                    echo -n "    \"$(basename "$doctype" .json)\""
                fi
            done
        fi
        echo ''
        echo '  ],'
        echo '  "scripts": ['
        if [ -d "$data_dir/scripts" ]; then
            local first=true
            for script in "$data_dir/scripts"/*.py; do
                if [ -f "$script" ]; then
                    [ "$first" = true ] && first=false || echo ','
                    echo -n "    \"$(basename "$script" .py)\""
                fi
            done
        fi
        echo ''
        echo '  ]'
        echo '}'
    else
        echo "ERPNext Injected Items:"
        echo ""
        echo "Apps:"
        if [ -d "$data_dir/apps" ]; then
            for app in "$data_dir/apps"/*; do
                [ -f "$app" ] && echo "  - $(basename "$app")"
            done
        fi
        echo ""
        echo "DocTypes:"
        if [ -d "$data_dir/doctypes" ]; then
            for doctype in "$data_dir/doctypes"/*.json; do
                [ -f "$doctype" ] && echo "  - $(basename "$doctype" .json)"
            done
        fi
        echo ""
        echo "Scripts:"
        if [ -d "$data_dir/scripts" ]; then
            for script in "$data_dir/scripts"/*.py; do
                [ -f "$script" ] && echo "  - $(basename "$script" .py)"
            done
        fi
    fi
}

# Help message
erpnext::help() {
    cat << EOF
ERPNext Resource Management

Usage: resource-erpnext <command> [options]

Commands:
  start      Start ERPNext services
  stop       Stop ERPNext services
  restart    Restart ERPNext services
  status     Show ERPNext status
  install    Install ERPNext
  uninstall  Uninstall ERPNext
  inject     Inject custom apps/doctypes/scripts
  list       List injected items
  help       Show this help message

Injection Examples:
  resource-erpnext inject --type app myapp.tar.gz
  resource-erpnext inject --type doctype custom.json
  resource-erpnext inject --type script workflow.py

Environment Variables:
  ERPNEXT_PORT     API port (default: 8020)
  ERPNEXT_DATA_DIR Data directory

For more information, see resources/erpnext/README.md
EOF
}