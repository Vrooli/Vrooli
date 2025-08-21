#!/bin/bash
# OpenRouter configuration functionality

# Get script directory
OPENROUTER_CONFIGURE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${OPENROUTER_CONFIGURE_DIR}/core.sh"
source "${OPENROUTER_CONFIGURE_DIR}/../../../../lib/utils/log.sh"

#######################################
# Configure OpenRouter API key
# Arguments:
#   --api-key: API key to set (optional, will prompt if not provided)
#   --vault: Store in Vault (default)
#   --file: Store in credentials file
# Returns:
#   0 on success, 1 on failure
#######################################
openrouter::configure() {
    local api_key=""
    local storage="vault"
    
    # Parse arguments
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --api-key)
                api_key="$2"
                shift 2
                ;;
            --vault)
                storage="vault"
                shift
                ;;
            --file)
                storage="file"
                shift
                ;;
            *)
                shift
                ;;
        esac
    done
    
    # Prompt for API key if not provided
    if [[ -z "$api_key" ]]; then
        echo "Configure OpenRouter API Access"
        echo "================================"
        echo ""
        echo "To get an API key:"
        echo "1. Visit https://openrouter.ai/keys"
        echo "2. Sign in or create an account"
        echo "3. Create a new API key"
        echo "4. Copy the key (starts with 'sk-or-')"
        echo ""
        read -p "Enter your OpenRouter API key: " -r api_key
        
        if [[ -z "$api_key" ]]; then
            log::error "API key cannot be empty"
            return 1
        fi
    fi
    
    # Validate API key format
    if [[ ! "$api_key" =~ ^sk-or- ]] && [[ "$api_key" != "sk-placeholder-key" ]]; then
        log::warn "API key should start with 'sk-or-' for OpenRouter"
        read -p "Continue anyway? (y/n): " -r confirm
        if [[ ! "$confirm" =~ ^[Yy] ]]; then
            return 1
        fi
    fi
    
    # Store the API key
    case "$storage" in
        vault)
            # Check if Vault is running
            if ! docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^vault$'; then
                log::error "Vault is not running. Please start Vault first or use --file option"
                return 1
            fi
            
            # Store in Vault
            if docker exec vault sh -c "export VAULT_TOKEN=myroot && vault kv put secret/vrooli/openrouter api_key='$api_key'" >/dev/null 2>&1; then
                log::success "API key stored in Vault successfully"
                
                # Test the connection
                export OPENROUTER_API_KEY="$api_key"
                echo ""
                echo "Testing connection..."
                if openrouter::test_connection; then
                    log::success "Successfully connected to OpenRouter API"
                    
                    # Show available models
                    echo ""
                    echo "Available models (showing first 5):"
                    openrouter::list_models 2>/dev/null | head -5
                else
                    log::warn "Could not verify API connection. Please check your API key"
                fi
            else
                log::error "Failed to store API key in Vault"
                return 1
            fi
            ;;
            
        file)
            # Store in credentials file
            local creds_dir="${var_ROOT_DIR}/data/credentials"
            local creds_file="${creds_dir}/openrouter-credentials.json"
            
            mkdir -p "$creds_dir"
            
            cat > "$creds_file" <<EOF
{
    "data": {
        "apiKey": "$api_key"
    },
    "metadata": {
        "created": "$(date -Iseconds)",
        "source": "cli-configure"
    }
}
EOF
            
            chmod 600 "$creds_file"
            log::success "API key stored in credentials file"
            
            # Test the connection
            export OPENROUTER_API_KEY="$api_key"
            echo ""
            echo "Testing connection..."
            if openrouter::test_connection; then
                log::success "Successfully connected to OpenRouter API"
                
                # Show available models
                echo ""
                echo "Available models (showing first 5):"
                openrouter::list_models 2>/dev/null | head -5
            else
                log::warn "Could not verify API connection. Please check your API key"
            fi
            ;;
            
        *)
            log::error "Invalid storage option: $storage"
            return 1
            ;;
    esac
    
    return 0
}

#######################################
# Show current configuration
# Returns:
#   Configuration details
#######################################
openrouter::show_config() {
    echo "OpenRouter Configuration"
    echo "========================"
    
    # Initialize to load API key
    openrouter::init >/dev/null 2>&1
    
    if [[ -z "$OPENROUTER_API_KEY" ]]; then
        echo "Status: Not configured"
        echo ""
        echo "Run 'resource-openrouter configure' to set up your API key"
    else
        if [[ "$OPENROUTER_API_KEY" == "sk-placeholder-key" ]]; then
            echo "Status: Placeholder key configured"
            echo "API Key: sk-placeholder-key (not functional)"
            echo ""
            echo "Run 'resource-openrouter configure' to set up a real API key"
        else
            echo "Status: Configured"
            echo "API Key: ${OPENROUTER_API_KEY:0:10}..." # Show first 10 chars only
            echo "API Base: ${OPENROUTER_API_BASE:-https://openrouter.ai/api/v1}"
            echo "Default Model: ${OPENROUTER_DEFAULT_MODEL:-openai/gpt-3.5-turbo}"
            
            # Check where it's stored
            if docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^vault$'; then
                local vault_key
                vault_key=$(docker exec vault sh -c "export VAULT_TOKEN=myroot && vault kv get -field=api_key secret/vrooli/openrouter 2>/dev/null" || true)
                if [[ "$vault_key" == "$OPENROUTER_API_KEY" ]]; then
                    echo "Storage: Vault"
                fi
            fi
            
            local creds_file="${var_ROOT_DIR}/data/credentials/openrouter-credentials.json"
            if [[ -f "$creds_file" ]]; then
                local file_key
                file_key=$(jq -r '.data.apiKey // empty' "$creds_file" 2>/dev/null || true)
                if [[ "$file_key" == "$OPENROUTER_API_KEY" ]]; then
                    echo "Storage: Credentials file"
                fi
            fi
            
            # Test connection
            echo ""
            echo "Testing connection..."
            if openrouter::test_connection; then
                log::success "API connection is working"
            else
                log::error "API connection failed"
            fi
        fi
    fi
}

# Export functions
export -f openrouter::configure
export -f openrouter::show_config