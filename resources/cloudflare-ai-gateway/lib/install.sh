#!/bin/bash
# Installation functions for Cloudflare AI Gateway

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
SCRIPT_DIR="${APP_ROOT}/resources/cloudflare-ai-gateway/lib"
RESOURCE_DIR="${APP_ROOT}/resources/cloudflare-ai-gateway"
RESOURCE_NAME="cloudflare-ai-gateway"

# Source utilities
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/lib/utils/install-resource-cli.sh"

# Install the resource
install_cloudflare_ai_gateway() {
    echo "Installing Cloudflare AI Gateway resource..."
    
    # Register CLI
    if ! install_resource_cli "${RESOURCE_NAME}" "${RESOURCE_DIR}/cli.sh"; then
        echo "Error: Failed to register CLI" >&2
        return 1
    fi
    
    # Initialize data directory
    local data_dir="${var_DATA_DIR}/resources/cloudflare-ai-gateway"
    mkdir -p "${data_dir}"
    mkdir -p "${data_dir}/configs"
    mkdir -p "${data_dir}/logs"
    
    # Check for Cloudflare credentials
    echo "Checking for Cloudflare credentials..."
    
    local has_creds=false
    
    # Check Vault first
    if command -v resource-vault &>/dev/null; then
        if resource-vault get cloudflare_account_id &>/dev/null 2>&1; then
            has_creds=true
            echo "✓ Found Cloudflare credentials in Vault"
        fi
    fi
    
    # Check environment variables
    if [[ "${has_creds}" == "false" ]]; then
        if [[ -n "${CLOUDFLARE_ACCOUNT_ID:-}" ]] && [[ -n "${CLOUDFLARE_API_TOKEN:-}" ]]; then
            has_creds=true
            echo "✓ Found Cloudflare credentials in environment"
        fi
    fi
    
    if [[ "${has_creds}" == "false" ]]; then
        echo ""
        echo "⚠️  Cloudflare credentials not found!"
        echo ""
        echo "To configure Cloudflare AI Gateway, you need:"
        echo "  1. A Cloudflare account (free tier works)"
        echo "  2. An API token with AI Gateway permissions"
        echo ""
        echo "You can set credentials using:"
        echo "  Option 1: Vault (recommended)"
        echo "    resource-vault set cloudflare_account_id YOUR_ACCOUNT_ID"
        echo "    resource-vault set cloudflare_api_token YOUR_API_TOKEN"
        echo ""
        echo "  Option 2: Environment variables"
        echo "    export CLOUDFLARE_ACCOUNT_ID=YOUR_ACCOUNT_ID"
        echo "    export CLOUDFLARE_API_TOKEN=YOUR_API_TOKEN"
        echo ""
        echo "Get your credentials at: https://dash.cloudflare.com/profile/api-tokens"
        echo ""
    fi
    
    echo "✓ Cloudflare AI Gateway resource installed successfully"
    echo ""
    echo "To get started:"
    echo "  ${RESOURCE_NAME} status    # Check status"
    echo "  ${RESOURCE_NAME} start     # Activate gateway"
    echo "  ${RESOURCE_NAME} help      # Show all commands"
    
    return 0
}

# Uninstall the resource
uninstall_cloudflare_ai_gateway() {
    echo "Uninstalling Cloudflare AI Gateway resource..."
    
    # Unregister CLI
    uninstall_resource_cli "${RESOURCE_NAME}"
    
    echo "✓ Cloudflare AI Gateway resource uninstalled"
    echo "Note: Data directory preserved at: ${var_DATA_DIR}/resources/cloudflare-ai-gateway"
    
    return 0
}