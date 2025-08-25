#!/usr/bin/env bash
# SimPy stop module

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
SIMPY_STOP_DIR="${APP_ROOT}/resources/simpy/lib"

# Source dependencies
# shellcheck disable=SC1091
source "${SIMPY_STOP_DIR}/core.sh"

#######################################
# Stop SimPy service
#######################################
simpy::stop() {
    log::header "Stopping SimPy"
    
    if ! simpy::is_running; then
        log::info "SimPy is not running"
        return 0
    fi
    
    local pid
    pid=$(simpy::get_pid)
    
    if [[ -n "$pid" ]]; then
        log::info "Stopping SimPy (PID: $pid)..."
        kill "$pid" 2>/dev/null
        
        # Wait for process to stop
        local max_attempts=10
        local attempt=0
        
        while [[ $attempt -lt $max_attempts ]]; do
            if ! simpy::is_running; then
                log::success "SimPy stopped successfully"
                return 0
            fi
            sleep 1
            ((attempt++))
        done
        
        # Force kill if still running
        log::warning "Force stopping SimPy..."
        kill -9 "$pid" 2>/dev/null
        sleep 1
        
        if ! simpy::is_running; then
            log::success "SimPy force stopped"
            return 0
        fi
    fi
    
    log::error "Failed to stop SimPy"
    return 1
}