#!/usr/bin/env bash

#######################################
# Vault Adapter for Browserless
# Description: UI automation adapter for HashiCorp Vault
#
# Provides browser-based fallback interfaces for Vault operations
# when the API is unavailable or when dealing with UI-only features.
#
# This adapter enables:
#   - Secret management via UI
#   - Policy configuration through browser
#   - Auth method setup via UI
#   - Audit log viewing
#######################################

# Define adapter directories using cached APP_ROOT
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../../.." && builtin pwd)}"
VAULT_ADAPTER_DIR="${APP_ROOT}/resources/browserless/adapters/vault"
ADAPTERS_DIR="${APP_ROOT}/resources/browserless/adapters"

# Source adapter framework
source "${ADAPTERS_DIR}/common.sh"

# Export adapter name for context
export BROWSERLESS_ADAPTER_NAME="vault"

#######################################
# Initialize vault adapter
# Sets up vault-specific configuration
# Returns:
#   0 on success, 1 on failure
#######################################
vault::init() {
    # Initialize base adapter framework
    adapter::init "vault"
    
    # Load vault configuration if available
    adapter::load_target_config "vault"
    
    # Get Vault credentials from Vrooli's unified secrets system
    local vault_token="${VAULT_TOKEN:-}"
    local vault_url="${VAULT_URL:-}"
    
    # Try to get credentials from Vrooli secrets system
    if command -v secrets::resolve >/dev/null 2>&1; then
        if [[ -z "$vault_token" ]]; then
            vault_token=$(secrets::resolve "VAULT_TOKEN" 2>/dev/null || echo "")
        fi
        if [[ -z "$vault_url" ]]; then
            vault_url=$(secrets::resolve "VAULT_URL" 2>/dev/null || echo "")
        fi
    fi
    
    # Build vault URL using port registry for local, or use provided URL for remote
    if [[ -z "$vault_url" ]]; then
        # Get vault port from registry
        local vault_port
        if command -v resources::get_default_port >/dev/null 2>&1; then
            vault_port=$(resources::get_default_port "vault" 2>/dev/null || echo "8200")
        else
            vault_port="8200"
        fi
        vault_url="http://localhost:${vault_port}"
    fi
    
    export VAULT_URL="$vault_url"
    export VAULT_TOKEN="${vault_token:-}"
    export VAULT_TIMEOUT="${VAULT_TIMEOUT:-30000}"
    
    # Log credential status
    if [[ -n "$vault_token" ]]; then
        log::info "🔐 Found Vault credentials from Vrooli secrets system"
        log::info "🔐 Token: [${#vault_token} characters] | URL: $VAULT_URL"
    else
        log::info "ℹ️  No Vault credentials found - will attempt without authentication"
        log::info "    (This may work for development Vault instances)"
    fi
    
    log::debug "Vault adapter initialized with URL: $VAULT_URL"
    return 0
}

#######################################
# List available vault adapter commands
# Used by the adapter framework for discovery
# Returns:
#   List of available commands
#######################################
adapter::list_commands() {
    echo "add-secret      - Add secret via UI automation"
    echo "list-secrets    - List secrets via UI scraping"
    echo "configure-auth  - Configure auth method via UI"
    echo "view-policies   - View policies via browser"
}

#######################################
# Main command dispatcher for vault adapter
# Routes commands to appropriate implementations
# Arguments:
#   $1 - Command name
#   $@ - Command arguments
# Returns:
#   Command exit status
#######################################
vault::dispatch() {
    local command="${1:-}"
    shift || true
    
    # Initialize adapter
    vault::init
    
    # Check browserless health
    if ! adapter::check_browserless_health; then
        return 1
    fi
    
    case "$command" in
        add-secret)
            vault::add_secret "$@"
            ;;
        list-secrets)
            vault::list_secrets "$@"
            ;;
        configure-auth)
            vault::configure_auth "$@"
            ;;
        view-policies)
            vault::view_policies "$@"
            ;;
        help|--help|-h|"")
            vault::show_help
            ;;
        *)
            log::error "Unknown vault adapter command: $command"
            vault::show_help
            return 1
            ;;
    esac
}

