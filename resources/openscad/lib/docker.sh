#!/usr/bin/env bash
# OpenSCAD Docker Functions

# Start OpenSCAD container - alias to lifecycle function
openscad::docker::start() {
    openscad::start "$@"
}

# Stop OpenSCAD container - alias to lifecycle function  
openscad::docker::stop() {
    openscad::stop "$@"
}

# Restart OpenSCAD container - alias to lifecycle function
openscad::docker::restart() {
    openscad::restart "$@"
}

# Show OpenSCAD logs
openscad::docker::logs() {
    local lines="${1:-50}"
    
    if openscad::is_running; then
        log::info "Showing last ${lines} lines of OpenSCAD logs"
        docker logs "${OPENSCAD_CONTAINER_NAME}" --tail "$lines"
    else
        log::warn "OpenSCAD container not running"
        # Show logs from last run if available
        if openscad::container_exists; then
            log::info "Showing logs from stopped container"
            docker logs "${OPENSCAD_CONTAINER_NAME}" --tail "$lines"
        else
            log::info "No container logs available"
        fi
    fi
}