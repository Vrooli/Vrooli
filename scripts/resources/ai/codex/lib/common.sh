#\!/usr/bin/env bash
# Codex Common Functions

# Set script directory for sourcing
CODEX_COMMON_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# Source configuration
# shellcheck disable=SC1091
source "${CODEX_COMMON_DIR}/../config/defaults.sh"

# Source shared utilities
# shellcheck disable=SC1091
source "${CODEX_COMMON_DIR}/../../../../lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${CODEX_COMMON_DIR}/../../../../lib/utils/log.sh" 2>/dev/null || true

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
    # Try Vault if available
    elif command -v vault &>/dev/null; then
        api_key=$(vault kv get -field=api_key secret/openai 2>/dev/null || echo "")
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
