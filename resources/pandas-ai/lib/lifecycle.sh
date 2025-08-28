#!/usr/bin/env bash
set -euo pipefail

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
PANDAS_AI_LIB_DIR="${APP_ROOT}/resources/pandas-ai/lib"

# Source dependencies
source "${PANDAS_AI_LIB_DIR}/common.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${PANDAS_AI_LIB_DIR}/status.sh"
source "${PANDAS_AI_LIB_DIR}/install.sh"

# Start Pandas AI
pandas_ai::start() {
    log::info "Starting Pandas AI"
    
    # Check if already running
    if pandas_ai::is_running; then
        log::warning "Pandas AI is already running"
        return 0
    fi
    
    # Install if needed
    if ! pandas_ai::is_installed; then
        log::info "Pandas AI not installed, installing..."
        pandas_ai::install || return 1
    fi
    
    local port
    port=$(pandas_ai::get_port)
    
    # Start the service
    log::info "Starting Pandas AI service on port ${port}..."
    
    # Set environment variables
    export PANDAS_AI_PORT="${port}"
    
    # Check for OpenAI API key in vault or environment
    if [[ -f "${var_ROOT_DIR}/data/credentials/openrouter-credentials.json" ]]; then
        local api_key
        api_key=$(jq -r '.data.apiKey // empty' "${var_ROOT_DIR}/data/credentials/openrouter-credentials.json" 2>/dev/null || true)
        if [[ -n "${api_key}" ]]; then
            export OPENAI_API_KEY="${api_key}"
        fi
    fi
    
    # Start the server in background
    # Use system python3 if venv doesn't have pip
    local python_cmd="python3"
    if [[ -f "${PANDAS_AI_VENV_DIR}/bin/pip" ]]; then
        python_cmd="${PANDAS_AI_VENV_DIR}/bin/python"
    fi
    
    nohup "${python_cmd}" "${PANDAS_AI_SCRIPTS_DIR}/server.py" \
        > "${PANDAS_AI_LOG_FILE}" 2>&1 &
    
    local pid=$!
    echo "${pid}" > "${PANDAS_AI_PID_FILE}"
    
    # Wait for service to be ready
    local max_attempts=30
    local attempt=0
    while [[ ${attempt} -lt ${max_attempts} ]]; do
        if curl -sf "http://localhost:${port}/health" >/dev/null 2>&1; then
            log::success "Pandas AI started successfully"
            return 0
        fi
        sleep 1
        ((attempt++))
    done
    
    log::error "Failed to start Pandas AI service"
    pandas_ai::stop
    return 1
}

# Stop Pandas AI
pandas_ai::stop() {
    log::info "Stopping Pandas AI"
    
    if [[ -f "${PANDAS_AI_PID_FILE}" ]]; then
        local pid
        pid=$(cat "${PANDAS_AI_PID_FILE}")
        if kill -0 "${pid}" 2>/dev/null; then
            kill "${pid}"
            sleep 2
            if kill -0 "${pid}" 2>/dev/null; then
                kill -9 "${pid}" 2>/dev/null || true
            fi
        fi
        rm -f "${PANDAS_AI_PID_FILE}"
    fi
    
    # Also kill any process on the port
    local port
    port=$(pandas_ai::get_port)
    local pids
    pids=$(lsof -ti:${port} 2>/dev/null || true)
    if [[ -n "${pids}" ]]; then
        echo "${pids}" | xargs kill -9 2>/dev/null || true
    fi
    
    log::success "Pandas AI stopped"
    return 0
}

# Export functions for use by CLI
export -f pandas_ai::start
export -f pandas_ai::stop