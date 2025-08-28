#!/bin/bash

# OpenCode Common Functions
set -euo pipefail

# Get directories  
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENCODE_LIB_DIR="${APP_ROOT}/resources/opencode/lib"
OPENCODE_DIR="${APP_ROOT}/resources/opencode"

# Configuration (use standard var_DATA_DIR which is already set by the main CLI)
OPENCODE_DATA_DIR="${var_DATA_DIR:-${APP_ROOT}/data}/opencode"
OPENCODE_CONFIG_FILE="${OPENCODE_DATA_DIR}/config.json"
OPENCODE_EXTENSION_ID="rjmacarthy.twinny"
OPENCODE_EXTENSION_NAME="Twinny - AI Code Completion"
OPENCODE_PORT=3355

# VS Code paths
VSCODE_EXTENSIONS_DIR="${HOME}/.vscode/extensions"
if command -v code &>/dev/null; then
    VSCODE_COMMAND="code"
elif command -v code-server &>/dev/null; then
    VSCODE_COMMAND="code-server"
else
    VSCODE_COMMAND=""
fi

# Helper functions
opencode_ensure_dirs() {
    mkdir -p "${OPENCODE_DATA_DIR}"
    mkdir -p "${OPENCODE_DATA_DIR}/logs"
    mkdir -p "${OPENCODE_DATA_DIR}/cache"
}

opencode_is_installed() {
    if [[ -z "${VSCODE_COMMAND}" ]]; then
        return 1
    fi
    
    # Check if extension is installed
    if ${VSCODE_COMMAND} --list-extensions 2>/dev/null | grep -q "${OPENCODE_EXTENSION_ID}"; then
        return 0
    fi
    
    return 1
}

opencode_get_version() {
    if opencode_is_installed; then
        ${VSCODE_COMMAND} --list-extensions --show-versions 2>/dev/null | \
            grep "${OPENCODE_EXTENSION_ID}" | \
            cut -d '@' -f2 || echo "unknown"
    else
        echo "not_installed"
    fi
}

opencode_get_models() {
    local model_source="${1:-both}"
    local models="[]"
    
    case "${model_source}" in
        ollama)
            if command -v ollama &>/dev/null; then
                models=$(ollama list 2>/dev/null | tail -n +2 | awk '{print $1}' | jq -R -s -c 'split("\n") | map(select(length > 0))')
            fi
            ;;
        openrouter)
            if [[ -f "${OPENCODE_CONFIG_FILE}" ]]; then
                local api_key=$(jq -r '.openrouter_api_key // ""' "${OPENCODE_CONFIG_FILE}")
                if [[ -n "${api_key}" && "${api_key}" != "null" ]]; then
                    models='["openrouter/gpt-4-turbo", "openrouter/claude-3-opus", "openrouter/mistral-large"]'
                fi
            fi
            ;;
        both)
            local ollama_models=$(opencode_get_models ollama)
            local openrouter_models=$(opencode_get_models openrouter)
            models=$(echo "${ollama_models} ${openrouter_models}" | jq -s 'add | unique')
            ;;
    esac
    
    echo "${models}"
}