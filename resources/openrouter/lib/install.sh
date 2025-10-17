#!/bin/bash
# OpenRouter installation functionality

# Get script directory
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
OPENROUTER_INSTALL_DIR="${APP_ROOT}/resources/openrouter/lib"

# Source dependencies
source "${OPENROUTER_INSTALL_DIR}/core.sh"
source "${APP_ROOT}/scripts/resources/lib/credentials-utils.sh"

# Main install function
openrouter::install() {
    local verbose="${1:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Installing OpenRouter resource..."
    
    # OpenRouter is an API service, no local installation needed
    # Just ensure configuration is set up
    
    # Register CLI (v2.0 contract)
    local cli_installer="${APP_ROOT}/scripts/lib/resources/install-resource-cli.sh"
    if [[ -f "$cli_installer" ]]; then
        "$cli_installer" "${APP_ROOT}/resources/openrouter"
    fi
    
    # Try to get API key from user if not already configured
    if [[ -z "$OPENROUTER_API_KEY" ]]; then
        # Check Vault first
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
        
        # If still no key, inform user but don't fail (external API service)
        if [[ -z "$OPENROUTER_API_KEY" ]]; then
            log::warn "OpenRouter API key not found"
            log::info "Please obtain an API key from https://openrouter.ai/keys"
            log::info "Then either:"
            log::info "  1. Store in Vault: docker exec vault sh -c 'vault kv put secret/resources/openrouter/api/main value=<YOUR_KEY>'"
            log::info "  2. Set environment variable: export OPENROUTER_API_KEY=<YOUR_KEY>"
            log::info "  3. Add to .env file: OPENROUTER_API_KEY=<YOUR_KEY>"
            
            # Set placeholder key so install succeeds
            export OPENROUTER_API_KEY="sk-placeholder-key"
        fi
    fi
    
    # Check if we have a placeholder key
    if [[ "$OPENROUTER_API_KEY" == "sk-placeholder-key" ]]; then
        [[ "$verbose" == "true" ]] && log::info "OpenRouter configured with placeholder key"
        
        # Register the resource as configured (not fully running)
        local registry_file="${var_ROOT_DIR}/.vrooli/resource-registry/openrouter.json"
        mkdir -p "${registry_file%/*}"
        
        cat > "$registry_file" <<EOF
{
    "name": "openrouter",
    "category": "ai",
    "type": "api",
    "enabled": true,
    "status": "configured",
    "api_base": "$OPENROUTER_API_BASE",
    "note": "Placeholder key installed - replace with real API key for full functionality",
    "installed_at": "$(date -Iseconds)"
}
EOF
        
        # Also create/update the credentials file for consistency
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
        
        [[ "$verbose" == "true" ]] && log::warn "Replace placeholder key with real API key to enable full functionality"
        return 0
    fi
    
    # Test the connection for real keys
    if openrouter::test_connection; then
        [[ "$verbose" == "true" ]] && log::success "OpenRouter API is accessible"
        
        # Register the resource as running
        local registry_file="${var_ROOT_DIR}/.vrooli/resource-registry/openrouter.json"
        mkdir -p "${registry_file%/*}"
        
        cat > "$registry_file" <<EOF
{
    "name": "openrouter",
    "category": "ai",
    "type": "api",
    "enabled": true,
    "status": "running",
    "api_base": "$OPENROUTER_API_BASE",
    "installed_at": "$(date -Iseconds)"
}
EOF
        
        # Also create/update the credentials file
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
        
        return 0
    else
        log::error "Failed to connect to OpenRouter API"
        return 1
    fi
}

# Uninstall function
openrouter::uninstall() {
    local verbose="${1:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Uninstalling OpenRouter resource..."
    
    # Remove registry entry
    rm -f "${var_ROOT_DIR}/.vrooli/resource-registry/openrouter.json"
    
    # Note: We don't remove the API key from Vault as it might be used elsewhere
    
    [[ "$verbose" == "true" ]] && log::success "OpenRouter resource uninstalled"
    return 0
}

# Export functions
export -f openrouter::install
export -f openrouter::uninstall