#!/usr/bin/env bash
################################################################################
# MusicGen Docker Management Functions
################################################################################

# Restart MusicGen container
musicgen::restart() {
    log::info "Restarting MusicGen..."
    musicgen::stop "$@"
    musicgen::start "$@"
}

# Show MusicGen logs
musicgen::logs() {
    local lines="${1:-50}"
    if docker ps -q -f name=musicgen >/dev/null 2>&1; then
        docker logs --tail "$lines" musicgen 2>&1
    else
        log::warning "MusicGen container is not running"
        return 1
    fi
}