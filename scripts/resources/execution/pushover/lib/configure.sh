#!/bin/bash
# Pushover configuration functionality

# Get script directory
PUSHOVER_CONFIGURE_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Source dependencies
source "${PUSHOVER_CONFIGURE_DIR}/core.sh"

# Configure Pushover credentials
pushover::configure() {
    local app_token=""
    local user_key=""
    local use_vault="${1:-true}"
    
    log::header "Configure Pushover"
    echo
    
    # Initialize
    pushover::init false
    
    # Check if already configured
    if pushover::is_configured; then
        log::info "Pushover is already configured"
        echo "Current configuration source:"
        if [[ -f "$PUSHOVER_CREDENTIALS_FILE" ]]; then
            echo "  - Local credentials file"
        fi
        if command -v resource-vault >/dev/null 2>&1; then
            local vault_data
            vault_data=$(resource-vault get pushover 2>/dev/null)
            if [[ -n "$vault_data" ]]; then
                echo "  - Vault (secure storage)"
            fi
        fi
        echo
        read -p "Do you want to reconfigure? (y/N): " reconfigure
        if [[ "${reconfigure,,}" != "y" ]]; then
            return 0
        fi
    fi
    
    # Get credentials from user
    echo "Please provide your Pushover credentials"
    echo "Get them from: https://pushover.net"
    echo
    
    read -p "App Token: " app_token
    read -p "User Key: " user_key
    
    # Validate inputs
    if [[ -z "$app_token" ]] || [[ -z "$user_key" ]]; then
        log::error "Both App Token and User Key are required"
        return 1
    fi
    
    # Test credentials
    log::info "Testing credentials..."
    local test_response
    test_response=$(curl -s -X POST \
        -F "token=${app_token}" \
        -F "user=${user_key}" \
        "${PUSHOVER_API_URL}/users/validate.json" 2>/dev/null)
    
    if ! echo "$test_response" | jq -e '.status == 1' >/dev/null 2>&1; then
        log::error "Invalid credentials. Please check your App Token and User Key"
        echo "$test_response" | jq '.' 2>/dev/null
        return 1
    fi
    
    log::success "Credentials validated successfully"
    
    # Store credentials
    local storage_method="local"
    
    # Try Vault first if available
    if [[ "$use_vault" == "true" ]] && command -v resource-vault >/dev/null 2>&1; then
        local vault_status
        vault_status=$(resource-vault status --format json 2>/dev/null)
        
        if echo "$vault_status" | jq -e '.unsealed == true' >/dev/null 2>&1; then
            log::info "Storing credentials in Vault (secure)..."
            
            # Create JSON for Vault
            local vault_json
            vault_json=$(jq -n \
                --arg app_token "$app_token" \
                --arg user_key "$user_key" \
                '{app_token: $app_token, user_key: $user_key}')
            
            if echo "$vault_json" | resource-vault set pushover - >/dev/null 2>&1; then
                log::success "Credentials stored in Vault"
                storage_method="vault"
            else
                log::warning "Failed to store in Vault, falling back to local file"
            fi
        fi
    fi
    
    # Store locally if Vault not used or failed
    if [[ "$storage_method" == "local" ]]; then
        log::info "Storing credentials locally..."
        
        # Create credentials file
        cat > "$PUSHOVER_CREDENTIALS_FILE" <<EOF
{
  "app_token": "${app_token}",
  "user_key": "${user_key}"
}
EOF
        
        # Set restrictive permissions
        chmod 600 "$PUSHOVER_CREDENTIALS_FILE"
        
        log::success "Credentials stored in: $PUSHOVER_CREDENTIALS_FILE"
        log::warning "Consider using Vault for more secure storage"
    fi
    
    # Test final configuration
    export PUSHOVER_APP_TOKEN="$app_token"
    export PUSHOVER_USER_KEY="$user_key"
    
    if pushover::health_check true; then
        log::success "Pushover configured successfully!"
        echo
        echo "You can now send notifications with:"
        echo "  resource-pushover send -m \"Your message\""
        return 0
    else
        log::error "Configuration failed"
        return 1
    fi
}

# Clear Pushover credentials
pushover::clear_credentials() {
    log::header "Clear Pushover Credentials"
    
    local cleared=false
    
    # Clear from Vault
    if command -v resource-vault >/dev/null 2>&1; then
        if resource-vault delete pushover >/dev/null 2>&1; then
            log::info "Cleared credentials from Vault"
            cleared=true
        fi
    fi
    
    # Clear local file
    if [[ -f "$PUSHOVER_CREDENTIALS_FILE" ]]; then
        rm -f "$PUSHOVER_CREDENTIALS_FILE"
        log::info "Cleared local credentials file"
        cleared=true
    fi
    
    # Clear environment
    unset PUSHOVER_APP_TOKEN
    unset PUSHOVER_USER_KEY
    
    if [[ "$cleared" == "true" ]]; then
        log::success "Credentials cleared successfully"
    else
        log::info "No credentials found to clear"
    fi
    
    return 0
}