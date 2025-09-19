#!/bin/bash

# Core helpers for the OpenCode AI CLI resource
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENCODE_DIR="${APP_ROOT}/resources/opencode"

# Load defaults
source "${OPENCODE_DIR}/config/defaults.sh"

# Data directories (var_DATA_DIR is exported by the global resource runtime)
OPENCODE_DATA_DIR="${var_DATA_DIR:-${APP_ROOT}/data}/opencode"
OPENCODE_CONFIG_FILE="${OPENCODE_DATA_DIR}/config.json"
OPENCODE_LOG_DIR="${OPENCODE_DATA_DIR}/logs"

# Secrets loading cache flag
OPENCODE_SECRETS_LOADED=${OPENCODE_SECRETS_LOADED:-0}

opencode::ensure_dirs() {
    mkdir -p "${OPENCODE_DATA_DIR}"
    mkdir -p "${OPENCODE_LOG_DIR}"
    mkdir -p "${OPENCODE_DATA_DIR}/cache"
}

opencode::load_secrets() {
    if [[ "${OPENCODE_SECRETS_LOADED}" -eq 1 ]]; then
        return 0
    fi

    if command -v resource-vault &>/dev/null; then
        local export_cmd
        if export_cmd=$(resource-vault secrets export opencode 2>/dev/null); then
            if [[ -n "${export_cmd}" ]]; then
                eval "${export_cmd}"
            fi
        fi
    fi

    if command -v jq &>/dev/null; then
        local vrooli_root="${VROOLI_ROOT:-"$HOME/Vrooli"}"
        local secrets_file="${vrooli_root}/.vrooli/secrets.json"
        if [[ -f "${secrets_file}" ]]; then
            local vars=(
                OPENROUTER_API_KEY
                CLOUDFLARE_API_TOKEN
                CLOUDFLARE_ACCOUNT_ID
                CLOUDFLARE_AI_GATEWAY_SLUG
            )
            for var_name in "${vars[@]}"; do
                if [[ -z "${!var_name:-}" ]]; then
                    local value
                    value=$(jq -r --arg key "${var_name}" '.[$key] // empty' "${secrets_file}" 2>/dev/null)
                    if [[ -n "${value}" && "${value}" != "null" ]]; then
                        export "${var_name}"="${value}"
                    fi
                fi
            done
        fi
    fi

    OPENCODE_SECRETS_LOADED=1
    return 0
}

opencode::python_bin() {
    if command -v python3 &>/dev/null; then
        echo "python3"
    elif command -v python &>/dev/null; then
        echo "python"
    else
        echo "" 
    fi
}

opencode::ensure_python() {
    local py
    py=$(opencode::python_bin)
    if [[ -z "${py}" ]]; then
        log::error "Python 3 is required for OpenCode CLI. Install python3 and retry."
        return 1
    fi
    return 0
}

opencode::supported_provider() {
    local provider="${1:-}"
    for candidate in "${OPENCODE_SUPPORTED_PROVIDERS[@]}"; do
        if [[ "${candidate}" == "${provider}" ]]; then
            return 0
        fi
    done
    return 1
}

opencode::read_config() {
    local key="${1:-}"
    if [[ ! -f "${OPENCODE_CONFIG_FILE}" ]]; then
        echo ""
        return 0
    fi
    if command -v jq &>/dev/null; then
        jq -r --arg key "${key}" '.[$key] // ""' "${OPENCODE_CONFIG_FILE}" 2>/dev/null
    else
        # Fallback: simple grep (best effort)
        grep -E "\"${key}\"" "${OPENCODE_CONFIG_FILE}" | head -n1 | sed -E 's/.*: \"(.*)\".*/\1/'
    fi
}

opencode::run_cli() {
    opencode::ensure_python || return 1
    opencode::load_secrets || true

    local py
    py=$(opencode::python_bin)
    "${py}" "${OPENCODE_CLI_ENTRYPOINT}" --config "${OPENCODE_CONFIG_FILE}" "$@"
}

opencode::models_json() {
    opencode::run_cli models list --json
}

opencode::require_config() {
    if [[ ! -f "${OPENCODE_CONFIG_FILE}" ]]; then
        log::error "OpenCode configuration not found. Run 'resource-opencode manage install' first."
        return 1
    fi
    return 0
}
