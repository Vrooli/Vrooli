#!/usr/bin/env bash
# ====================================================================
# n8n API Access Setup Script
# ====================================================================
#
# Automates the process of setting up API access for n8n, addressing
# the authentication confusion identified during integration testing.
#
# This script:
# 1. Detects current n8n authentication state
# 2. Provides clear instructions for API key creation
# 3. Validates authentication setup
# 4. Updates configuration files automatically
#
# Usage:
#   ./setup-api-access.sh                    # Interactive setup
#   ./setup-api-access.sh --api-key KEY      # Set API key directly
#   ./setup-api-access.sh --validate         # Validate existing setup
#   ./setup-api-access.sh --reset            # Reset authentication
#
# ====================================================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
N8N_BASE_URL="http://localhost:5678"
CONFIG_FILE="$HOME/.vrooli/resources.local.json"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Parse command line arguments
API_KEY=""
VALIDATE_ONLY=false
RESET_AUTH=false
INTERACTIVE=true

while [[ $# -gt 0 ]]; do
    case $1 in
        --api-key)
            API_KEY="$2"
            INTERACTIVE=false
            shift 2
            ;;
        --validate)
            VALIDATE_ONLY=true
            INTERACTIVE=false
            shift
            ;;
        --reset)
            RESET_AUTH=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [--api-key KEY] [--validate] [--reset]"
            exit 0
            ;;
        *)
            echo "Unknown option: $1"
            exit 1
            ;;
    esac
done

#######################################
# Print colored output
#######################################
print_status() {
    local color=$1
    local message=$2
    echo -e "${color}${message}${NC}"
}

print_success() { print_status "$GREEN" "✅ $1"; }
print_error() { print_status "$RED" "❌ $1"; }
print_warning() { print_status "$YELLOW" "⚠️  $1"; }
print_info() { print_status "$BLUE" "ℹ️  $1"; }

#######################################
# Check if n8n is running and accessible
#######################################
check_n8n_status() {
    print_info "Checking n8n status..."
    
    if ! curl -s --max-time 5 "$N8N_BASE_URL/healthz" >/dev/null 2>&1; then
        print_error "n8n is not accessible at $N8N_BASE_URL"
        print_info "Make sure n8n is running: ./manage.sh --action start"
        return 1
    fi
    
    local health_response
    health_response=$(curl -s --max-time 5 "$N8N_BASE_URL/healthz" || echo "")
    
    if echo "$health_response" | grep -q '"status":"ok"'; then
        print_success "n8n is running and healthy"
        return 0
    else
        print_warning "n8n is running but may not be fully ready"
        print_info "Response: $health_response"
        return 1
    fi
}

#######################################
# Get n8n basic auth credentials
#######################################
get_basic_auth_credentials() {
    print_info "Getting n8n basic auth credentials..."
    
    # Try to get credentials from management script
    if [[ -f "$SCRIPT_DIR/manage.sh" ]]; then
        local auth_info
        auth_info=$("$SCRIPT_DIR/manage.sh" --action api-setup 2>/dev/null || echo "")
        
        if echo "$auth_info" | grep -q "Username:" && echo "$auth_info" | grep -q "Password:"; then
            local username
            local password
            username=$(echo "$auth_info" | grep "Username:" | cut -d' ' -f2)
            password=$(echo "$auth_info" | grep "Password:" | cut -d' ' -f2)
            
            print_success "Found basic auth credentials"
            print_info "Username: $username"
            print_info "Password: ${password:0:4}***"
            
            echo "$username:$password"
            return 0
        fi
    fi
    
    # Default credentials if not found
    print_warning "Could not retrieve credentials from management script"
    print_info "Using default credentials (admin:password)"
    echo "admin:password"
}

#######################################
# Validate API key works
#######################################
validate_api_key() {
    local api_key="$1"
    
    if [[ -z "$api_key" ]]; then
        print_error "No API key provided for validation"
        return 1
    fi
    
    print_info "Validating API key..."
    
    local response
    response=$(curl -s -w "%{http_code}" \
        -H "X-N8N-API-KEY: $api_key" \
        "$N8N_BASE_URL/rest/workflows" || echo "000")
    
    local http_code="${response: -3}"
    local body="${response%???}"
    
    case "$http_code" in
        200)
            print_success "API key is valid and working"
            return 0
            ;;
        401)
            print_error "API key is invalid or expired"
            return 1
            ;;
        403)
            print_error "API key is valid but lacks necessary permissions"
            return 1
            ;;
        *)
            print_error "Unexpected response (HTTP $http_code)"
            print_info "Response: $body"
            return 1
            ;;
    esac
}

