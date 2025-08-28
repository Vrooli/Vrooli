#!/bin/bash
# OpenRouter core functionality

# Define directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENROUTER_CORE_DIR="${APP_ROOT}/resources/openrouter/lib"
OPENROUTER_RESOURCE_DIR="${APP_ROOT}/resources/openrouter"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${OPENROUTER_RESOURCE_DIR}/config/defaults.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/resources/lib/credentials-utils.sh"

# Initialize OpenRouter
openrouter::init() {
    local verbose="${1:-false}"
    
    # Try to load API key from Vault first
    # Check if Vault is running via docker
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^vault$'; then
        local vault_key
        vault_key=$(docker exec vault sh -c "export VAULT_TOKEN=myroot && vault kv get -field=api_key secret/vrooli/openrouter 2>/dev/null" || true)
        if [[ -n "$vault_key" && "$vault_key" != "No value found at secret/vrooli/openrouter" ]]; then
            export OPENROUTER_API_KEY="$vault_key"
            [[ "$verbose" == "true" ]] && log::info "OpenRouter API key loaded from Vault"
        fi
    elif command -v vault >/dev/null 2>&1; then
        local vault_key
        vault_key=$(vault kv get -field=api_key secret/vrooli/openrouter 2>/dev/null || true)
        if [[ -n "$vault_key" ]]; then
            export OPENROUTER_API_KEY="$vault_key"
            [[ "$verbose" == "true" ]] && log::info "OpenRouter API key loaded from Vault"
        fi
    fi
    
    # Try to load from credentials file if not found yet
    if [[ -z "$OPENROUTER_API_KEY" ]]; then
        local creds_file="${var_ROOT_DIR}/data/credentials/openrouter-credentials.json"
        if [[ -f "$creds_file" ]]; then
            local file_key
            file_key=$(jq -r '.data.apiKey // empty' "$creds_file" 2>/dev/null || true)
            if [[ -n "$file_key" ]]; then
                export OPENROUTER_API_KEY="$file_key"
                [[ "$verbose" == "true" ]] && log::info "OpenRouter API key loaded from credentials file"
            fi
        fi
    fi
    
    # Fall back to environment variable
    if [[ -z "$OPENROUTER_API_KEY" ]]; then
        # Try to load from .env file
        if [[ -f "${var_ROOT_DIR}/.env" ]]; then
            source "${var_ROOT_DIR}/.env"
        fi
        
        if [[ -z "$OPENROUTER_API_KEY" ]]; then
            log::warn "OpenRouter API key not found. Please set OPENROUTER_API_KEY or store in Vault"
            return 1
        fi
    fi
    
    return 0
}

# Test API connectivity
openrouter::test_connection() {
    local timeout="${1:-$OPENROUTER_HEALTH_CHECK_TIMEOUT}"
    local model="${2:-$OPENROUTER_HEALTH_CHECK_MODEL}"
    
    if [[ -z "$OPENROUTER_API_KEY" ]]; then
        openrouter::init || return 1
    fi
    
    local response
    response=$(timeout "$timeout" curl -s -X POST \
        -H "Authorization: Bearer $OPENROUTER_API_KEY" \
        -H "Content-Type: application/json" \
        -d '{"model": "'"$model"'", "messages": [{"role": "user", "content": "test"}], "max_tokens": 1}' \
        "${OPENROUTER_API_BASE}/chat/completions" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    # Check for error in response
    if echo "$response" | grep -q '"error"'; then
        return 1
    fi
    
    return 0
}

# Get available models
openrouter::list_models() {
    local timeout="${1:-$OPENROUTER_TIMEOUT}"
    
    if [[ -z "$OPENROUTER_API_KEY" ]]; then
        openrouter::init || return 1
    fi
    
    timeout "$timeout" curl -s \
        -H "Authorization: Bearer $OPENROUTER_API_KEY" \
        "${OPENROUTER_API_BASE}/models" 2>/dev/null | \
        jq -r '.data[].id' 2>/dev/null || return 1
}

# Get usage/credits
openrouter::get_usage() {
    local timeout="${1:-$OPENROUTER_TIMEOUT}"
    
    if [[ -z "$OPENROUTER_API_KEY" ]]; then
        openrouter::init || return 1
    fi
    
    timeout "$timeout" curl -s \
        -H "Authorization: Bearer $OPENROUTER_API_KEY" \
        "https://openrouter.ai/api/v1/auth/key" 2>/dev/null
}

# Get API key (helper function for content scripts)
openrouter::get_api_key() {
    if [[ -z "$OPENROUTER_API_KEY" ]]; then
        openrouter::init >/dev/null 2>&1 || return 1
    fi
    echo "$OPENROUTER_API_KEY"
}