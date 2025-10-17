#!/usr/bin/env bash
################################################################################
# MusicGen Test Functions
################################################################################

# Smoke test - verify MusicGen is running and responsive
musicgen::test::smoke() {
    log::info "Running MusicGen smoke test..."
    
    # Check if container is running
    if ! docker ps -q -f name=musicgen >/dev/null 2>&1; then
        log::error "MusicGen container is not running"
        return 1
    fi
    
    # Check if service is responding (example endpoint)
    local musicgen_port="${MUSICGEN_PORT:-7860}"
    if curl -s -f "http://localhost:${musicgen_port}/api/health" >/dev/null 2>&1; then
        log::success "MusicGen is running and responsive on port ${musicgen_port}"
        return 0
    else
        log::error "MusicGen is not responding on port ${musicgen_port}"
        return 1
    fi
}