#!/bin/bash
# Gemini core functionality

# Define directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
GEMINI_CORE_DIR="${APP_ROOT}/resources/gemini/lib"
GEMINI_RESOURCE_DIR="${APP_ROOT}/resources/gemini"

# Source dependencies
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${GEMINI_RESOURCE_DIR}/config/defaults.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/resources/lib/credentials-utils.sh"
source "${GEMINI_RESOURCE_DIR}/lib/cache.sh"
source "${GEMINI_RESOURCE_DIR}/lib/tokens.sh"

# Initialize Gemini with proper Vault integration
gemini::init() {
    local verbose="${1:-false}"
    
    # Try resource-vault command (preferred method)
    if command -v resource-vault >/dev/null 2>&1; then
        local vault_key
        # Use the path defined in secrets.yaml
        vault_key=$(resource-vault content get --path "resources/gemini/api/key" --key "gemini_api_key" --format raw 2>/dev/null || true)
        if [[ -z "$vault_key" || "$vault_key" == "No value found"* ]]; then
            # Try legacy path for backwards compatibility
            vault_key=$(resource-vault content get --path "vrooli/gemini" --key "api_key" --format raw 2>/dev/null || true)
        fi
        
        if [[ -n "$vault_key" && "$vault_key" != "No value found"* ]]; then
            export GEMINI_API_KEY="$vault_key"
            [[ "$verbose" == "true" ]] && log::info "Gemini API key loaded via resource-vault"
        fi
    fi
    
    # Try to load from credentials file if not found yet
    if [[ -z "$GEMINI_API_KEY" ]]; then
        local creds_file="${var_ROOT_DIR}/data/credentials/gemini-credentials.json"
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
    
    # Load optional configuration from Vault if available
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^vault$'; then
        local model=$(docker exec vault sh -c "export VAULT_TOKEN=myroot && vault kv get -field=default_model secret/resources/gemini/config/model 2>/dev/null" || true)
        [[ -n "$model" && "$model" != "No value found"* ]] && export GEMINI_DEFAULT_MODEL="$model"
        
        local rpm=$(docker exec vault sh -c "export VAULT_TOKEN=myroot && vault kv get -field=rate_limit_rpm secret/resources/gemini/config/rate_limit_rpm 2>/dev/null" || true)
        [[ -n "$rpm" && "$rpm" != "No value found"* ]] && export GEMINI_RATE_LIMIT_RPM="$rpm"
        
        local tpm=$(docker exec vault sh -c "export VAULT_TOKEN=myroot && vault kv get -field=rate_limit_tpm secret/resources/gemini/config/rate_limit_tpm 2>/dev/null" || true)
        [[ -n "$tpm" && "$tpm" != "No value found"* ]] && export GEMINI_RATE_LIMIT_TPM="$tpm"
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

# Generate content with caching support
gemini::generate() {
    local prompt="$1"
    local model="${2:-$GEMINI_DEFAULT_MODEL}"
    local timeout="${3:-$GEMINI_TIMEOUT}"
    local temperature="${4:-0.7}"
    local use_cache="${5:-${GEMINI_CACHE_ENABLED:-true}}"
    
    if [[ -z "$GEMINI_API_KEY" ]] || [[ "$GEMINI_API_KEY" == "placeholder-gemini-key" ]]; then
        gemini::init || return 1
    fi
    
    if [[ "$GEMINI_API_KEY" == "placeholder-gemini-key" ]]; then
        echo "Error: Real API key required for generation"
        return 1
    fi
    
    # Check cache first if enabled
    local cache_key
    if [[ "$use_cache" == "true" ]]; then
        cache_key=$(gemini::cache::generate_key "$prompt" "$model" "$temperature")
        local cached_response
        cached_response=$(gemini::cache::get "$cache_key" 2>/dev/null)
        if [[ -n "$cached_response" ]]; then
            log::debug "Cache hit for prompt"
            echo "$cached_response"
            return 0
        fi
        log::debug "Cache miss for prompt"
    fi
    
    local response
    response=$(timeout "$timeout" curl -s -X POST \
        -H "Content-Type: application/json" \
        -d "{
            \"contents\": [{\"parts\": [{\"text\": \"$prompt\"}]}],
            \"generationConfig\": {
                \"temperature\": $temperature
            }
        }" \
        "${GEMINI_API_BASE}/models/${model}:generateContent?key=${GEMINI_API_KEY}" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        echo "Error: Request failed"
        return 1
    fi
    
    # Extract text from response
    local result
    result=$(echo "$response" | jq -r '.candidates[0].content.parts[0].text' 2>/dev/null)
    if [[ -z "$result" || "$result" == "null" ]]; then
        echo "Error: Failed to parse response"
        return 1
    fi
    
    # Cache the result if caching is enabled
    if [[ "$use_cache" == "true" && -n "$cache_key" ]]; then
        gemini::cache::set "$cache_key" "$result" "$GEMINI_CACHE_TTL" 2>/dev/null || true
        log::debug "Response cached"
    fi
    
    # Log token usage if tracking is enabled
    if [[ "$GEMINI_TOKEN_TRACKING_ENABLED" == "true" ]]; then
        gemini::tokens::log_usage "$prompt" "$result" "$model" 2>/dev/null || true
    fi
    
    echo "$result"
    return 0
}

# Export functions
export -f gemini::init
export -f gemini::test_connection
export -f gemini::list_models
export -f gemini::generate