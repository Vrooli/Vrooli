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

# Get script directory
VAULT_ADAPTER_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ADAPTERS_DIR="$(dirname "$VAULT_ADAPTER_DIR")"

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
    
    # Set default vault URL if not configured
    export VAULT_URL="${VAULT_URL:-http://localhost:8200}"
    export VAULT_TIMEOUT="${VAULT_TIMEOUT:-30000}"
    
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
    
    local function_code='
    export default async ({ page }) => {
        const secretPath = "'$secret_path'";
        const kvPairs = '$kv_pairs';
        const vaultUrl = "'${VAULT_URL}'";
        
        try {
            // Navigate to Vault UI
            await page.goto(vaultUrl, {
                waitUntil: "networkidle2",
                timeout: 30000
            });
            
            // Login if needed (simplified example)
            const tokenInput = await page.$("input[data-test-token]");
            if (tokenInput && "'${VAULT_TOKEN}'") {
                await tokenInput.type("'${VAULT_TOKEN}'");
                await page.click("[data-test-auth-submit]");
                await page.waitForNavigation();
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