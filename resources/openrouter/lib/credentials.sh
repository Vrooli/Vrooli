#!/bin/bash
# OpenRouter credentials display functionality

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENROUTER_LIB_DIR="${APP_ROOT}/resources/openrouter/lib"

# Source dependencies
source "${OPENROUTER_LIB_DIR}/core.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"

#######################################
# Display integration credentials
# Arguments:
#   --json: Output in JSON format
# Returns:
#   Integration credentials and instructions
#######################################
openrouter::credentials() {
    local json_output=false
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --json)
                json_output=true
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Initialize to load API key
    openrouter::init >/dev/null 2>&1
    
    if [[ "$json_output" == "true" ]]; then
        # JSON output
        cat <<EOF
{
    "service": "OpenRouter",
    "api_endpoint": "${OPENROUTER_API_BASE:-https://openrouter.ai/api/v1}",
    "authentication": {
        "type": "Bearer Token",
        "header": "Authorization: Bearer \$OPENROUTER_API_KEY",
        "key_prefix": "sk-or-"
    },
    "configuration": {
        "api_key_configured": $([ -n "$OPENROUTER_API_KEY" ] && echo "true" || echo "false"),
        "api_key_location": $([ -n "$OPENROUTER_API_KEY" ] && echo '"Environment/Vault/File"' || echo '"Not configured"'),
        "default_model": "${OPENROUTER_DEFAULT_MODEL:-openai/gpt-3.5-turbo}"
    },
    "documentation": {
        "api_docs": "https://openrouter.ai/docs",
        "models_list": "https://openrouter.ai/models",
        "get_api_key": "https://openrouter.ai/keys"
    }
}
EOF
    else
        # Human-readable output
        echo "OpenRouter Integration Credentials"
        echo "=================================="
        echo ""
        echo "API Endpoint: ${OPENROUTER_API_BASE:-https://openrouter.ai/api/v1}"
        echo ""
        echo "Authentication:"
        echo "  Type: Bearer Token"
        echo "  Header: Authorization: Bearer \$OPENROUTER_API_KEY"
        echo "  Key Prefix: sk-or-*"
        echo ""
        echo "Current Configuration:"
        if [[ -n "$OPENROUTER_API_KEY" ]]; then
            if [[ "$OPENROUTER_API_KEY" == "sk-placeholder-key" ]]; then
                echo "  Status: Placeholder key configured"
                echo "  API Key: sk-placeholder-key (not functional)"
            else
                echo "  Status: Configured"
                echo "  API Key: ${OPENROUTER_API_KEY:0:10}..."
            fi
            
            # Check storage location
            if docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^vault$'; then
                local vault_key
                vault_key=$(docker exec vault sh -c "export VAULT_TOKEN=myroot && vault kv get -field=api_key secret/vrooli/openrouter 2>/dev/null" || true)
                if [[ "$vault_key" == "$OPENROUTER_API_KEY" ]]; then
                    echo "  Storage: Vault"
                fi
            fi
            
            local creds_file="${var_ROOT_DIR}/data/credentials/openrouter-credentials.json"
            if [[ -f "$creds_file" ]]; then
                local file_key
                file_key=$(jq -r '.data.apiKey // empty' "$creds_file" 2>/dev/null || true)
                if [[ "$file_key" == "$OPENROUTER_API_KEY" ]]; then
                    echo "  Storage: Credentials file"
                fi
            fi
        else
            echo "  Status: Not configured"
            echo "  API Key: Not set"
        fi
        
        echo "  Default Model: ${OPENROUTER_DEFAULT_MODEL:-openai/gpt-3.5-turbo}"
        echo ""
        echo "How to Configure:"
        echo "  1. Get API key from: https://openrouter.ai/keys"
        echo "  2. Configure with: resource-openrouter configure --api-key <key>"
        echo ""
        echo "Example Usage:"
        echo '  curl -H "Authorization: Bearer $OPENROUTER_API_KEY" \'
        echo '       -H "Content-Type: application/json" \'
        echo '       -d '"'"'{"model":"openai/gpt-3.5-turbo","messages":[{"role":"user","content":"Hello"}]}'"'"' \'
        echo '       https://openrouter.ai/api/v1/chat/completions'
        echo ""
        echo "Documentation:"
        echo "  API Docs: https://openrouter.ai/docs"
        echo "  Models List: https://openrouter.ai/models"
    fi
}

# Export function
export -f openrouter::credentials