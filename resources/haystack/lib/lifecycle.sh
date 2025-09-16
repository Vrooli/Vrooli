#!/usr/bin/env bash
set -euo pipefail

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
HAYSTACK_LIB_DIR="${APP_ROOT}/resources/haystack/lib"

# Source dependencies
source "${HAYSTACK_LIB_DIR}/common.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${HAYSTACK_LIB_DIR}/status.sh"
source "${HAYSTACK_LIB_DIR}/install.sh"

# Start Haystack
haystack::start() {
    # Handle --wait flag (it's the default behavior anyway)
    local wait_flag=true
    if [[ "${1:-}" == "--wait" ]]; then
        wait_flag=true
    fi
    
    log::info "Starting Haystack"
    
    # Check if already running
    if haystack::is_running; then
        log::warning "Haystack is already running"
        return 0
    fi
    
    # Install if needed
    if ! haystack::is_installed; then
        log::info "Haystack not installed, installing..."
        haystack::install || return 1
    fi
    
    local port
    port=$(haystack::get_port)
    
    # Start the service
    log::info "Starting Haystack service on port ${port}..."
    
    # Set environment variables
    export HAYSTACK_PORT="${port}"
    
    # Check for OpenRouter API key if available
    if [[ -f "${var_ROOT_DIR}/data/credentials/openrouter-credentials.json" ]]; then
        local api_key
        api_key=$(jq -r '.data.apiKey // empty' "${var_ROOT_DIR}/data/credentials/openrouter-credentials.json" 2>/dev/null || true)
        if [[ -n "${api_key}" ]]; then
            export OPENAI_API_KEY="${api_key}"
        fi
    fi
    
    # Ensure log directory exists
    mkdir -p "${HAYSTACK_LOG_DIR}"
    
    # Start the server in background
    nohup "${HAYSTACK_VENV_DIR}/bin/python" "${HAYSTACK_SCRIPTS_DIR}/server.py" \
        >> "${HAYSTACK_LOG_FILE}" 2>&1 &
    
    local pid=$!
    echo "${pid}" > "${HAYSTACK_PID_FILE}"
    
    # Give the process a moment to start before checking
    sleep 2
    
    # Wait for service to be ready
    local max_attempts=30
    local attempt=0
    log::info "Waiting for Haystack to become healthy (up to ${max_attempts} seconds)..."
    
    while [[ ${attempt} -lt ${max_attempts} ]]; do
        # Check if process is still running
        if ! kill -0 "${pid}" 2>/dev/null; then
            log::error "Haystack process died during startup"
            rm -f "${HAYSTACK_PID_FILE}"
            return 1
        fi
        
        # Try health check
        if timeout 5 curl -sf "http://localhost:${port}/health" &>/dev/null; then
            log::success "Haystack started successfully"
            return 0
        fi
        
        # Show progress every 5 attempts
        if [[ $((attempt % 5)) -eq 0 ]] && [[ ${attempt} -gt 0 ]]; then
            log::info "Still waiting for Haystack to start (${attempt}/${max_attempts})..."
        fi
        
        sleep 1
        ((attempt++))
    done
    
    log::error "Failed to start Haystack service - health check never responded"
    haystack::stop
    return 1
}

# Stop Haystack
haystack::stop() {
    log::info "Stopping Haystack"
    
    if [[ -f "${HAYSTACK_PID_FILE}" ]]; then
        local pid
        pid=$(cat "${HAYSTACK_PID_FILE}")
        if kill -0 "${pid}" 2>/dev/null; then
            kill "${pid}"
            sleep 2
            if kill -0 "${pid}" 2>/dev/null; then
                kill -9 "${pid}" 2>/dev/null || true
            fi
        fi
        rm -f "${HAYSTACK_PID_FILE}"
    fi
    
    # Also kill any process on the port
    local port
    port=$(haystack::get_port)
    local pids
    pids=$(lsof -ti:${port} 2>/dev/null || true)
    if [[ -n "${pids}" ]]; then
        echo "${pids}" | xargs kill -9 2>/dev/null || true
    fi
    
    # Wait for port to be free
    local retries=10
    while [[ $retries -gt 0 ]]; do
        if ! lsof -ti:${port} &>/dev/null; then
            break
        fi
        sleep 1
        retries=$((retries - 1))
    done
    
    log::success "Haystack stopped"
    return 0
}

# Restart Haystack
haystack::restart() {
    log::info "Restarting Haystack"
    
    # Stop the service and ensure it's fully stopped
    haystack::stop
    
    # Give a bit more time for port to be released
    sleep 3
    
    # Pass any arguments to start (like --wait)
    # The start function already handles --wait by default
    if haystack::start "$@"; then
        log::success "Haystack restarted successfully"
        return 0
    else
        log::error "Failed to restart Haystack"
        return 1
    fi
}