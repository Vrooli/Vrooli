#!/usr/bin/env bash
# Vault Development Token Setup
# Script for setting up development tokens for Vault

# Source shared secrets management library
# Use the same project root detection method as the secrets library
_vault_setup_detect_project_root() {
    local current_dir
    APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
    current_dir="${APP_ROOT}/resources/vault/lib"
    
    # Walk up directory tree looking for .vrooli directory
    while [[ "$current_dir" != "/" ]]; do
        if [[ -d "$current_dir/.vrooli" ]]; then
            echo "$current_dir"
            return 0
        fi
        current_dir="${current_dir%/*"
    done
    
    # Fallback: assume we're in scripts and go up to project root
    echo "/home/matthalloran8/Vrooli"
}

PROJECT_ROOT="$(_vault_setup_detect_project_root)"
# shellcheck disable=SC1091
source "$PROJECT_ROOT/scripts/lib/service/secrets.sh"
# shellcheck disable=SC1091
source "$PROJECT_ROOT/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LIB_SYSTEM_DIR}/trash.sh" 2>/dev/null || true
# ====================================================================
# Vault Development Token Setup Script  
# ====================================================================
#
# Automates Vault token management and authentication setup for 
# development environments, addressing the token confusion identified
# during integration testing.
#
# This script:
# 1. Detects current Vault state (sealed/unsealed/initialized)
# 2. Manages development tokens safely  
# 3. Creates test secrets for integration testing
# 4. Validates token permissions and access
#
# Usage:
#   ./setup-dev-tokens.sh                     # Interactive setup
#   ./setup-dev-tokens.sh --create-token      # Create new dev token
#   ./setup-dev-tokens.sh --validate          # Validate existing token
#   ./setup-dev-tokens.sh --setup-secrets     # Create test secrets
#   ./setup-dev-tokens.sh --reset             # Reset dev environment
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
VAULT_BASE_URL="http://localhost:8200"
TOKEN_FILE="/tmp/vault-token"
CONFIG_FILE="$(secrets::get_project_config_file)"
APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*/../../.." && builtin pwd)}"
VAULT_LIB_DIR="${APP_ROOT}/resources/vault/lib"

# Parse command line arguments
CREATE_TOKEN=false
VALIDATE_ONLY=false
SETUP_SECRETS=false
RESET_ENV=false
INTERACTIVE=true

while [[ $# -gt 0 ]]; do
    case $1 in
        --create-token)
            CREATE_TOKEN=true
            INTERACTIVE=false
            shift
            ;;
        --validate)
            VALIDATE_ONLY=true
            INTERACTIVE=false
            shift
            ;;
        --setup-secrets)
            SETUP_SECRETS=true
            INTERACTIVE=false
            shift
            ;;
        --reset)
            RESET_ENV=true
            shift
            ;;
        -h|--help)
            echo "Usage: $0 [--create-token] [--validate] [--setup-secrets] [--reset]"
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
# Check Vault status
#######################################
check_vault_status() {
    print_info "Checking Vault status..."
    
    # Check if Vault is accessible
    if ! curl -s --max-time 5 "$VAULT_BASE_URL/v1/sys/health" >/dev/null 2>&1; then
        print_error "Vault is not accessible at $VAULT_BASE_URL"
        print_info "Make sure Vault is running: ./manage.sh --action start"
        return 1
    fi
    
    # Get detailed status
    local health_response
    health_response=$(curl -s "$VAULT_BASE_URL/v1/sys/health" || echo "{}")
    
    local initialized
    local sealed
    initialized=$(echo "$health_response" | jq -r '.initialized // false')
    sealed=$(echo "$health_response" | jq -r '.sealed // true')
    
    if [[ "$initialized" != "true" ]]; then
        print_error "Vault is not initialized"
        print_info "Initialize Vault: ./manage.sh --action init-dev"
        return 1
    fi
    
    if [[ "$sealed" == "true" ]]; then
        print_error "Vault is sealed"
        print_info "Unseal Vault: ./manage.sh --action unseal"
        return 1
    fi
    
    print_success "Vault is initialized and unsealed"
    return 0
}

#######################################
# Get current token from file
#######################################
get_current_token() {
    if [[ -f "$TOKEN_FILE" ]]; then
        cat "$TOKEN_FILE" 2>/dev/null || echo ""
    else
        echo ""
    fi
}

