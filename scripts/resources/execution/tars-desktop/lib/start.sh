#!/bin/bash
# TARS-desktop start/stop functionality

# Get script directory
TARS_DESKTOP_START_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${TARS_DESKTOP_START_DIR}/core.sh"

# Start TARS-desktop
tars_desktop::start() {
    local verbose="${1:-false}"
    
    log::header "Starting TARS-desktop"
    
    # Initialize
    tars_desktop::init "$verbose"
    
    # Check if already running
    if tars_desktop::is_running; then
        log::success "TARS-desktop is already running"
        return 0
    fi
    
    # Check if installed
    if ! tars_desktop::is_installed; then
        log::error "TARS-desktop is not installed. Run 'install' first"
        return 1
    fi
    
    # Start server
    log::info "Starting TARS-desktop server on port $TARS_DESKTOP_PORT..."
    
    # Create log directory
    local log_dir="${var_LOG_DIR:-/home/matthalloran8/Vrooli/logs}/tars-desktop"
    mkdir -p "$log_dir"
    
    # Start server in background
    (
        cd "$TARS_DESKTOP_INSTALL_DIR"
        
        # Determine python command
        if [[ -f "${TARS_DESKTOP_VENV_DIR}/bin/activate" ]]; then
            source "${TARS_DESKTOP_VENV_DIR}/bin/activate"
            PYTHON_CMD="python"
        else
            PYTHON_CMD="python3"
        fi
        
        export TARS_DESKTOP_PORT
        export DISPLAY="${DISPLAY:-:0}"
        nohup $PYTHON_CMD server.py \
            > "${log_dir}/server.log" 2>&1 &
        echo $! > "${TARS_DESKTOP_INSTALL_DIR}/server.pid"
    )
    
    # Wait for server to start
    log::info "Waiting for server to start..."
    local retries=0
    local max_retries=30
    
    while [[ $retries -lt $max_retries ]]; do
        if tars_desktop::health_check; then
            log::success "TARS-desktop started successfully"
            
            # Show capabilities if verbose
            if [[ "$verbose" == "true" ]]; then
                local capabilities
                capabilities=$(tars_desktop::get_capabilities 2>/dev/null)
                if [[ -n "$capabilities" ]]; then
                    log::info "Capabilities:"
                    echo "$capabilities" | jq '.' 2>/dev/null || echo "$capabilities"
                fi
            fi
            
            return 0
        fi
        
        sleep 1
        ((retries++))
    done
    
    log::error "Failed to start TARS-desktop server"
    
    # Check logs for errors
    if [[ -f "${log_dir}/server.log" ]]; then
        log::error "Recent logs:"
        tail -10 "${log_dir}/server.log"
    fi
    
    return 1
}

# Stop TARS-desktop
tars_desktop::stop() {
    local verbose="${1:-false}"
    
    log::header "Stopping TARS-desktop"
    
    # Check if running
    if ! tars_desktop::is_running; then
        log::info "TARS-desktop is not running"
        return 0
    fi
    
    # Try to stop gracefully using PID file
    if [[ -f "${TARS_DESKTOP_INSTALL_DIR}/server.pid" ]]; then
        local pid
        pid=$(cat "${TARS_DESKTOP_INSTALL_DIR}/server.pid")
        if kill -0 "$pid" 2>/dev/null; then
            log::info "Stopping process $pid..."
            kill "$pid"
            sleep 2
            
            # Force kill if still running
            if kill -0 "$pid" 2>/dev/null; then
                log::warn "Process didn't stop gracefully, forcing..."
                kill -9 "$pid"
            fi
        fi
        rm -f "${TARS_DESKTOP_INSTALL_DIR}/server.pid"
    fi
    
    # Kill any remaining processes
    local pids
    pids=$(pgrep -f "tars_desktop.*server")
    if [[ -n "$pids" ]]; then
        log::info "Stopping remaining processes..."
        echo "$pids" | xargs kill -9 2>/dev/null
    fi
    
    # Verify stopped
    if ! tars_desktop::is_running; then
        log::success "TARS-desktop stopped successfully"
        return 0
    else
        log::error "Failed to stop TARS-desktop"
        return 1
    fi
}

# Restart TARS-desktop
tars_desktop::restart() {
    local verbose="${1:-false}"
    
    log::header "Restarting TARS-desktop"
    
    tars_desktop::stop "$verbose"
    sleep 2
    tars_desktop::start "$verbose"
}

# Export functions
export -f tars_desktop::start
export -f tars_desktop::stop
export -f tars_desktop::restart