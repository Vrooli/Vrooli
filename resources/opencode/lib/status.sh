#!/bin/bash
# OpenCode Status Functions

# Source common functions
source "${BASH_SOURCE[0]%/*}/common.sh"

opencode::status() {
    local status_code=0
    
    log::info "OpenCode Extension Status:"
    echo "=========================="
    
    # Check VS Code availability
    if [[ -n "${VSCODE_COMMAND}" ]]; then
        echo "✅ VS Code Command: ${VSCODE_COMMAND}"
    else
        echo "❌ VS Code Command: Not found"
        status_code=1
    fi
    
    # Check extension installation
    if opencode_is_installed; then
        local version=$(opencode_get_version)
        echo "✅ Twinny Extension: Installed (v${version})"
    else
        echo "❌ Twinny Extension: Not installed"
        status_code=1
    fi
    
    # Check configuration
    if [[ -f "${OPENCODE_CONFIG_FILE}" ]]; then
        echo "✅ Configuration: ${OPENCODE_CONFIG_FILE}"
        
        # Show key configuration details
        if command -v jq &>/dev/null; then
            local provider=$(jq -r '.provider // "not_set"' "${OPENCODE_CONFIG_FILE}")
            local chat_model=$(jq -r '.chat_model // "not_set"' "${OPENCODE_CONFIG_FILE}")
            local completion_model=$(jq -r '.completion_model // "not_set"' "${OPENCODE_CONFIG_FILE}")
            
            echo "   Provider: ${provider}"
            echo "   Chat Model: ${chat_model}"
            echo "   Completion Model: ${completion_model}"
        fi
    else
        echo "⚠️  Configuration: Not found (${OPENCODE_CONFIG_FILE})"
    fi
    
    # Check data directory
    if [[ -d "${OPENCODE_DATA_DIR}" ]]; then
        echo "✅ Data Directory: ${OPENCODE_DATA_DIR}"
    else
        echo "⚠️  Data Directory: Not found (${OPENCODE_DATA_DIR})"
    fi
    
    # Check available models
    local models=$(opencode_get_models)
    local model_count=$(echo "${models}" | jq '. | length')
    if [[ "${model_count}" -gt 0 ]]; then
        echo "✅ Available Models: ${model_count} found"
        if [[ "${model_count}" -le 5 ]]; then
            echo "${models}" | jq -r '.[]' | sed 's/^/   - /'
        else
            echo "${models}" | jq -r '.[:3][]' | sed 's/^/   - /'
            echo "   - ... and $((model_count - 3)) more"
        fi
    else
        echo "⚠️  Available Models: None found"
    fi
    
    echo "=========================="
    
    return $status_code
}

opencode::status::check() {
    # Health check function for test::smoke
    local status_code=0
    
    # Check if VS Code is available
    if [[ -z "${VSCODE_COMMAND}" ]]; then
        log::error "Health check failed: VS Code not found"
        status_code=1
    fi
    
    # Check if extension is installed
    if ! opencode_is_installed; then
        log::error "Health check failed: Twinny extension not installed"
        status_code=1
    fi
    
    if [[ $status_code -eq 0 ]]; then
        log::success "OpenCode health check passed"
    else
        log::error "OpenCode health check failed"
    fi
    
    return $status_code
}