#!/usr/bin/env bash
set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
PANDAS_AI_LIB_DIR="${APP_ROOT}/resources/pandas-ai/lib"

# Source dependencies
source "${PANDAS_AI_LIB_DIR}/common.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${PANDAS_AI_LIB_DIR}/status.sh"

# Add/inject analysis script
pandas_ai::content::add() {
    local script_path="${1:-}"
    
    if [[ -z "${script_path}" ]]; then
        log::error "Script path is required"
        echo "Usage: resource-pandas-ai content add <script.py>"
        return 1
    fi
    
    if [[ ! -f "${script_path}" ]]; then
        log::error "Script file not found: ${script_path}"
        return 1
    fi
    
    # Ensure content directory exists
    mkdir -p "${PANDAS_AI_CONTENT_DIR}"
    mkdir -p "${PANDAS_AI_SCRIPTS_DIR}"
    
    # Copy script to content directory
    local script_name
    script_name=$(basename "${script_path}")
    local dest_path="${PANDAS_AI_CONTENT_DIR}/${script_name}"
    local exec_path="${PANDAS_AI_SCRIPTS_DIR}/${script_name}"
    
    log::info "Adding analysis script: ${script_name}"
    cp "${script_path}" "${dest_path}"
    cp "${script_path}" "${exec_path}"
    chmod +x "${exec_path}"
    
    # Validate Python syntax
    local python_cmd="python3"
    if [[ -f "${PANDAS_AI_VENV_DIR}/bin/python" ]] && [[ -f "${PANDAS_AI_VENV_DIR}/bin/pip" ]]; then
        python_cmd="${PANDAS_AI_VENV_DIR}/bin/python"
    fi
    
    if "${python_cmd}" -m py_compile "${exec_path}" 2>/dev/null; then
        log::success "Analysis script added successfully: ${script_name}"
    else
        log::warning "Script may have syntax issues, but added anyway"
        log::success "Analysis script added: ${script_name}"
    fi
    
    return 0
}

# List available analysis scripts
pandas_ai::content::list() {
    log::info "Available analysis scripts:"
    
    if [[ ! -d "${PANDAS_AI_CONTENT_DIR}" ]] || [[ -z "$(ls -A "${PANDAS_AI_CONTENT_DIR}" 2>/dev/null)" ]]; then
        echo "  No analysis scripts found"
        return 0
    fi
    
    local count=0
    for script in "${PANDAS_AI_CONTENT_DIR}"/*.py; do
        if [[ -f "${script}" ]]; then
            local script_name
            script_name=$(basename "${script}")
            echo "  - ${script_name}"
            ((count++))
        fi
    done 2>/dev/null
    
    if [[ ${count} -eq 0 ]]; then
        echo "  No Python analysis scripts found"
    else
        echo ""
        echo "Use 'resource-pandas-ai content get <name>' to view script contents"
        echo "Use 'resource-pandas-ai content execute <name|query>' to run analysis"
    fi
    
    return 0
}

# Get/show analysis script content
pandas_ai::content::get() {
    local script_name="${1:-}"
    
    if [[ -z "${script_name}" ]]; then
        log::error "Script name is required"
        echo "Usage: resource-pandas-ai content get <script.py>"
        return 1
    fi
    
    local script_path="${PANDAS_AI_CONTENT_DIR}/${script_name}"
    if [[ ! -f "${script_path}" ]]; then
        log::error "Script not found: ${script_name}"
        log::info "Available scripts:"
        pandas_ai::content::list
        return 1
    fi
    
    log::header "Analysis Script: ${script_name}"
    cat "${script_path}"
    
    return 0
}

# Remove analysis script
pandas_ai::content::remove() {
    local script_name="${1:-}"
    
    if [[ -z "${script_name}" ]]; then
        log::error "Script name is required"
        echo "Usage: resource-pandas-ai content remove <script.py>"
        return 1
    fi
    
    local content_path="${PANDAS_AI_CONTENT_DIR}/${script_name}"
    local exec_path="${PANDAS_AI_SCRIPTS_DIR}/${script_name}"
    
    if [[ ! -f "${content_path}" ]]; then
        log::error "Script not found: ${script_name}"
        return 1
    fi
    
    rm -f "${content_path}"
    rm -f "${exec_path}"
    
    log::success "Analysis script removed: ${script_name}"
    return 0
}

# Execute analysis (script or query)
pandas_ai::content::execute() {
    local input="${1:-}"
    
    if [[ -z "${input}" ]]; then
        log::error "Input is required (script name or query)"
        echo "Usage: resource-pandas-ai content execute <script.py|query>"
        return 1
    fi
    
    # Check if Pandas AI is running
    if ! pandas_ai::is_running; then
        log::error "Pandas AI is not running"
        log::info "Start it with: resource-pandas-ai manage start"
        return 1
    fi
    
    # Check if input is a script file
    if [[ "${input}" == *.py ]]; then
        local script_path="${PANDAS_AI_SCRIPTS_DIR}/${input}"
        if [[ ! -f "${script_path}" ]]; then
            log::error "Script not found: ${input}"
            log::info "Available scripts:"
            pandas_ai::content::list
            return 1
        fi
        
        log::header "Executing analysis script: ${input}"
        local python_cmd="python3"
        if [[ -f "${PANDAS_AI_VENV_DIR}/bin/python" ]] && [[ -f "${PANDAS_AI_VENV_DIR}/bin/pip" ]]; then
            python_cmd="${PANDAS_AI_VENV_DIR}/bin/python"
        fi
        "${python_cmd}" "${script_path}"
    else
        # Send query to API
        local port
        port=$(pandas_ai::get_port)
        
        log::header "Executing analysis query"
        log::info "Query: ${input}"
        
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
export -f pandas_ai::content::add
export -f pandas_ai::content::list
export -f pandas_ai::content::get
export -f pandas_ai::content::remove
export -f pandas_ai::content::execute