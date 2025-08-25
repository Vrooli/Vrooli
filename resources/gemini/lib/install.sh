#!/bin/bash
# Gemini installation functionality

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
GEMINI_INSTALL_DIR="${APP_ROOT}/resources/gemini/lib"

# Source dependencies
source "${GEMINI_INSTALL_DIR}/core.sh"
source "${GEMINI_INSTALL_DIR}/../../../../lib/utils/log.sh"

# Install Gemini (configure API access)
gemini::install() {
    local verbose="${1:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Installing Gemini API configuration..."
    
    # Initialize configuration
    gemini::init "$verbose"
    
    # Create credentials file if needed
    local credentials_dir="${var_ROOT_DIR}/data/credentials"
    local credentials_file="${credentials_dir}/gemini-credentials.json"
    
    if [[ ! -d "$credentials_dir" ]]; then
        mkdir -p "$credentials_dir"
        [[ "$verbose" == "true" ]] && log::info "Created credentials directory"
    fi
    
    # Create or update credentials file
    if [[ ! -f "$credentials_file" ]]; then
        cat > "$credentials_file" <<EOF
{
    "type": "gemini",
    "name": "Google Gemini API",
    "data": {
        "apiKey": "${GEMINI_API_KEY:-placeholder-gemini-key}",
        "baseUrl": "${GEMINI_API_BASE}"
    }
}
EOF
        [[ "$verbose" == "true" ]] && log::info "Created Gemini credentials file"
    fi
    
    # Try to store in Vault if available
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^vault$'; then
        if [[ -n "$GEMINI_API_KEY" && "$GEMINI_API_KEY" != "placeholder-gemini-key" ]]; then
            docker exec vault sh -c "export VAULT_TOKEN=myroot && vault kv put secret/vrooli/gemini api_key='$GEMINI_API_KEY'" >/dev/null 2>&1
            [[ "$verbose" == "true" ]] && log::info "Stored Gemini API key in Vault"
        fi
    fi
    
    # Register CLI using standard helper
    "${var_SCRIPTS_RESOURCES_LIB_DIR}/resources/install-resource-cli.sh" "${APP_ROOT}/resources/gemini" 2>/dev/null || true
    
    # Verify installation
    if gemini::test_connection 2>/dev/null; then
        log::success "Gemini installed and API accessible"
    else
        if [[ "$GEMINI_API_KEY" == "placeholder-gemini-key" ]]; then
            log::warn "Gemini installed with placeholder key. Add real API key to enable functionality"
        else
            log::warn "Gemini installed but API not accessible. Check API key validity"
        fi
    fi
    
    return 0
}

# Uninstall Gemini
gemini::uninstall() {
    local verbose="${1:-false}"
    
    [[ "$verbose" == "true" ]] && log::info "Uninstalling Gemini configuration..."
    
    # Remove CLI symlink
    local cli_link="/usr/local/bin/resource-gemini"
    if [[ -L "$cli_link" ]] || [[ -f "$cli_link" ]]; then
        sudo rm -f "$cli_link"
        [[ "$verbose" == "true" ]] && log::info "Removed resource-gemini CLI"
    fi
    
    # Remove from Vault if present
    if docker ps --format '{{.Names}}' 2>/dev/null | grep -q '^vault$'; then
        docker exec vault sh -c "export VAULT_TOKEN=myroot && vault kv delete secret/vrooli/gemini" >/dev/null 2>&1
        [[ "$verbose" == "true" ]] && log::info "Removed Gemini credentials from Vault"
    fi
    
    log::success "Gemini configuration removed"
    return 0
}

# Export functions
export -f gemini::install
export -f gemini::uninstall