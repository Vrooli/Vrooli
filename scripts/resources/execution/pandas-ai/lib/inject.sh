#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
PANDAS_AI_LIB_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${PANDAS_AI_LIB_DIR}/common.sh"
source "${PANDAS_AI_LIB_DIR}/../../../../lib/utils/log.sh"
source "${PANDAS_AI_LIB_DIR}/status.sh"

# Inject analysis scripts into Pandas AI
pandas_ai::inject() {
    local script_path="${1:-}"
    
    if [[ -z "${script_path}" ]]; then
        log::error "Script path is required"
        echo "Usage: resource-pandas-ai inject <script.py>"
        return 1
    fi
    
    if [[ ! -f "${script_path}" ]]; then
        log::error "Script file not found: ${script_path}"
        return 1
    fi
    
    # Check if Pandas AI is installed
    if ! pandas_ai::is_installed; then
        log::error "Pandas AI is not installed"
        return 1
    fi
    
    log::header "Injecting script into Pandas AI"
    
    # Copy script to scripts directory
    local script_name
    script_name=$(basename "${script_path}")
    local dest_path="${PANDAS_AI_SCRIPTS_DIR}/${script_name}"
    
    log::info "Copying ${script_name} to Pandas AI scripts directory..."
    cp "${script_path}" "${dest_path}"
    chmod +x "${dest_path}"
    
    # Validate Python syntax
    local python_cmd="python3"
    # Only use venv python if pip is also available (complete venv)
    if [[ -f "${PANDAS_AI_VENV_DIR}/bin/python" ]] && [[ -f "${PANDAS_AI_VENV_DIR}/bin/pip" ]]; then
        python_cmd="${PANDAS_AI_VENV_DIR}/bin/python"
    fi
    
    if "${python_cmd}" -m py_compile "${dest_path}" 2>/dev/null; then
        log::success "Script syntax validated"
    else
        log::warning "Script may have syntax issues"
    fi
    
    log::success "Script injected successfully: ${script_name}"
    log::info "Execute with: resource-pandas-ai analyze ${script_name}"
    
    return 0
}

# Analyze data using injected script or query
pandas_ai::analyze() {
    local input="${1:-}"
    
    if [[ -z "${input}" ]]; then
        log::error "Input is required (script name or query)"
        echo "Usage: resource-pandas-ai analyze <script.py|query>"
        return 1
    fi
    
    # Check if Pandas AI is running
    if ! pandas_ai::is_running; then
        log::error "Pandas AI is not running"
        log::info "Start it with: resource-pandas-ai start"
        return 1
    fi
    
    # Check if input is a script file
    if [[ "${input}" == *.py ]]; then
        local script_path="${PANDAS_AI_SCRIPTS_DIR}/${input}"
        if [[ ! -f "${script_path}" ]]; then
            log::error "Script not found: ${input}"
            log::info "Available scripts:"
            ls -1 "${PANDAS_AI_SCRIPTS_DIR}/"*.py 2>/dev/null | xargs -n1 basename || echo "  None"
            return 1
        fi
        
        log::header "Running analysis script: ${input}"
        local python_cmd="python3"
        # Only use venv python if pip is also available (complete venv)
        if [[ -f "${PANDAS_AI_VENV_DIR}/bin/python" ]] && [[ -f "${PANDAS_AI_VENV_DIR}/bin/pip" ]]; then
            python_cmd="${PANDAS_AI_VENV_DIR}/bin/python"
        fi
        "${python_cmd}" "${script_path}"
    else
        # Send query to API
        local port
        port=$(pandas_ai::get_port)
        
        log::header "Sending analysis query to Pandas AI"
        
        # Create a simple test dataset for the query
        local response
        response=$(curl -sf -X POST "http://localhost:${port}/analyze" \
            -H "Content-Type: application/json" \
            -d "{
                \"data\": {
                    \"values\": [1, 2, 3, 4, 5],
                    \"labels\": [\"A\", \"B\", \"C\", \"D\", \"E\"]
                },
                \"query\": \"${input}\"
            }" 2>/dev/null || echo '{"success": false, "error": "Failed to connect"}')
        
        if echo "${response}" | jq -r '.success' 2>/dev/null | grep -q true; then
            log::success "Analysis completed"
            echo "${response}" | jq -r '.result'
        else
            log::error "Analysis failed"
            echo "${response}" | jq -r '.error // "Unknown error"' 2>/dev/null || echo "${response}"
            return 1
        fi
    fi
    
    return 0
}

# Export functions for use by CLI
export -f pandas_ai::inject
export -f pandas_ai::analyze