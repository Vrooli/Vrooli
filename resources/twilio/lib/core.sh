#!/usr/bin/env bash
################################################################################
# Twilio Core Library - v2.0 Universal Contract Implementation
# 
# Core functionality for Twilio cloud communications platform
################################################################################

set -euo pipefail

# Get the directory of this script
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
TWILIO_DIR="${APP_ROOT}/resources/twilio"
TWILIO_LIB_DIR="${TWILIO_DIR}/lib"

# Source utilities and common functions
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${APP_ROOT}/scripts/lib/utils/format.sh"
source "${APP_ROOT}/scripts/lib/utils/log.sh"
source "${APP_ROOT}/scripts/resources/port_registry.sh"
source "${TWILIO_LIB_DIR}/common.sh"

################################################################################
# Core Functions Required by v2.0 Contract
################################################################################

# Initialize Twilio resource
twilio::core::init() {
    log::debug "Initializing Twilio resource..."
    
    # Ensure required directories exist
    twilio::ensure_dirs
    
    # Load secrets if Vault is available
    twilio::core::load_secrets
    
    return 0
}

# Load secrets from Vault or environment
twilio::core::load_secrets() {
    local vault_available=false
    
    # Check if Vault resource is available
    if command -v resource-vault &>/dev/null && resource-vault status &>/dev/null; then
        vault_available=true
        log::debug "Vault available, loading secrets..."
        
        # Try to load secrets from Vault
        if resource-vault secrets check twilio &>/dev/null; then
            # Export secrets as environment variables
            eval "$(resource-vault secrets export twilio 2>/dev/null || true)"
            log::debug "Secrets loaded from Vault"
        else
            log::warn "Twilio secrets not configured in Vault"
        fi
    fi
    
    # Fall back to environment variables or credentials file
    if [[ -z "${TWILIO_ACCOUNT_SID:-}" ]] || [[ -z "${TWILIO_AUTH_TOKEN:-}" ]]; then
        # Try loading from credentials file
        if [[ -f "$TWILIO_CREDENTIALS_FILE" ]]; then
            export TWILIO_ACCOUNT_SID=$(jq -r '.account_sid // empty' "$TWILIO_CREDENTIALS_FILE" 2>/dev/null || echo "")
            export TWILIO_AUTH_TOKEN=$(jq -r '.auth_token // empty' "$TWILIO_CREDENTIALS_FILE" 2>/dev/null || echo "")
            
            if [[ -n "$TWILIO_ACCOUNT_SID" ]] && [[ -n "$TWILIO_AUTH_TOKEN" ]]; then
                log::debug "Credentials loaded from file"
            fi
        fi
    fi
    
    # Check if we have valid credentials
    if [[ -z "${TWILIO_ACCOUNT_SID:-}" ]] || [[ -z "${TWILIO_AUTH_TOKEN:-}" ]]; then
        log::debug "No Twilio credentials available"
        return 1
    fi
    
    return 0
}

# Validate Twilio configuration
twilio::core::validate() {
    local errors=0
    
    # Check if Twilio CLI is installed
    if ! twilio::is_installed; then
        log::error "Twilio CLI is not installed"
        ((errors++))
    fi
    
    # Check if credentials are configured
    if ! twilio::core::has_valid_credentials; then
        log::error "Twilio credentials are not configured or invalid"
        ((errors++))
    fi
    
    # Check if we can connect to Twilio API
    if [[ $errors -eq 0 ]]; then
        if ! twilio::core::test_api_connection; then
            log::error "Cannot connect to Twilio API"
            ((errors++))
        fi
    fi
    
    return $errors
}

# Check if credentials are valid
twilio::core::has_valid_credentials() {
    # Load secrets first
    twilio::core::load_secrets || true
    
    # Check environment variables
    if [[ -n "${TWILIO_ACCOUNT_SID:-}" ]] && [[ -n "${TWILIO_AUTH_TOKEN:-}" ]]; then
        # Validate format
        if [[ "$TWILIO_ACCOUNT_SID" =~ ^AC[a-f0-9]{32}$ ]] || [[ "$TWILIO_ACCOUNT_SID" =~ ^AC.*test.*$ ]]; then
            return 0
        fi
    fi
    
    return 1
}

