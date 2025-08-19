#!/bin/bash
# Gemini core functionality

# Get script directory
GEMINI_CORE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
GEMINI_RESOURCE_DIR="$(dirname "$GEMINI_CORE_DIR")"

# Source dependencies
source "${GEMINI_RESOURCE_DIR}/../../../lib/utils/var.sh"
source "${GEMINI_RESOURCE_DIR}/config/defaults.sh"
source "${GEMINI_RESOURCE_DIR}/../../../lib/utils/format.sh"
source "${GEMINI_RESOURCE_DIR}/../../../lib/utils/log.sh"
source "${GEMINI_RESOURCE_DIR}/../../lib/credentials-utils.sh"

# Initialize Gemini
gemini::init() {
    local verbose="${1:-false}"
    
    # Try to load API key from Vault first
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^vault$'; then
        local vault_key
        vault_key=$(docker exec vault sh -c "export VAULT_TOKEN=myroot && vault kv get -field=api_key secret/vrooli/gemini 2>/dev/null" || true)
        if [[ -n "$vault_key" && "$vault_key" != "No value found at secret/vrooli/gemini" ]]; then
            export GEMINI_API_KEY="$vault_key"
            [[ "$verbose" == "true" ]] && log::info "Gemini API key loaded from Vault"
        fi
    elif command -v vault >/dev/null 2>&1; then
        local vault_key
        vault_key=$(vault kv get -field=api_key secret/vrooli/gemini 2>/dev/null || true)
        if [[ -n "$vault_key" ]]; then
            export GEMINI_API_KEY="$vault_key"
            [[ "$verbose" == "true" ]] && log::info "Gemini API key loaded from Vault"
        fi
    fi
    
    # Try to load from credentials file if not found yet
    if [[ -z "$GEMINI_API_KEY" ]]; then
        local creds_file="${var_ROOT_DIR}/.vrooli/gemini-credentials.json"
        if [[ -f "$creds_file" ]]; then
            local file_key
            file_key=$(jq -r '.data.apiKey // empty' "$creds_file" 2>/dev/null || true)
            if [[ -n "$file_key" ]]; then
                export GEMINI_API_KEY="$file_key"
                [[ "$verbose" == "true" ]] && log::info "Gemini API key loaded from credentials file"
            fi
        fi
    fi
    
    # Fall back to environment variable
    if [[ -z "$GEMINI_API_KEY" ]]; then
        # Try to load from .env file
        if [[ -f "${var_ROOT_DIR}/.env" ]]; then
            source "${var_ROOT_DIR}/.env"
        fi
        
        if [[ -z "$GEMINI_API_KEY" ]]; then
            # Use placeholder for initial setup
            export GEMINI_API_KEY="placeholder-gemini-key"
            [[ "$verbose" == "true" ]] && log::warn "Gemini API key not found. Using placeholder"
            return 0
        fi
    fi
    
    return 0
}

# Test API connectivity
gemini::test_connection() {
    local timeout="${1:-$GEMINI_HEALTH_CHECK_TIMEOUT}"
    local model="${2:-$GEMINI_HEALTH_CHECK_MODEL}"
    
    if [[ -z "$GEMINI_API_KEY" ]] || [[ "$GEMINI_API_KEY" == "placeholder-gemini-key" ]]; then
        gemini::init || return 1
    fi
    
    # Skip test if using placeholder
    if [[ "$GEMINI_API_KEY" == "placeholder-gemini-key" ]]; then
        return 1
    fi
    
    local response
    response=$(timeout "$timeout" curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"contents\": [{\"parts\": [{\"text\": \"test\"}]}]}" \
        "${GEMINI_API_BASE}/models/${model}:generateContent?key=${GEMINI_API_KEY}" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        return 1
    fi
    
    # Check for error in response
    if echo "$response" | grep -q '"error"'; then
        return 1
    fi
    
    return 0
}

# List available models
gemini::list_models() {
    local timeout="${1:-$GEMINI_TIMEOUT}"
    
    if [[ -z "$GEMINI_API_KEY" ]] || [[ "$GEMINI_API_KEY" == "placeholder-gemini-key" ]]; then
        gemini::init || return 1
    fi
    
    if [[ "$GEMINI_API_KEY" == "placeholder-gemini-key" ]]; then
        echo "gemini-pro"
        echo "gemini-pro-vision"
        return 0
    fi
    
    timeout "$timeout" curl -s \
        "${GEMINI_API_BASE}/models?key=${GEMINI_API_KEY}" 2>/dev/null | \
        jq -r '.models[].name' 2>/dev/null | \
        sed 's|^models/||' || return 1
}

# Generate content
gemini::generate() {
    local prompt="$1"
    local model="${2:-$GEMINI_DEFAULT_MODEL}"
    local timeout="${3:-$GEMINI_TIMEOUT}"
    
    if [[ -z "$GEMINI_API_KEY" ]] || [[ "$GEMINI_API_KEY" == "placeholder-gemini-key" ]]; then
        gemini::init || return 1
    fi
    
    if [[ "$GEMINI_API_KEY" == "placeholder-gemini-key" ]]; then
        echo "Error: Real API key required for generation"
        return 1
    fi
    
    local response
    response=$(timeout "$timeout" curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{\"contents\": [{\"parts\": [{\"text\": \"$prompt\"}]}]}" \
        "${GEMINI_API_BASE}/models/${model}:generateContent?key=${GEMINI_API_KEY}" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        echo "Error: Request failed"
        return 1
    fi
    
    # Extract text from response
    echo "$response" | jq -r '.candidates[0].content.parts[0].text' 2>/dev/null || {
        echo "Error: Failed to parse response"
        return 1
    }
}

# Export functions
export -f gemini::init
export -f gemini::test_connection
export -f gemini::list_models
export -f gemini::generate