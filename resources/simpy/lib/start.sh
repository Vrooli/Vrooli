#!/usr/bin/env bash
# SimPy start module

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
SIMPY_START_DIR="${APP_ROOT}/resources/simpy/lib"

# Source dependencies
# shellcheck disable=SC1091
source "${SIMPY_START_DIR}/core.sh"
# shellcheck disable=SC1091
source "${SIMPY_START_DIR}/install.sh"

#######################################
# Start SimPy service
#######################################
simpy::start() {
    log::header "Starting SimPy"
    
    # Install if not already installed
    if ! simpy::is_installed; then
        log::info "SimPy not installed, installing now..."
        simpy::install || return 1
    fi
    
    # Check if already running
    if simpy::is_running; then
        log::info "SimPy is already running (PID: $(simpy::get_pid))"
        return 0
    fi
    
    # Export environment variables
    simpy::export_config
    
    # Start service
    log::info "Starting SimPy service on port ${SIMPY_PORT}..."
    cd "$SIMPY_DATA_DIR" || return 1
    
    nohup python3 "$SIMPY_DATA_DIR/simpy-service.py" \
        > "$SIMPY_LOG_FILE" 2>&1 &
    
    local pid=$!
    
    # Wait for service to start
    local max_attempts=30
    local attempt=0
    
    while [[ $attempt -lt $max_attempts ]]; do
        if simpy::test_connection; then
            log::success "SimPy started successfully (PID: $pid)"
            return 0
        fi
        sleep 1
        ((attempt++))
    done
    
    log::error "Failed to start SimPy service"
    return 1
}