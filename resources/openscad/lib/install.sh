#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
OPENSCAD_INSTALL_DIR="${APP_ROOT}/resources/openscad/lib"

# Dependencies are expected to be sourced by caller

# Install OpenSCAD (pull Docker image)
openscad::install() {
    log::info "Installing OpenSCAD..."
    
    if ! openscad::is_installed; then
        log::error "Docker is required but not installed"
        return 1
    fi
    
    # Ensure directories exist
    openscad::ensure_dirs
    
    # Pull OpenSCAD Docker image
    log::info "Pulling OpenSCAD Docker image..."
    if docker pull "${OPENSCAD_IMAGE}"; then
        # Register CLI with vrooli
        local resource_dir="${OPENSCAD_INSTALL_DIR}/.."
        if [[ -f "${resource_dir}/../../../lib/resources/install-resource-cli.sh" ]]; then
            "${resource_dir}/../../../lib/resources/install-resource-cli.sh" "${resource_dir}" 2>/dev/null || true
        fi
        
        log::success "OpenSCAD installed successfully"
        return 0
    else
        log::error "Failed to pull OpenSCAD image"
        return 1
    fi
}

# Uninstall OpenSCAD
openscad::uninstall() {
    log::info "Uninstalling OpenSCAD..."
    
    # Stop container if running
    if openscad::is_running; then
        log::info "Stopping OpenSCAD container..."
        docker stop "${OPENSCAD_CONTAINER_NAME}" >/dev/null 2>&1 || true
    fi
    
    # Remove container if exists
    if openscad::container_exists; then
        log::info "Removing OpenSCAD container..."
        docker rm "${OPENSCAD_CONTAINER_NAME}" >/dev/null 2>&1 || true
    fi
    
    # Remove image
    if docker images --format "{{.Repository}}:{{.Tag}}" | grep -q "^${OPENSCAD_IMAGE}$"; then
        log::info "Removing OpenSCAD image..."
        docker rmi "${OPENSCAD_IMAGE}" >/dev/null 2>&1 || true
    fi
    
    log::success "OpenSCAD uninstalled successfully"
    return 0
}