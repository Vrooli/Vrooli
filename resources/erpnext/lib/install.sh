#!/bin/bash
# ERPNext Installation Functions

# Install ERPNext
erpnext::install::execute() {
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
erpnext::install::uninstall() {
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
    
    # Remove data directory
    local data_dir="${HOME}/.erpnext"
    if [ -d "$data_dir" ]; then
        rm -rf "$data_dir" || {
            log::warn "Failed to remove data directory: $data_dir"
        }
    fi
    
    # Remove compose file
    local compose_file="$ERPNEXT_RESOURCE_DIR/docker/docker-compose.yml"
    if [ -f "$compose_file" ]; then
        rm -f "$compose_file" || {
            log::warn "Failed to remove compose file"
        }
    fi
    
    log::success "ERPNext uninstalled"
    return 0
}