#######################################
# Validate token permissions
#######################################
validate_token() {
    local token="$1"
    
    if [[ -z "$token" ]]; then
        print_error "No token provided for validation"
        return 1
    fi
    
    print_info "Validating token permissions..."
    
    # Test token lookup
    local lookup_response
    lookup_response=$(curl -s -H "X-Vault-Token: $token" \
        "$VAULT_BASE_URL/v1/auth/token/lookup-self" 2>/dev/null || echo "{}")
    
    if echo "$lookup_response" | grep -q '"errors"'; then
        print_error "Token is invalid or expired"
        return 1
    fi
    
    # Check policies
    local policies
    policies=$(echo "$lookup_response" | jq -r '.data.policies[]? // empty' 2>/dev/null)
    
    if [[ -z "$policies" ]]; then
        print_warning "Token has no policies assigned"
        return 1
    fi
    
    print_success "Token is valid"
    print_info "Policies: $policies"
    
    # Test basic secret operations
    print_info "Testing secret operations..."
    
    # Test write permission
    local test_path="test/token-validation"
    local test_data='{"test": "validation"}'
    
    local write_response
    write_response=$(curl -s -X POST \
        -H "X-Vault-Token: $token" \
        -H "Content-Type: application/json" \
        -d "{\"data\": $test_data}" \
        "$VAULT_BASE_URL/v1/secret/data/$test_path" 2>/dev/null || echo "{}")
    
    if echo "$write_response" | grep -q '"errors"'; then
        print_warning "Token cannot write secrets"
        print_info "Write error: $(echo "$write_response" | jq -r '.errors[]? // "unknown"')"
    else
        print_success "Token can write secrets"
        
        # Clean up test secret
        curl -s -X DELETE -H "X-Vault-Token: $token" \
            "$VAULT_BASE_URL/v1/secret/data/$test_path" >/dev/null 2>&1 || true
    fi
    
    return 0
}

#######################################
# Create development token
#######################################
create_dev_token() {
    print_info "Creating development token..."
    
    # For development, we'll use the root token that should already exist
    local root_token
    root_token=$(get_current_token)
    
    if [[ -z "$root_token" ]]; then
        print_error "No existing token found"
        print_info "Initialize Vault first: ./manage.sh --action init-dev"
        return 1
    fi
    
    # Validate the existing token
    if validate_token "$root_token"; then
        print_success "Development token is ready: $TOKEN_FILE"
        return 0
    else
        print_error "Existing token is not valid"
        return 1
    fi
}

