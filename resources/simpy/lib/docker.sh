#!/usr/bin/env bash
# SimPy Docker-compatible service management
# Note: SimPy runs as a native Python service, not Docker

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SIMPY_DOCKER_DIR="${APP_ROOT}/resources/simpy/lib"

# Source dependencies
# shellcheck disable=SC1091
source "${SIMPY_DOCKER_DIR}/core.sh"
# shellcheck disable=SC1091
source "${SIMPY_DOCKER_DIR}/start.sh"
# shellcheck disable=SC1091
source "${SIMPY_DOCKER_DIR}/stop.sh"

#######################################
# Start SimPy service (Docker interface)
#######################################
simpy::docker::start() {
    simpy::start "$@"
}

#######################################
# Stop SimPy service (Docker interface)
#######################################
simpy::docker::stop() {
    simpy::stop "$@"
}

#######################################
# Restart SimPy service (Docker interface)
#######################################
simpy::docker::restart() {
    log::info "Restarting SimPy service..."
    simpy::stop "$@" || true
    sleep 2
    simpy::start "$@"
}

#######################################
# Show SimPy logs (Docker interface)
#######################################
simpy::docker::logs() {
    local lines="${1:-50}"
    
    if [[ ! -f "$SIMPY_LOG_FILE" ]]; then
        log::info "No log file found at: $SIMPY_LOG_FILE"
        return 0
    fi
    
    log::info "SimPy service logs (last $lines lines):"
    tail -n "$lines" "$SIMPY_LOG_FILE"
}