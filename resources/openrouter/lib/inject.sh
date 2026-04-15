#!/bin/bash
# OpenRouter injection functionality

# Resolve local resource paths
SCRIPT_DIR="$(builtin cd "${BASH_SOURCE[0]%/*}" && builtin pwd)"
RESOURCE_DIR="$(builtin cd "${SCRIPT_DIR}/.." && builtin pwd)"
REPO_ROOT="$(builtin cd "${RESOURCE_DIR}/../.." && builtin pwd)"
OPENROUTER_INJECT_DIR="${RESOURCE_DIR}/lib"

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
            mkdir -p "${OPENROUTER_CONFIG_DIR}"
            local agent_config="${OPENROUTER_AGENT_CONFIG_FILE}"
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
