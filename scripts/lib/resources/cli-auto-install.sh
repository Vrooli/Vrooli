#!/usr/bin/env bash
################################################################################
# Resource CLI Auto-Installation Helper
# 
# Provides a simple function that resources can call during installation
# to automatically install their CLI if they have one.
#
# Usage (in resource install function):
#   resource_cli::auto_install
#
################################################################################

# Auto-install CLI for current resource
resource_cli::auto_install() {
    local resource_dir="${1:-${RESOURCE_DIR:-${BASH_SOURCE[0]%/*}}}"
    
    # Check if resource has a CLI
    if [[ ! -f "${resource_dir}/cli.sh" ]]; then
        return 0  # No CLI, nothing to do
    fi
    
    # Find the universal installer
    local cli_installer
    if [[ -n "${var_SCRIPTS_RESOURCES_LIB_DIR:-}" ]]; then
        cli_installer="${var_SCRIPTS_RESOURCES_LIB_DIR}/cli/install-resource-cli.sh"
    else
        # Fallback path resolution
        local script_dir
        script_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
        cli_installer="${script_dir}/cli/install-resource-cli.sh"
    fi
    
    if [[ ! -f "$cli_installer" ]]; then
        log::warning "Universal CLI installer not found at $cli_installer"
        return 1
    fi
    
    # Install the CLI
    log::info "Installing resource CLI..."
    if "$cli_installer" "$resource_dir"; then
        return 0
    else
        log::warning "CLI installation failed (non-critical)"
        return 1
    fi
}

# Export the function for use by resources
export -f resource_cli::auto_install