#######################################
# Update configuration file with API key
#######################################
update_config_file() {
    local api_key="$1"
    
    print_info "Updating configuration file..."
    
    if [[ ! -f "$CONFIG_FILE" ]]; then
        print_warning "Configuration file not found, creating basic structure"
        mkdir -p "$(dirname "$CONFIG_FILE")"
        echo '{"services": {"automation": {}}}' > "$CONFIG_FILE"
    fi
    
    # Use jq to update the configuration
    if command -v jq >/dev/null 2>&1; then
        local temp_file
        temp_file=$(mktemp)
        
        jq --arg key "$api_key" \
           '.services.automation.n8n.apiKey = $key' \
           "$CONFIG_FILE" > "$temp_file" && mv "$temp_file" "$CONFIG_FILE"
        
        print_success "Configuration file updated with API key"
        return 0
    else
        print_warning "jq not available, cannot update configuration automatically"
        print_info "Please manually add the API key to $CONFIG_FILE:"
        print_info "  \"services\": { \"automation\": { \"n8n\": { \"apiKey\": \"$api_key\" } } }"
        return 1
    fi
}

#######################################
# Interactive API key setup
#######################################
interactive_setup() {
    print_info "=== n8n API Access Setup ==="
    echo
    
    # Get basic auth credentials
    local basic_auth
    basic_auth=$(get_basic_auth_credentials)
    local username="${basic_auth%:*}"
    local password="${basic_auth#*:}"
    
    echo
    print_info "To set up API access, you need to:"
    echo "  1. Open n8n web interface: $N8N_BASE_URL"
    echo "  2. Login with: $username / $password"
    echo "  3. Go to Settings → n8n API"
    echo "  4. Click 'Create an API key'"
    echo "  5. Set a label (e.g., 'CLI Access')"
    echo "  6. Copy the generated API key"
    echo
    
    print_warning "The API key is shown only once - make sure to copy it!"
    echo
    
    # Wait for user to create API key
    local api_key
    while true; do
        read -r -p "Enter the API key (or 'quit' to exit): " api_key
        
        if [[ "$api_key" == "quit" ]]; then
            print_info "Setup cancelled"
            return 1
        fi
        
        if [[ -z "$api_key" ]]; then
            print_warning "Please enter a valid API key"
            continue
        fi
        
        # Test the API key
        if validate_api_key "$api_key"; then
            break
        else
            print_warning "API key validation failed, please try again"
        fi
    done
    
    # Update configuration
    if update_config_file "$api_key"; then
        echo
        print_success "API access setup complete!"
        print_info "You can now use the n8n API with workflows"
        return 0
    else
        print_error "Failed to update configuration file"
        return 1
    fi
}

#######################################
# Reset authentication setup
#######################################
reset_authentication() {
    print_info "Resetting n8n authentication setup..."
    
    if [[ -f "$CONFIG_FILE" ]] && command -v jq >/dev/null 2>&1; then
        local temp_file
        temp_file=$(mktemp)
        
        jq 'del(.services.automation.n8n.apiKey)' \
           "$CONFIG_FILE" > "$temp_file" && mv "$temp_file" "$CONFIG_FILE"
        
        print_success "API key removed from configuration"
    fi
    
    print_info "Authentication has been reset"
    print_info "Run this script again to set up a new API key"
}

#######################################
# Validate existing setup
#######################################
validate_existing_setup() {
    print_info "Validating existing n8n API setup..."
    
    # Check if API key exists in config
    local existing_key=""
    if [[ -f "$CONFIG_FILE" ]] && command -v jq >/dev/null 2>&1; then
        existing_key=$(jq -r '.services.automation.n8n.apiKey // empty' "$CONFIG_FILE" 2>/dev/null || echo "")
    fi
    
    if [[ -z "$existing_key" ]]; then
        print_warning "No API key found in configuration"
        print_info "Run this script without --validate to set up API access"
        return 1
    fi
    
    print_info "Found API key in configuration"
    
    # Validate the key
    if validate_api_key "$existing_key"; then
        print_success "Existing API setup is working correctly"
        return 0
    else
        print_error "Existing API key is not working"
        print_info "Run this script with --reset to set up a new API key"
        return 1
    fi
}

#######################################
# Main execution
#######################################
main() {
    echo "n8n API Access Setup"
    echo "===================="
    echo
    
    # Check n8n status first
    if ! check_n8n_status; then
        exit 1
    fi
    echo
    
    # Handle different modes
    if [[ "$RESET_AUTH" == true ]]; then
        reset_authentication
        exit 0
    fi
    
    if [[ "$VALIDATE_ONLY" == true ]]; then
        validate_existing_setup
        exit $?
    fi
    
    if [[ -n "$API_KEY" ]]; then
        # Direct API key provided
        print_info "Setting up API access with provided key..."
        if validate_api_key "$API_KEY" && update_config_file "$API_KEY"; then
            print_success "API access setup complete!"
            exit 0
        else
            print_error "Setup failed"
            exit 1
        fi
    fi
    
    if [[ "$INTERACTIVE" == true ]]; then
        interactive_setup
        exit $?
    fi
    
    print_error "No valid operation specified"
    exit 1
}

# Execute main function
main "$@"