#######################################
# Show help for vault adapter
# Displays usage information and examples
#######################################
vault::show_help() {
    cat <<EOF
Vault Adapter for Browserless

Usage:
  resource-browserless for vault <command> [options]

Commands:
  add-secret <path> <key>=<value>  Add secret to Vault
  list-secrets <path>              List secrets at path
  configure-auth <method>          Configure auth method
  view-policies                    View all policies

Examples:
  # Add a secret
  resource-browserless for vault add-secret secret/myapp username=admin password=secret

  # List secrets
  resource-browserless for vault list-secrets secret/

  # Configure GitHub auth
  resource-browserless for vault configure-auth github

Environment Variables:
  VAULT_URL     - Vault instance URL (default: http://localhost:8200)
  VAULT_TOKEN   - Root token for authentication
  VAULT_TIMEOUT - Operation timeout in ms (default: 30000)

Notes:
  - This adapter provides UI-based fallback when Vault API is unavailable
  - Requires browserless to be running and healthy
  - Authentication token should be set via environment variable

EOF
    return 0
}

#######################################
# Add secret via UI automation
# Arguments:
#   $1 - Secret path
#   $2+ - Key=value pairs
# Returns:
#   0 on success, 1 on failure
#######################################
vault::add_secret() {
    local secret_path="${1:-}"
    shift || true
    
    if [[ -z "$secret_path" ]]; then
        log::error "Secret path required"
        echo "Usage: resource-browserless for vault add-secret <path> key=value [key2=value2 ...]"
        return 1
    fi
    
    log::header "🔐 Adding Secret to Vault via UI: $secret_path"
    
    # Parse key-value pairs
    local kv_pairs="{}"
    for arg in "$@"; do
        if [[ "$arg" == *"="* ]]; then
            local key="${arg%%=*}"
            local value="${arg#*=}"
            kv_pairs=$(echo "$kv_pairs" | jq --arg k "$key" --arg v "$value" '. + {($k): $v}')
        fi
    done
    
    # Inject credentials into browserless function (using placeholder pattern)
    local function_code='
    export default async ({ page }) => {
        const secretPath = "'$secret_path'";
        const kvPairs = '$kv_pairs';
        const vaultUrl = "%VAULT_URL%";
        const vaultToken = "%VAULT_TOKEN%";
        
        try {
            // Navigate to Vault UI
            await page.goto(vaultUrl, {
                waitUntil: "networkidle2",
                timeout: 30000
            });
            
            // Login if needed (check for token authentication)
            if (vaultToken && vaultToken !== "%VAULT_TOKEN_PLACEHOLDER%") {
                const tokenInput = await page.$("input[data-test-token]");
                if (tokenInput) {
                    await tokenInput.type(vaultToken);
                    await page.click("[data-test-auth-submit]");
                    await page.waitForNavigation();
                }
            }
            
            // Navigate to secrets engine
            await page.goto(`${vaultUrl}/ui/vault/secrets`, {
                waitUntil: "networkidle2"
            });
            
            // Click create secret
            await page.click("[data-test-secret-create]");
            
            // Fill in path
            await page.type("[data-test-secret-path]", secretPath);
            
            // Add key-value pairs
            for (const [key, value] of Object.entries(kvPairs)) {
                await page.click("[data-test-secret-add-row]");
                const keyInputs = await page.$$("[data-test-secret-key]");
                const valueInputs = await page.$$("[data-test-secret-value]");
                
                if (keyInputs.length > 0 && valueInputs.length > 0) {
                    await keyInputs[keyInputs.length - 1].type(key);
                    await valueInputs[valueInputs.length - 1].type(value);
                }
            }
            
            // Save secret
            await page.click("[data-test-secret-save]");
            await page.waitForTimeout(2000);
            
            return {
                success: true,
                path: secretPath,
                keys: Object.keys(kvPairs),
                timestamp: new Date().toISOString()
            };
            
        } catch (error) {
            return {
                success: false,
                error: error.message,
                timestamp: new Date().toISOString()
            };
        }
    }'
    
    # Inject credentials into browserless function
    function_code=$(echo "$function_code" | sed "s|%VAULT_URL%|$VAULT_URL|g")
    function_code=$(echo "$function_code" | sed "s|%VAULT_TOKEN%|$VAULT_TOKEN|g")
    
    # Execute via adapter framework
    adapter::execute_browser_function "$function_code" "$VAULT_TIMEOUT" "true" "vault_add_secret"
}

# Placeholder functions for future implementation
vault::list_secrets() {
    log::warn "List secrets functionality coming soon"
    log::info "This will scrape the Vault UI to list secrets when the API is unavailable"
    return 1
}

vault::configure_auth() {
    log::warn "Configure auth functionality coming soon"
    log::info "This will configure auth methods via the Vault UI"
    return 1
}

vault::view_policies() {
    log::warn "View policies functionality coming soon"
    log::info "This will scrape policies from the Vault UI"
    return 1
}

# Export main dispatcher for CLI integration
export -f vault::dispatch
export -f vault::init