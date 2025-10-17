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

# Source Cloudflare AI Gateway integration
source "${OPENROUTER_RESOURCE_DIR}/lib/cloudflare.sh"

# Initialize OpenRouter
openrouter::init() {
    local verbose="${1:-false}"
    
    # Try to load API key following the secrets standard
    # Path as defined in secrets.yaml: secret/resources/openrouter/api/main
    
    # Try resource-vault command (preferred method)
    if command -v resource-vault >/dev/null 2>&1; then
        local vault_key
        # Try standard path first
        vault_key=$(resource-vault content get --path "resources/openrouter/api/main" --key "value" --format raw 2>/dev/null || true)
        
        # Filter out error messages (they start with ANSI codes or [ERROR])
        if [[ "$vault_key" == *"[ERROR]"* ]] || [[ "$vault_key" == *"[0;"* ]]; then
            vault_key=""
        fi
        
        # Fallback to legacy path if not found
        if [[ -z "$vault_key" || "$vault_key" == "No value found"* ]]; then
            vault_key=$(resource-vault content get --path "vrooli/openrouter" --key "api_key" --format raw 2>/dev/null || true)
            # Filter out error messages again
            if [[ "$vault_key" == *"[ERROR]"* ]] || [[ "$vault_key" == *"[0;"* ]]; then
                vault_key=""
            fi
        fi
        
        if [[ -n "$vault_key" && "$vault_key" != "No value found"* ]]; then
            export OPENROUTER_API_KEY="$vault_key"
            [[ "$verbose" == "true" ]] && log::info "OpenRouter API key loaded via resource-vault"
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
            # shellcheck disable=SC1091
            source "${var_ROOT_DIR}/.env"
        fi
        
        if [[ -z "$OPENROUTER_API_KEY" ]]; then
            [[ "$verbose" == "true" ]] && log::warn "OpenRouter API key not found. Please set OPENROUTER_API_KEY or store in Vault"
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
    
    # Use Cloudflare Gateway if configured
    local api_url
    api_url=$(openrouter::cloudflare::get_gateway_url "$OPENROUTER_API_BASE" "$model")
    
    local response
    response=$(timeout "$timeout" curl -s -X POST \
        -H "Authorization: Bearer $OPENROUTER_API_KEY" \
        -H "Content-Type: application/json" \
        -d '{"model": "'"$model"'", "messages": [{"role": "user", "content": "test"}], "max_tokens": 1}' \
        "${api_url}/chat/completions" 2>/dev/null)
    
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
    local timeout_value="${OPENROUTER_TIMEOUT:-30}"

    if [[ $# -gt 0 && "$1" != --* ]]; then
        timeout_value="$1"
        shift
    fi

    local output_format="text"
    local provider_filter=""
    local search_term=""
    local limit_value=""

    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                output_format="json"
                ;;
            --provider|-p)
                provider_filter="${2:-}"
                shift
                ;;
            --search|--contains)
                search_term="${2:-}"
                shift
                ;;
            --limit|-n)
                limit_value="${2:-}"
                shift
                ;;
            --help|-h)
                cat <<'EOF'
Usage: resource-openrouter content models [options]

Options:
  --json            Output structured JSON (includes metadata for each model)
  --provider <id>   Filter by provider prefix (e.g. openai, anthropic)
  --search <term>   Filter by substring in model id, name, or description
  --limit <n>       Limit the number of returned models
  -h, --help        Show this help message

Examples:
  resource-openrouter content models --limit 10
  resource-openrouter content models --provider openai --json
  resource-openrouter content models --search code
EOF
                return 0
                ;;
            *)
                log::error "Unknown option: $1"
                return 1
                ;;
        esac
        shift
    done

    if [[ -n "$limit_value" && ! "$limit_value" =~ ^[0-9]+$ ]]; then
        log::error "Invalid limit: ${limit_value}. Provide a positive integer."
        return 1
    fi

    local provider_prefix="${provider_filter}"
    if [[ -n "$provider_prefix" && "$provider_prefix" != */ ]]; then
        provider_prefix="${provider_prefix}/"
    fi

    local previous_timeout="${OPENROUTER_TIMEOUT:-}"
    OPENROUTER_TIMEOUT="${timeout_value}"

    local raw_catalog
    if ! raw_catalog=$(openrouter::models::fetch_catalog); then
        log::warn "Unable to fetch OpenRouter model catalog; returning default model list"
        raw_catalog='{"data": []}'
    fi

    OPENROUTER_TIMEOUT="${previous_timeout}"

    local limit_json="null"
    if [[ -n "$limit_value" ]]; then
        limit_json="$limit_value"
    fi

    local normalized
    if ! normalized=$(openrouter::models::filter_catalog "${raw_catalog}" "${provider_prefix}" "${search_term}" "${limit_json}"); then
        normalized="[]"
    fi

    if [[ "${output_format}" == "json" ]]; then
        openrouter::models::build_response "${normalized}"
        return $?
    fi

    if [[ "${normalized}" == "[]" ]]; then
        echo "${OPENROUTER_DEFAULT_MODEL:-openai/gpt-3.5-turbo}"
        return 0
    fi

    jq -r '.[].id' <<<"${normalized}"
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
