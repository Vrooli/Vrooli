#!/bin/bash

# Core helpers for the OpenCode AI CLI resource (official binary integration)
set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENCODE_DIR="${APP_ROOT}/resources/opencode"

# Load defaults
source "${OPENCODE_DIR}/config/defaults.sh"

# Data directories (var_DATA_DIR is exported by the global resource runtime)
OPENCODE_DATA_DIR="${var_DATA_DIR:-${APP_ROOT}/data}/opencode"
OPENCODE_BIN_DIR="${OPENCODE_DATA_DIR}/bin"
OPENCODE_BIN="${OPENCODE_BIN_DIR}/opencode"
OPENCODE_CACHE_DIR="${OPENCODE_DATA_DIR}/cache"
OPENCODE_LOG_DIR="${OPENCODE_DATA_DIR}/logs"
OPENCODE_VERSION_FILE="${OPENCODE_DATA_DIR}/VERSION"

# Config paths (we pin them via env vars before invoking the CLI)
OPENCODE_CONFIG_DIR="${OPENCODE_DATA_DIR}/config"
OPENCODE_CONFIG_FILE="${OPENCODE_CONFIG_DIR}/opencode.json"
OPENCODE_XDG_CONFIG_HOME="${OPENCODE_DATA_DIR}/xdg-config"
OPENCODE_XDG_DATA_HOME="${OPENCODE_DATA_DIR}/xdg-data"

# Secrets loading cache flag
OPENCODE_SECRETS_LOADED=${OPENCODE_SECRETS_LOADED:-0}

opencode::ensure_dirs() {
    mkdir -p "${OPENCODE_DATA_DIR}"
    mkdir -p "${OPENCODE_BIN_DIR}"
    mkdir -p "${OPENCODE_CACHE_DIR}"
    mkdir -p "${OPENCODE_LOG_DIR}"
    mkdir -p "${OPENCODE_CONFIG_DIR}"
    mkdir -p "${OPENCODE_XDG_CONFIG_HOME}"
    mkdir -p "${OPENCODE_XDG_CONFIG_HOME}/opencode"
    mkdir -p "${OPENCODE_XDG_DATA_HOME}"
    mkdir -p "${OPENCODE_XDG_DATA_HOME}/opencode"
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

opencode::supported_provider() {
    local provider="${1:-}"
    for candidate in "${OPENCODE_SUPPORTED_PROVIDERS[@]}"; do
        if [[ "${candidate}" == "${provider}" ]]; then
            return 0
        fi
    done
    return 1
}

opencode::ensure_cli() {
    if [[ ! -x "${OPENCODE_BIN}" ]]; then
        log::error "OpenCode CLI binary not found at ${OPENCODE_BIN}"
        log::info "Run 'resource-opencode manage install' to download the official CLI."
        return 1
    fi
    return 0
}

opencode::export_runtime_env() {
    export OPENCODE_CONFIG="${OPENCODE_CONFIG_FILE}"
    export XDG_CONFIG_HOME="${OPENCODE_XDG_CONFIG_HOME}"
    export XDG_DATA_HOME="${OPENCODE_XDG_DATA_HOME}"
    export XDG_CACHE_HOME="${OPENCODE_CACHE_DIR}"
}

opencode::default_config_payload() {
    local provider="${1:-${OPENCODE_DEFAULT_PROVIDER}}"
    local chat_model="${2:-${OPENCODE_DEFAULT_CHAT_MODEL}}"
    local completion_model="${3:-${OPENCODE_DEFAULT_COMPLETION_MODEL}}"

    cat <<EOF
{
  "\$schema": "https://opencode.ai/config.json",
  "model": "${provider}/${chat_model}",
  "small_model": "${provider}/${completion_model}",
  "instructions": [
    "AGENTS.md"
  ]
}
EOF
}

opencode::ensure_config() {
    if [[ -f "${OPENCODE_CONFIG_FILE}" ]]; then
        return 0
    fi
    log::info "Creating default OpenCode config at ${OPENCODE_CONFIG_FILE}"
    mkdir -p "${OPENCODE_CONFIG_DIR}"
    opencode::default_config_payload >"${OPENCODE_CONFIG_FILE}"
}

opencode::run_cli() {
    opencode::ensure_dirs
    opencode::ensure_cli || return 1
    opencode::ensure_config
    opencode::load_secrets || true
    opencode::export_runtime_env

    PATH="${OPENCODE_BIN_DIR}:${PATH}" \
    OPENCODE_LOG_DIR="${OPENCODE_LOG_DIR}" \
        "${OPENCODE_BIN}" "$@"
}

opencode::require_config() {
    if [[ ! -f "${OPENCODE_CONFIG_FILE}" ]]; then
        log::error "OpenCode configuration not found. Run 'resource-opencode manage install' first."
        return 1
    fi
    return 0
}
