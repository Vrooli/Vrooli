#!/bin/bash
# Installation functions for Cloudflare AI Gateway

set -euo pipefail

VROOLI_ROOT="${VROOLI_ROOT:-${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}}"
SCRIPT_DIR="${VROOLI_ROOT}/resources/cloudflare-ai-gateway/lib"
RESOURCE_DIR="${VROOLI_ROOT}/resources/cloudflare-ai-gateway"
RESOURCE_NAME="cloudflare-ai-gateway"

# Source utilities
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh"
# Install the resource
install_cloudflare_ai_gateway() {
    echo "Installing Cloudflare AI Gateway resource..."
    
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
        if resource-vault content get --path "resources/cloudflare/account_id" --format raw &>/dev/null 2>&1; then
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
        echo "    resource-vault content add --path resources/cloudflare/account_id --value YOUR_ACCOUNT_ID"
        echo "    resource-vault content add --path resources/cloudflare/api_token --value YOUR_API_TOKEN"
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
    
    echo "✓ Cloudflare AI Gateway resource uninstalled"
    echo "Note: Data directory preserved at: ${var_DATA_DIR}/resources/cloudflare-ai-gateway"
    
    return 0
}