# Test API connection
twilio::core::test_api_connection() {
    if ! twilio::is_installed; then
        return 1
    fi
    
    if ! twilio::core::has_valid_credentials; then
        return 1
    fi
    
    # For test accounts, just validate format
    if [[ "${TWILIO_ACCOUNT_SID:-}" =~ test ]] || [[ "${TWILIO_ACCOUNT_SID:-}" == "AC_test_"* ]]; then
        log::debug "Test mode detected, skipping API connection test"
        return 0
    fi
    
    local cmd
    cmd=$(twilio::get_command)
    
    # Export credentials for CLI
    export TWILIO_ACCOUNT_SID="${TWILIO_ACCOUNT_SID}"
    export TWILIO_AUTH_TOKEN="${TWILIO_AUTH_TOKEN}"
    
    # Test API connection with timeout
    if timeout 5 "$cmd" api:core:accounts:list --limit 1 &>/dev/null; then
        return 0
    fi
    
    return 1
}

# Health check implementation
twilio::core::health_check() {
    local timeout="${1:-5}"
    
    # Initialize and load secrets
    twilio::core::init
    
    # Quick health check
    if twilio::is_installed && twilio::core::has_valid_credentials; then
        # For test accounts, just check credentials format
        if [[ "${TWILIO_ACCOUNT_SID:-}" =~ test ]]; then
            echo '{"status":"healthy","message":"Test credentials configured"}'
            return 0
        fi
        
        # For production, test API connection
        if timeout "$timeout" bash -c 'twilio::core::test_api_connection' &>/dev/null; then
            echo '{"status":"healthy","message":"API connection successful"}'
            return 0
        else
            echo '{"status":"degraded","message":"API connection failed"}'
            return 1
        fi
    else
        echo '{"status":"unhealthy","message":"Not configured"}'
        return 1
    fi
}

# Get resource information
twilio::core::info() {
    local format="${1:-text}"
    
    # Collect information
    local version=$(twilio::get_version)
    local installed=$(twilio::is_installed && echo "true" || echo "false")
    local configured=$(twilio::core::has_valid_credentials && echo "true" || echo "false")
    local account_sid="${TWILIO_ACCOUNT_SID:-not_configured}"
    
    if [[ "$format" == "json" ]]; then
        cat <<EOF
{
  "name": "twilio",
  "category": "communication",
  "version": "$version",
  "installed": $installed,
  "configured": $configured,
  "account_sid": "$account_sid",
  "config_dir": "$TWILIO_CONFIG_DIR",
  "data_dir": "$TWILIO_DATA_DIR"
}
EOF
    else
        echo "Twilio Resource Information"
        echo "=========================="
        echo "Version: $version"
        echo "Installed: $installed"
        echo "Configured: $configured"
        echo "Account: ${account_sid:0:10}..."
        echo "Config Dir: $TWILIO_CONFIG_DIR"
        echo "Data Dir: $TWILIO_DATA_DIR"
    fi
}

# Clean up resources
twilio::core::cleanup() {
    log::debug "Cleaning up Twilio resources..."
    
    # Stop any running monitors
    if [[ -f "$TWILIO_MONITOR_PID_FILE" ]]; then
        local pid=$(cat "$TWILIO_MONITOR_PID_FILE")
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
        fi
        rm -f "$TWILIO_MONITOR_PID_FILE"
    fi
    
    return 0
}

################################################################################
# Export Functions
################################################################################

export -f twilio::core::init
export -f twilio::core::load_secrets
export -f twilio::core::validate
export -f twilio::core::has_valid_credentials
export -f twilio::core::test_api_connection
export -f twilio::core::health_check
export -f twilio::core::info
export -f twilio::core::cleanup