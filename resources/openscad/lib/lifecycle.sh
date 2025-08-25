#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENSCAD_LIFECYCLE_DIR="${APP_ROOT}/resources/openscad/lib"

# Dependencies are expected to be sourced by caller

# Start OpenSCAD container
openscad::start() {
    if openscad::is_running; then
        log::info "OpenSCAD is already running"
        return 0
    fi
    
    log::info "Starting OpenSCAD..."
    
    # Ensure directories exist
    openscad::ensure_dirs
    
    # Check if container exists but stopped
    if openscad::container_exists; then
        log::info "Starting existing OpenSCAD container..."
        if docker start "${OPENSCAD_CONTAINER_NAME}"; then
            log::success "OpenSCAD started successfully"
            return 0
        else
            log::warning "Failed to start existing container, removing and recreating..."
            docker rm "${OPENSCAD_CONTAINER_NAME}" >/dev/null 2>&1 || true
        fi
    fi
    
    # Create and start new container
    log::info "Creating new OpenSCAD container..."
    if docker run -d \
        --name "${OPENSCAD_CONTAINER_NAME}" \
        --network vrooli-network \
        -v "${OPENSCAD_SCRIPTS_DIR}:/scripts:ro" \
        -v "${OPENSCAD_OUTPUT_DIR}:/output" \
        -v "${OPENSCAD_INJECTED_DIR}:/injected:ro" \
        --entrypoint "/bin/sh" \
        "${OPENSCAD_IMAGE}" \
        -c "while true; do sleep 3600; done"; then
        
        log::success "OpenSCAD started successfully"
        return 0
    else
        log::error "Failed to start OpenSCAD"
        return 1
    fi
}

# Stop OpenSCAD container
openscad::stop() {
    if ! openscad::is_running; then
        log::info "OpenSCAD is not running"
        return 0
    fi
    
    log::info "Stopping OpenSCAD..."
    
    if docker stop "${OPENSCAD_CONTAINER_NAME}" >/dev/null 2>&1; then
        log::success "OpenSCAD stopped successfully"
        return 0
    else
        log::error "Failed to stop OpenSCAD"
        return 1
    fi
}

# Restart OpenSCAD
openscad::restart() {
    log::info "Restarting OpenSCAD..."
    
    openscad::stop || true
    sleep 2
    openscad::start
}