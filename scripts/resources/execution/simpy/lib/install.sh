#!/bin/bash

# SimPy Resource - Installation Functions
set -euo pipefail

# Get the script directory
SIMPY_INSTALL_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source common functions
source "$SIMPY_INSTALL_LIB_DIR/common.sh"

# Install SimPy
install::main() {
    log::header "📦 Installing SimPy"
    
    # Check if already installed
    if simpy::is_installed; then
        log::info "SimPy is already installed"
        local version=$(simpy::get_version)
        log::success "Version: $version"
        return 0
    fi
    
    # Create directories
    log::info "Creating SimPy directories..."
    mkdir -p "$SIMPY_SIMULATIONS_DIR"
    mkdir -p "$SIMPY_OUTPUTS_DIR"
    mkdir -p "$SIMPY_LOGS_DIR"
    
    # Build Docker image
    log::info "Building SimPy Docker image..."
    local dockerfile_dir="$SIMPY_DIR/docker"
    
    if [[ ! -f "$dockerfile_dir/Dockerfile" ]]; then
        log::error "Dockerfile not found at: $dockerfile_dir/Dockerfile"
        return 1
    fi
    
    if docker build -t "$SIMPY_IMAGE_NAME" "$dockerfile_dir" &>/dev/null; then
        log::success "Docker image built successfully"
    else
        log::error "Failed to build Docker image"
        return 1
    fi
    
    # Copy example simulations
    if [[ -d "$SIMPY_EXAMPLES_DIR" ]]; then
        log::info "Copying example simulations..."
        cp -r "$SIMPY_EXAMPLES_DIR"/* "$SIMPY_SIMULATIONS_DIR/" 2>/dev/null || true
    fi
    
    # Register CLI with Vrooli
    log::info "Registering SimPy CLI with Vrooli..."
    "$SIMPY_LIB_DIR/../../../../lib/resources/install-resource-cli.sh" "$SIMPY_DIR/cli.sh" "$SIMPY_RESOURCE_NAME"
    
    # Verify installation
    if simpy::is_installed; then
        local version=$(simpy::get_version)
        log::success "✅ SimPy installed successfully (version: $version)"
        return 0
    else
        log::error "❌ SimPy installation failed"
        return 1
    fi
}

# Uninstall SimPy
uninstall::main() {
    log::header "🗑️  Uninstalling SimPy"
    
    # Remove Docker image
    if docker image inspect "$SIMPY_IMAGE_NAME" &>/dev/null; then
        log::info "Removing Docker image..."
        docker rmi "$SIMPY_IMAGE_NAME" &>/dev/null || true
    fi
    
    log::success "✅ SimPy uninstalled successfully"
    return 0
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    case "${1:-install}" in
        install)
            install::main
            ;;
        uninstall)
            uninstall::main
            ;;
        *)
            echo "Usage: $0 [install|uninstall]"
            exit 1
            ;;
    esac
fi