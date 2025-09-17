#!/usr/bin/env bash
# Codex Common Functions

# Set script directory for sourcing
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
CODEX_COMMON_DIR="${APP_ROOT}/resources/codex/lib"

# Source configuration
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/codex/config/defaults.sh"

# Source shared utilities
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/log.sh"

#######################################
# Determine writable Codex home directory
# Returns:
#   Path to writable Codex home directory
#######################################
codex::ensure_home() {
    if [[ -n "${CODEX_EFFECTIVE_HOME:-}" && -d "${CODEX_EFFECTIVE_HOME}" ]]; then
        echo "${CODEX_EFFECTIVE_HOME}"
        return 0
    fi

    local preferred_home="${CODEX_HOME:-${HOME}/.codex}"
    local resolved_home=""

    if mkdir -p "${preferred_home}" 2>/dev/null; then
        local test_file="${preferred_home}/.codex-write-test"
        if touch "${test_file}" 2>/dev/null; then
            rm -f "${test_file}"
            resolved_home="${preferred_home}"
            CODEX_HOME_OVERRIDE_REQUIRED="false"
        else
            resolved_home=""
        fi
    fi

    if [[ -z "${resolved_home}" ]]; then
        local workspace_base="${CODEX_WORKSPACE:-/tmp/codex-workspace}"
        local fallback_home="${workspace_base}/.codex-home"
        if mkdir -p "${fallback_home}" 2>/dev/null; then
            resolved_home="${fallback_home}"
            log::warn "Codex home ${preferred_home} is not writable. Using fallback: ${resolved_home}"
        else
            resolved_home="/tmp/codex-home"
            mkdir -p "${resolved_home}" 2>/dev/null || true
            log::warn "Codex home not writable. Falling back to ${resolved_home}"
        fi
        CODEX_HOME_OVERRIDE_REQUIRED="true"
    fi

    mkdir -p "${resolved_home}/sessions" "${resolved_home}/log" "${resolved_home}/logs" \
        "${resolved_home}/scripts" "${resolved_home}/outputs" 2>/dev/null || true

    export CODEX_HOME="${resolved_home}"
    CODEX_EFFECTIVE_HOME="${resolved_home}"

    echo "${resolved_home}"
}

#######################################
# Get OpenAI API key from Vault or environment
# Returns:
#   API key string or empty if not found
#######################################
codex::get_api_key() {
    local api_key=""
    
    # Try environment variable first
    if [[ -n "${OPENAI_API_KEY:-}" ]]; then
        api_key="${OPENAI_API_KEY}"
    # Try Vault using resource-vault command
    elif command -v resource-vault &>/dev/null; then
        api_key=$(resource-vault content get --path "resources/codex/api/openai" --key "api_key" --format raw 2>/dev/null || echo "")
    # Try credentials file
    elif [[ -f "${HOME}/.openai/credentials" ]]; then
        api_key=$(grep "api_key=" "${HOME}/.openai/credentials" | cut -d= -f2)
    fi
    
    echo "${api_key}"
}

#######################################
# Check if Codex is configured
# Returns:
#   0 if configured, 1 otherwise
#######################################
codex::is_configured() {
    local api_key
    api_key=$(codex::get_api_key)
    
    if [[ -n "${api_key}" ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Check if Codex service is available
# Returns:
#   0 if available, 1 otherwise
#######################################
codex::is_available() {
    local api_key
    api_key=$(codex::get_api_key)
    
    if [[ -z "${api_key}" ]]; then
        return 1
    fi
    
    # Test API connection with a simple models request
    local response
    response=$(curl -s -o /dev/null -w "%{http_code}" \
        -H "Authorization: Bearer ${api_key}" \
        "${CODEX_API_ENDPOINT}/models" 2>/dev/null)
    
    if [[ "${response}" == "200" ]]; then
        return 0
    else
        return 1
    fi
}

#######################################
# Get Codex version/model info
# Returns:
#   Model name or "unknown"
#######################################
codex::get_version() {
    echo "${CODEX_DEFAULT_MODEL}"
}

#######################################
# Save status to file
# Arguments:
#   status - Status to save (running/stopped)
#######################################
codex::save_status() {
    local status="${1:-stopped}"
    echo "${status}" > "${CODEX_STATUS_FILE}"
}
