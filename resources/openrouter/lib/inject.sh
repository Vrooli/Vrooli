#!/bin/bash
# OpenRouter injection functionality

# Define directory using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENROUTER_INJECT_DIR="${APP_ROOT}/resources/openrouter/lib"

# Source dependencies
source "${OPENROUTER_INJECT_DIR}/core.sh"

# Inject OpenRouter configuration into other resources
openrouter::inject() {
    local target="${1:-}"
    local data="${2:-}"
    local verbose="${3:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Injecting OpenRouter configuration..."
    
    # Initialize to get API key
    openrouter::init || return 1
    
    case "$target" in
        "n8n"|"windmill"|"huginn")
            # Create credentials JSON for workflow engines
            local creds_file="${var_ROOT_DIR}/data/credentials/openrouter-credentials.json"
            cat > "$creds_file" <<EOF
{
    "type": "openrouter",
    "name": "OpenRouter API",
    "data": {
        "apiKey": "$OPENROUTER_API_KEY",
        "baseUrl": "$OPENROUTER_API_BASE"
    }
}
EOF
            [[ "$verbose" == "true" ]] && log::success "OpenRouter credentials injected for $target"
            ;;
            
        "env")
            # Export to environment file
            local env_file="${var_ROOT_DIR}/.env"
            if ! grep -q "OPENROUTER_API_KEY" "$env_file" 2>/dev/null; then
                echo "OPENROUTER_API_KEY=$OPENROUTER_API_KEY" >> "$env_file"
                echo "OPENROUTER_API_BASE=$OPENROUTER_API_BASE" >> "$env_file"
                [[ "$verbose" == "true" ]] && log::success "OpenRouter configuration added to .env"
            fi
            ;;
            
        "claude-code"|"cline"|"agent-s2")
            # Create config for AI agents
            local agent_config="${var_ROOT_DIR}/.vrooli/openrouter-agent-config.json"
            cat > "$agent_config" <<EOF
{
    "provider": "openrouter",
    "apiKey": "$OPENROUTER_API_KEY",
    "baseUrl": "$OPENROUTER_API_BASE",
    "models": [
        "openai/gpt-4",
        "openai/gpt-3.5-turbo",
        "anthropic/claude-3-opus",
        "anthropic/claude-3-sonnet",
        "meta-llama/llama-3-70b-instruct"
    ]
}
EOF
            [[ "$verbose" == "true" ]] && log::success "OpenRouter configuration injected for $target"
            ;;
            
        *)
            log::warn "Unknown injection target: $target"
            return 1
            ;;
    esac
    
    return 0
}

# Export function
export -f openrouter::inject