#######################################
# Setup test secrets for integration
#######################################
setup_test_secrets() {
    print_info "Setting up test secrets for integration testing..."
    
    local token
    token=$(get_current_token)
    
    if [[ -z "$token" ]]; then
        print_error "No token available for secret setup"
        return 1
    fi
    
    # Define test secrets
    local secrets=(
        "test/stripe-config:{\"stripe_api_key\":\"sk_test_51AbCdEfGhIjKl\",\"environment\":\"test\"}"
        "test/openai-config:{\"openai_api_key\":\"sk-test-openai-abc123\",\"model\":\"gpt-4\",\"max_tokens\":\"1000\"}"
        "test/database-config:{\"host\":\"localhost\",\"port\":\"5432\",\"username\":\"test_user\",\"password\":\"test_pass\",\"database\":\"test_db\"}"
        "test/api-keys:{\"github_token\":\"ghp_test123\",\"discord_webhook\":\"https://discord.com/api/webhooks/test\",\"slack_token\":\"xoxb-test-token\"}"
    )
    
    local success_count=0
    local total_count=${#secrets[@]}
    
    for secret_def in "${secrets[@]}"; do
        local path="${secret_def%%:*}"
        local data="${secret_def#*:}"
        
        print_info "Creating secret: $path"
        
        local response
        response=$(curl -s -X POST \
            -H "X-Vault-Token: $token" \
            -H "Content-Type: application/json" \
            -d "{\"data\": $data}" \
            "$VAULT_BASE_URL/v1/secret/data/$path" 2>/dev/null || echo "{}")
        
        if echo "$response" | grep -q '"version"'; then
            echo "  ✅ Created successfully"
            ((success_count++))
        else
            echo "  ❌ Failed to create"
            print_info "  Error: $(echo "$response" | jq -r '.errors[]? // "unknown"')"
        fi
    done
    
    print_info "Created $success_count/$total_count test secrets"
    
    if [[ $success_count -eq $total_count ]]; then
        print_success "All test secrets created successfully"
        return 0
    else
        print_warning "Some test secrets failed to create"
        return 1
    fi
}

#######################################
# Update configuration file
#######################################
update_config_file() {
    local token="$1"
    
    print_info "Updating configuration file..."
    
    if [[ ! -f "$CONFIG_FILE" ]]; then
        print_warning "Configuration file not found, creating basic structure"
        mkdir -p "${CONFIG_FILE%/*"
        echo '{"services": {"storage": {}}}' > "$CONFIG_FILE"
    fi
    
    # Use jq to update the configuration
    if command -v jq >/dev/null 2>&1; then
        local temp_file
        temp_file=$(mktemp)
        
        jq --arg token "$token" \
           '.services.storage.vault.token = $token' \
           "$CONFIG_FILE" > "$temp_file" && mv "$temp_file" "$CONFIG_FILE"
        
        print_success "Configuration file updated with token"
        return 0
    else
        print_warning "jq not available, cannot update configuration automatically"
        print_info "Please manually add the token to $CONFIG_FILE:"
        print_info "  \"services\": { \"storage\": { \"vault\": { \"token\": \"$token\" } } }"
        return 1
    fi
}

#######################################
# Reset development environment
#######################################
reset_dev_environment() {
    print_info "Resetting Vault development environment..."
    
    local token
    token=$(get_current_token)
    
    if [[ -n "$token" ]]; then
        # Remove test secrets
        print_info "Cleaning up test secrets..."
        local test_paths=("test/stripe-config" "test/openai-config" "test/database-config" "test/api-keys")
        
        for path in "${test_paths[@]}"; do
            curl -s -X DELETE -H "X-Vault-Token: $token" \
                "$VAULT_BASE_URL/v1/secret/data/$path" >/dev/null 2>&1 || true
        done
    fi
    
    # Clean up token file
    if [[ -f "$TOKEN_FILE" ]]; then
        trash::safe_remove "$TOKEN_FILE" --temp
        print_info "Token file removed"
    fi
    
    # Update configuration
    if [[ -f "$CONFIG_FILE" ]] && command -v jq >/dev/null 2>&1; then
        local temp_file
        temp_file=$(mktemp)
        
        jq 'del(.services.storage.vault.token)' \
           "$CONFIG_FILE" > "$temp_file" && mv "$temp_file" "$CONFIG_FILE"
        
        print_info "Token removed from configuration"
    fi
    
    print_success "Development environment reset complete"
}

#######################################
# Interactive setup
#######################################
interactive_setup() {
    print_info "=== Vault Development Token Setup ==="
    echo
    
    # Create/validate token
    if ! create_dev_token; then
        print_error "Failed to set up development token"
        return 1
    fi
    echo
    
    # Setup test secrets
    print_info "Setting up test secrets for integration testing..."
    if setup_test_secrets; then
        print_success "Test secrets created successfully"
    else
        print_warning "Some test secrets failed to create"
    fi
    echo
    
    # Update configuration
    local token
    token=$(get_current_token)
    if [[ -n "$token" ]]; then
        update_config_file "$token"
    fi
    echo
    
    print_success "Vault development setup complete!"
    echo
    print_info "Next steps:"
    print_info "  1. Test Vault access: curl -H \"X-Vault-Token: $token\" $VAULT_BASE_URL/v1/secret/data/test/stripe-config"
    print_info "  2. Run integration tests: ../../index.sh --action test --resources vault"
    print_info "  3. Test n8n integration: ../../scenarios/research-assistant/test.sh"
}

#######################################
# Main execution
#######################################
main() {
    echo "Vault Development Token Setup"
    echo "============================="
    echo
    
    # Check Vault status first
    if ! check_vault_status; then
        exit 1
    fi
    echo
    
    # Handle different modes
    if [[ "$RESET_ENV" == true ]]; then
        reset_dev_environment
        exit 0
    fi
    
    if [[ "$VALIDATE_ONLY" == true ]]; then
        local token
        token=$(get_current_token)
        if [[ -n "$token" ]]; then
            validate_token "$token"
            exit $?
        else
            print_error "No token found to validate"
            exit 1
        fi
    fi
    
    if [[ "$CREATE_TOKEN" == true ]]; then
        create_dev_token
        exit $?
    fi
    
    if [[ "$SETUP_SECRETS" == true ]]; then
        setup_test_secrets
        exit $?
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