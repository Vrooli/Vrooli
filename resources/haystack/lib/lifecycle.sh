#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
HAYSTACK_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${HAYSTACK_LIB_DIR}/common.sh"
source "${HAYSTACK_LIB_DIR}/../../../../lib/utils/log.sh"
source "${HAYSTACK_LIB_DIR}/status.sh"
source "${HAYSTACK_LIB_DIR}/install.sh"

# Start Haystack
haystack::start() {
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
    
    # Start the server in background
    nohup "${HAYSTACK_VENV_DIR}/bin/python" "${HAYSTACK_SCRIPTS_DIR}/server.py" \
        > "${HAYSTACK_LOG_FILE}" 2>&1 &
    
    local pid=$!
    echo "${pid}" > "${HAYSTACK_PID_FILE}"
    
    # Wait for service to be ready
    local max_attempts=30
    local attempt=0
    while [[ ${attempt} -lt ${max_attempts} ]]; do
        if curl -sf "http://localhost:${port}/health" >/dev/null 2>&1; then
            log::success "Haystack started successfully"
            return 0
        fi
        sleep 1
        ((attempt++))
    done
    
    log::error "Failed to start Haystack service"
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
    
    log::success "Haystack stopped"
    return 0
}

# Restart Haystack
haystack::restart() {
    log::info "Restarting Haystack"
    haystack::stop
    sleep 2
    haystack::start
}