#!/bin/bash
# Secrets Integration Library for Vrooli Resources
# This library provides standard functions for resources to integrate with Vault secrets management

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ============================================================================
# Resource Secrets Integration Functions
# ============================================================================

# Initialize secrets.yaml template for a resource
resource_secrets_init() {
    local resource_name="${1:-${RESOURCE_NAME:-}}"
    
    if [[ -z "$resource_name" ]]; then
        echo -e "${RED}Error: Resource name not provided${NC}"
        return 1
    fi
    
    # Check if Vault CLI is available
    if ! command -v resource-vault &>/dev/null; then
        echo -e "${YELLOW}Warning: Vault CLI not available. Install with 'vrooli resource vault install'${NC}"
        return 1
    fi
    
    # Delegate to Vault's create-template command
    resource-vault secrets create-template "$resource_name"
    return $?
}

# Check if resource secrets are configured
resource_secrets_check() {
    local resource_name="${1:-${RESOURCE_NAME:-}}"
    
    if [[ -z "$resource_name" ]]; then
        echo -e "${RED}Error: Resource name not provided${NC}"
        return 1
    fi
    
    # Check if Vault is available
    if ! command -v resource-vault &>/dev/null; then
        echo -e "${YELLOW}Vault not installed - secrets cannot be checked${NC}"
        return 1
    fi
    
    # Check if Vault is running
    if ! resource-vault status &>/dev/null; then
        echo -e "${YELLOW}Vault is not running - cannot check secrets${NC}"
        return 1
    fi
    
    # Delegate to Vault's check command
    resource-vault secrets check "$resource_name"
    return $?
}

# Test that secrets work for the resource
resource_secrets_test() {
    local resource_name="${1:-${RESOURCE_NAME:-}}"
    
    if [[ -z "$resource_name" ]]; then
        echo -e "${RED}Error: Resource name not provided${NC}"
        return 1
    fi
    
    local resources_dir="${VROOLI_ROOT:-$HOME/Vrooli}/resources"
    local secrets_file="$resources_dir/$resource_name/config/secrets.yaml"
    
    if [[ ! -f "$secrets_file" ]]; then
        echo -e "${YELLOW}No secrets.yaml found for $resource_name${NC}"
        return 0  # Not an error if resource doesn't need secrets
    fi
    
    echo -e "${BLUE}Testing secrets for ${resource_name}...${NC}"
    
    # Load secrets into environment
    if load_resource_secrets "$resource_name"; then
        echo -e "${GREEN}✓ Secrets loaded successfully${NC}"
        
        # Run resource-specific health check if defined
        if command -v yq &>/dev/null; then
            local health_endpoint=$(yq eval '.health_check.endpoint // ""' "$secrets_file")
            if [[ -n "$health_endpoint" ]]; then
                echo "Testing health endpoint: $health_endpoint"
                # This would be resource-specific
                echo -e "${BLUE}Health check endpoint defined but must be tested by resource${NC}"
            fi
        fi
        
        return 0
    else
        echo -e "${RED}✗ Failed to load secrets${NC}"
        return 1
    fi
}

# Load secrets from Vault into environment variables
load_resource_secrets() {
    local resource_name="${1:-${RESOURCE_NAME:-}}"
    
    if [[ -z "$resource_name" ]]; then
        echo -e "${RED}Error: Resource name not provided${NC}" >&2
        return 1
    fi
    
    # Check if Vault is available and running
    if ! command -v resource-vault &>/dev/null; then
        # Vault not installed, try environment variables
        log_debug "Vault not available, using environment variables"
        return 0
    fi
    
    if ! resource-vault status &>/dev/null 2>&1; then
        # Vault not running, use environment variables
        log_debug "Vault not running, using environment variables"
        return 0
    fi
    
    # Export secrets from Vault
    local export_commands
    export_commands=$(resource-vault secrets export "$resource_name" 2>/dev/null)
    
    if [[ -n "$export_commands" ]]; then
        # Filter out comments and execute export commands
        while IFS= read -r line; do
            if [[ "$line" =~ ^export\ .+ ]]; then
                eval "$line"
            fi
        done <<< "$export_commands"
        
        log_debug "Loaded secrets from Vault for $resource_name"
        return 0
    else
        log_debug "No secrets found in Vault for $resource_name"
        return 0  # Not an error if no secrets
    fi
}

# Helper function to get a specific secret value
get_resource_secret() {
    local resource_name="${1:-${RESOURCE_NAME:-}}"
    local secret_name="${2:-}"
    local env_var="${3:-}"
    
    if [[ -z "$resource_name" || -z "$secret_name" ]]; then
        echo -e "${RED}Error: Resource name and secret name required${NC}" >&2
        return 1
    fi
    
    # First try environment variable if provided
    if [[ -n "$env_var" && -n "${!env_var:-}" ]]; then
        echo "${!env_var}"
        return 0
    fi
    
    # Try to get from Vault
    if command -v resource-vault &>/dev/null && resource-vault status &>/dev/null 2>&1; then
        local path="secret/resources/${resource_name}/${secret_name}"
        local value
        value=$(resource-vault content get --path "$path" 2>/dev/null | jq -r '.data.data.value // empty' 2>/dev/null)
        
        if [[ -n "$value" ]]; then
            echo "$value"
            return 0
        fi
    fi
    
    # No secret found
    return 1
}

# Register secrets commands for a resource CLI
register_secrets_commands() {
    local resource_name="${1:-${RESOURCE_NAME:-}}"
    
    if [[ -z "$resource_name" ]]; then
        echo -e "${RED}Error: Resource name not provided${NC}" >&2
        return 1
    fi
    
    # These functions will be available to resources that source this library
    # Resources can then add these to their CLI
    
    # Define wrapper functions
    eval "
    ${resource_name}_secrets_init() {
        resource_secrets_init '$resource_name'
    }
    
    ${resource_name}_secrets_check() {
        resource_secrets_check '$resource_name'
    }
    
    ${resource_name}_secrets_test() {
        resource_secrets_test '$resource_name'
    }
    "
    
    # Export the functions
    export -f "${resource_name}_secrets_init"
    export -f "${resource_name}_secrets_check"
    export -f "${resource_name}_secrets_test"
    
    echo -e "${GREEN}Secrets commands registered for ${resource_name}${NC}"
}

# Check if secrets.yaml exists for a resource
has_secrets_config() {
    local resource_name="${1:-${RESOURCE_NAME:-}}"
    
    if [[ -z "$resource_name" ]]; then
        return 1
    fi
    
    local resources_dir="${VROOLI_ROOT:-$HOME/Vrooli}/resources"
    local secrets_file="$resources_dir/$resource_name/config/secrets.yaml"
    
    [[ -f "$secrets_file" ]]
}

# Logging functions (fallback if not already defined)
if ! declare -f log_debug &>/dev/null; then
    log_debug() {
        [[ "${DEBUG:-}" == "true" ]] && echo "[DEBUG] $*" >&2
    }
fi

if ! declare -f log_info &>/dev/null; then
    log_info() {
        echo "[INFO] $*"
    }
fi

if ! declare -f log_warn &>/dev/null; then
    log_warn() {
        echo -e "${YELLOW}[WARN] $*${NC}"
    }
fi

if ! declare -f log_error &>/dev/null; then
    log_error() {
        echo -e "${RED}[ERROR] $*${NC}" >&2
    }
fi

# Export all functions for use by resources
export -f resource_secrets_init
export -f resource_secrets_check
export -f resource_secrets_test
export -f load_resource_secrets
export -f get_resource_secret
export -f register_secrets_commands
export -f has_secrets_config