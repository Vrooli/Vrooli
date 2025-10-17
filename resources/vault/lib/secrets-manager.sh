#!/bin/bash
# Vault Secrets Manager - Manages secrets for all Vrooli resources
# This implements the secrets.yaml standard for resource secret management

set -euo pipefail

# Get the directory of this script
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
RESOURCE_DIR="$(dirname "$SCRIPT_DIR")"

# Source required libraries
source "$SCRIPT_DIR/common.sh"
source "$SCRIPT_DIR/api.sh"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# ============================================================================
# Core Functions
# ============================================================================

# Scan all resources for secrets.yaml files
secrets_scan() {
    local verbose="${1:-false}"
    local resources_dir="${VROOLI_ROOT:-$HOME/Vrooli}/resources"
    local found_count=0
    local total_secrets=0
    
    echo -e "${BLUE}Scanning resources for secrets declarations...${NC}"
    echo
    
    # Find all secrets.yaml files
    while IFS= read -r secrets_file; do
        if [[ -f "$secrets_file" ]]; then
            local resource_name=$(basename "$(dirname "$(dirname "$secrets_file")")")
            local secret_count=0
            
            # Parse the YAML to count secrets
            if command -v yq &>/dev/null; then
                secret_count=$(yq eval '.secrets | .. | select(has("name")) | .name' "$secrets_file" 2>/dev/null | wc -l)
            else
                # Fallback to grep if yq not available
                secret_count=$(grep -c "^\s*- name:" "$secrets_file" 2>/dev/null || echo "0")
            fi
            
            ((found_count++))
            ((total_secrets += secret_count))
            
            echo -e "  ${GREEN}✓${NC} ${resource_name}: ${secret_count} secrets defined"
            
            if [[ "$verbose" == "true" ]]; then
                # Show secret names
                if command -v yq &>/dev/null; then
                    local names=$(yq eval '.secrets | .. | select(has("name")) | .name' "$secrets_file" 2>/dev/null)
                    if [[ -n "$names" ]]; then
                        echo "$names" | while read -r name; do
                            echo "      - $name"
                        done
                    fi
                fi
            fi
        fi
    done < <(find "$resources_dir" -type f -path "*/config/secrets.yaml" 2>/dev/null)
    
    echo
    echo -e "${BLUE}Summary:${NC} Found ${found_count} resources with ${total_secrets} total secrets defined"
    
    return 0
}

# Check secrets status for a specific resource
secrets_check() {
    local resource_name="${1:-}"
    
    if [[ -z "$resource_name" ]]; then
        echo -e "${RED}Error: Resource name required${NC}"
        echo "Usage: resource-vault secrets check <resource-name>"
        return 1
    fi
    
    local resources_dir="${VROOLI_ROOT:-$HOME/Vrooli}/resources"
    local secrets_file="$resources_dir/$resource_name/config/secrets.yaml"
    
    if [[ ! -f "$secrets_file" ]]; then
        echo -e "${YELLOW}Warning: No secrets.yaml found for resource '$resource_name'${NC}"
        echo "Path checked: $secrets_file"
        return 1
    fi
    
    echo -e "${BLUE}Checking secrets for resource: ${resource_name}${NC}"
    echo
    
    # Ensure Vault is unsealed and accessible
    if ! vault_is_ready; then
        echo -e "${RED}Error: Vault is not ready (sealed or not running)${NC}"
        return 1
    fi
    
    local missing_count=0
    local found_count=0
    local optional_count=0
    
    # Parse secrets.yaml and check each secret
    if command -v yq &>/dev/null; then
        # Process each secret category
        for category in api_keys database tokens certificates; do
            local secrets=$(yq eval ".secrets.${category}[]" "$secrets_file" 2>/dev/null)
            if [[ -n "$secrets" ]]; then
                echo -e "${BLUE}Category: ${category}${NC}"
                
                # Process each secret in the category
                while IFS= read -r secret_yaml; do
                    local name=$(echo "$secret_yaml" | yq eval '.name' -)
                    local path=$(echo "$secret_yaml" | yq eval '.path' - | sed "s/{resource}/$resource_name/g")
                    local required=$(echo "$secret_yaml" | yq eval '.required // true' -)
                    local description=$(echo "$secret_yaml" | yq eval '.description // ""' -)
                    
                    if [[ -n "$name" && -n "$path" ]]; then
                        # Check if secret exists in Vault
                        if vault::secret_exists "$path"; then
                            echo -e "  ${GREEN}✓${NC} $name: Found at $path"
                            ((found_count++))
                        else
                            if [[ "$required" == "true" ]]; then
                                echo -e "  ${RED}✗${NC} $name: MISSING (required) - $description"
                                ((missing_count++))
                            else
                                echo -e "  ${YELLOW}○${NC} $name: Not set (optional) - $description"
                                ((optional_count++))
                            fi
                        fi
                    fi
                done < <(yq eval ".secrets.${category}[]" "$secrets_file" 2>/dev/null)
            fi
        done
    else
        echo -e "${YELLOW}Warning: yq not installed, using basic parsing${NC}"
        # Fallback to basic grep parsing
        while IFS= read -r line; do
            if [[ "$line" =~ name:\ *\"([^\"]+)\" ]] || [[ "$line" =~ name:\ *\'([^\']+)\' ]]; then
                local name="${BASH_REMATCH[1]}"
                echo "  ? $name: (install yq for full checking)"
            fi
        done < "$secrets_file"
    fi
    
    echo
    echo -e "${BLUE}Summary for ${resource_name}:${NC}"
    echo "  Found: ${found_count} secrets"
    echo "  Missing (required): ${missing_count} secrets"
    echo "  Not set (optional): ${optional_count} secrets"
    
    if [[ $missing_count -gt 0 ]]; then
        echo
        echo -e "${YELLOW}Run 'resource-vault secrets init ${resource_name}' to set missing secrets${NC}"
        return 1
    fi
    
    return 0
}

# Initialize secrets for a resource (interactive)
secrets_init() {
    local resource_name="${1:-}"
    
    if [[ -z "$resource_name" ]]; then
        echo -e "${RED}Error: Resource name required${NC}"
        echo "Usage: resource-vault secrets init <resource-name>"
        return 1
    fi
    
    local resources_dir="${VROOLI_ROOT:-$HOME/Vrooli}/resources"
    local secrets_file="$resources_dir/$resource_name/config/secrets.yaml"
    
    if [[ ! -f "$secrets_file" ]]; then
        echo -e "${YELLOW}No secrets.yaml found for resource '$resource_name'${NC}"
        echo "Would you like to create one? (y/n)"
        read -r response
        if [[ "$response" == "y" ]]; then
            secrets_create_template "$resource_name"
            return $?
        else
            return 1
        fi
    fi
    
    echo -e "${BLUE}Initializing secrets for resource: ${resource_name}${NC}"
    echo
    
    # Ensure Vault is ready
    if ! vault_is_ready; then
        echo -e "${RED}Error: Vault is not ready${NC}"
        return 1
    fi
    
    # Process initialization section if present
    if command -v yq &>/dev/null; then
        # Auto-generate secrets if specified
        local auto_gen=$(yq eval '.initialization.auto_generate[]' "$secrets_file" 2>/dev/null)
        if [[ -n "$auto_gen" ]]; then
            echo -e "${BLUE}Auto-generating secrets...${NC}"
            while IFS= read -r gen_yaml; do
                local name=$(echo "$gen_yaml" | yq eval '.name' -)
                local type=$(echo "$gen_yaml" | yq eval '.type' -)
                local path=$(echo "$gen_yaml" | yq eval '.path' - | sed "s/{resource}/$resource_name/g")
                
                if [[ -n "$name" && -n "$type" && -n "$path" ]]; then
                    local value=""
                    case "$type" in
                        uuid)
                            value=$(uuidgen 2>/dev/null || cat /proc/sys/kernel/random/uuid 2>/dev/null || echo "$(date +%s)-$$-$RANDOM")
                            ;;
                        token)
                            value=$(openssl rand -hex 32 2>/dev/null || echo "token-$(date +%s)-$$-$RANDOM")
                            ;;
                        password)
                            value=$(openssl rand -base64 32 2>/dev/null || echo "pass-$(date +%s)-$$-$RANDOM")
                            ;;
                        *)
                            value="auto-${type}-$(date +%s)"
                            ;;
                    esac
                    
                    echo "  Generating $name ($type)..."
                    vault::put_secret "$path" "$value"
                    echo -e "  ${GREEN}✓${NC} Generated and stored: $name"
                fi
            done < <(echo "$auto_gen")
        fi
        
        # Prompt for user-provided secrets
        local prompt_secrets=$(yq eval '.initialization.prompt_user[]' "$secrets_file" 2>/dev/null)
        if [[ -n "$prompt_secrets" ]]; then
            echo
            echo -e "${BLUE}Please provide the following secrets:${NC}"
            while IFS= read -r prompt_yaml; do
                local name=$(echo "$prompt_yaml" | yq eval '.name' -)
                local prompt_text=$(echo "$prompt_yaml" | yq eval '.prompt' -)
                
                # Find the secret details in the main secrets section
                local secret_path=$(yq eval ".secrets | .. | select(.name == \"$name\") | .path" "$secrets_file" | sed "s/{resource}/$resource_name/g" | head -1)
                
                if [[ -n "$name" && -n "$secret_path" ]]; then
                    echo
                    echo "$prompt_text"
                    echo -n "Enter value for $name: "
                    read -rs secret_value
                    echo
                    
                    if [[ -n "$secret_value" ]]; then
                        # Store in Vault
                        vault::put_secret "$secret_path" "$secret_value" >/dev/null
                        echo -e "${GREEN}✓${NC} Stored secret: $name"
                    else
                        echo -e "${YELLOW}Skipped: $name${NC}"
                    fi
                fi
            done < <(echo "$prompt_secrets")
        fi
    else
        echo -e "${YELLOW}Warning: yq not installed, manual initialization required${NC}"
        echo "Please install yq for interactive initialization"
        return 1
    fi
    
    echo
    echo -e "${GREEN}Initialization complete!${NC}"
    echo "Run 'resource-vault secrets check ${resource_name}' to verify"
    
    return 0
}

# Validate all resource secrets
secrets_validate() {
    local resources_dir="${VROOLI_ROOT:-$HOME/Vrooli}/resources"
    local all_valid=true
    local checked_count=0
    local valid_count=0
    
    echo -e "${BLUE}Validating all resource secrets...${NC}"
    echo
    
    # Find all resources with secrets.yaml
    while IFS= read -r secrets_file; do
        if [[ -f "$secrets_file" ]]; then
            local resource_name=$(basename "$(dirname "$(dirname "$secrets_file")")")
            ((checked_count++))
            
            echo -e "${BLUE}Checking ${resource_name}...${NC}"
            if secrets_check "$resource_name" >/dev/null 2>&1; then
                echo -e "  ${GREEN}✓${NC} All required secrets present"
                ((valid_count++))
            else
                echo -e "  ${RED}✗${NC} Missing required secrets"
                all_valid=false
            fi
        fi
    done < <(find "$resources_dir" -type f -path "*/config/secrets.yaml" 2>/dev/null)
    
    echo
    echo -e "${BLUE}Validation Summary:${NC}"
    echo "  Resources checked: ${checked_count}"
    echo "  Fully configured: ${valid_count}"
    echo "  Need configuration: $((checked_count - valid_count))"
    
    if [[ "$all_valid" == "true" ]]; then
        echo -e "${GREEN}All resources have required secrets configured!${NC}"
        return 0
    else
        echo -e "${YELLOW}Some resources need secret configuration${NC}"
        return 1
    fi
}

# Export secrets as environment variables
secrets_export() {
    local resource_name="${1:-}"
    
    if [[ -z "$resource_name" ]]; then
        echo -e "${RED}Error: Resource name required${NC}"
        echo "Usage: resource-vault secrets export <resource-name>"
        return 1
    fi
    
    local resources_dir="${VROOLI_ROOT:-$HOME/Vrooli}/resources"
    local secrets_file="$resources_dir/$resource_name/config/secrets.yaml"
    
    if [[ ! -f "$secrets_file" ]]; then
        echo "# No secrets.yaml found for resource '$resource_name'"
        return 1
    fi
    
    # Ensure Vault is ready
    if ! vault_is_ready; then
        echo "# Error: Vault is not ready"
        return 1
    fi
    
    echo "# Vault secrets for resource: $resource_name"
    echo "# Generated: $(date)"
    echo
    
    # Parse secrets and generate export commands
    if command -v yq &>/dev/null; then
        for category in api_keys database tokens certificates; do
            local secrets=$(yq eval ".secrets.${category}[]" "$secrets_file" 2>/dev/null)
            if [[ -n "$secrets" ]]; then
                while IFS= read -r secret_yaml; do
                    local name=$(echo "$secret_yaml" | yq eval '.name' -)
                    local path=$(echo "$secret_yaml" | yq eval '.path' - | sed "s/{resource}/$resource_name/g")
                    local env_var=$(echo "$secret_yaml" | yq eval '.default_env // ""' -)
                    
                    if [[ -n "$name" && -n "$path" && -n "$env_var" ]]; then
                        # Fetch from Vault
                        local value=$(vault::get_secret "$path" 2>/dev/null)
                        if [[ -n "$value" ]]; then
                            echo "export ${env_var}='${value}'"
                        else
                            echo "# ${env_var} not found in Vault at $path"
                        fi
                    fi
                done < <(yq eval ".secrets.${category}[]" "$secrets_file" 2>/dev/null)
            fi
        done
    else
        echo "# Error: yq not installed, cannot parse secrets.yaml"
        return 1
    fi
    
    return 0
}

# Create a secrets.yaml template for a resource
secrets_create_template() {
    local resource_name="${1:-}"
    
    if [[ -z "$resource_name" ]]; then
        echo -e "${RED}Error: Resource name required${NC}"
        return 1
    fi
    
    local resources_dir="${VROOLI_ROOT:-$HOME/Vrooli}/resources"
    local resource_dir="$resources_dir/$resource_name"
    local config_dir="$resource_dir/config"
    local secrets_file="$config_dir/secrets.yaml"
    
    if [[ ! -d "$resource_dir" ]]; then
        echo -e "${RED}Error: Resource directory not found: $resource_dir${NC}"
        return 1
    fi
    
    # Create config directory if it doesn't exist
    mkdir -p "$config_dir"
    
    if [[ -f "$secrets_file" ]]; then
        echo -e "${YELLOW}Warning: secrets.yaml already exists for $resource_name${NC}"
        echo "Overwrite? (y/n)"
        read -r response
        if [[ "$response" != "y" ]]; then
            return 1
        fi
    fi
    
    # Detect resource type and create appropriate template
    local resource_type="generic"
    
    # Check if it's an AI resource
    if [[ -f "$config_dir/defaults.sh" ]]; then
        if grep -q "API_KEY\|TOKEN" "$config_dir/defaults.sh" 2>/dev/null; then
            resource_type="api"
        elif grep -q "DATABASE\|POSTGRES\|MYSQL" "$config_dir/defaults.sh" 2>/dev/null; then
            resource_type="database"
        fi
    fi
    
    echo -e "${BLUE}Creating secrets.yaml template for ${resource_name} (type: ${resource_type})...${NC}"
    
    case "$resource_type" in
        api)
            cat > "$secrets_file" << EOF
# Secrets configuration for $resource_name resource
version: "1.0"
resource: "$resource_name"
description: "API credentials and tokens for $resource_name service"

secrets:
  api_keys:
    - name: "primary_api_key"
      path: "secret/resources/{resource}/api/primary"
      description: "Primary API key for authentication"
      required: true
      format: "string"
      default_env: "$(echo "$resource_name" | tr '[:lower:]' '[:upper:]')_API_KEY"
      example: "sk-proj-..."

initialization:
  prompt_user:
    - name: "primary_api_key"
      prompt: "Enter your $resource_name API key"
      validation: "Test API connectivity"

health_check:
  required_secrets: ["primary_api_key"]
EOF
            ;;
            
        database)
            cat > "$secrets_file" << EOF
# Secrets configuration for $resource_name resource
version: "1.0"
resource: "$resource_name"
description: "Database credentials for $resource_name"

secrets:
  database:
    - name: "admin_password"
      path: "secret/resources/{resource}/db/admin_password"
      description: "Database administrator password"
      required: true
      format: "string"
      default_env: "$(echo "$resource_name" | tr '[:lower:]' '[:upper:]')_PASSWORD"
    
    - name: "connection_string"
      path: "secret/resources/{resource}/db/connection"
      description: "Full database connection string"
      required: false
      format: "string"
      default_env: "$(echo "$resource_name" | tr '[:lower:]' '[:upper:]')_CONNECTION_STRING"

initialization:
  auto_generate:
    - name: "admin_password"
      type: "password"
      path: "secret/resources/{resource}/db/admin_password"

health_check:
  required_secrets: ["admin_password"]
EOF
            ;;
            
        *)
            cat > "$secrets_file" << EOF
# Secrets configuration for $resource_name resource
version: "1.0"
resource: "$resource_name"
description: "Secrets and credentials for $resource_name"

secrets:
  # Add your secret categories here
  # Example categories: api_keys, database, tokens, certificates
  
  # Example API key:
  # api_keys:
  #   - name: "service_api_key"
  #     path: "secret/resources/{resource}/api/main"
  #     description: "Main service API key"
  #     required: true
  #     format: "string"
  #     default_env: "$(echo "$resource_name" | tr '[:lower:]' '[:upper:]')_API_KEY"
  #     example: "your-api-key-here"

# Optional: Auto-generation configuration
# initialization:
#   auto_generate:
#     - name: "internal_token"
#       type: "uuid"
#       path: "secret/resources/{resource}/internal/token"
#   
#   prompt_user:
#     - name: "service_api_key"
#       prompt: "Enter your API key"
#       validation: "Test connectivity"

# Optional: Health check configuration
# health_check:
#   required_secrets: ["service_api_key"]
EOF
            ;;
    esac
    
    echo -e "${GREEN}✓${NC} Created secrets.yaml template at: $secrets_file"
    echo
    echo "Next steps:"
    echo "1. Edit $secrets_file to define your secrets"
    echo "2. Run 'resource-vault secrets init $resource_name' to set secrets"
    echo "3. Run 'resource-vault secrets check $resource_name' to verify"
    
    return 0
}

# Discover potentially hardcoded secrets in a resource
secrets_discover() {
    local resource_name="${1:-}"
    
    if [[ -z "$resource_name" ]]; then
        echo -e "${RED}Error: Resource name required${NC}"
        echo "Usage: resource-vault secrets discover <resource-name>"
        return 1
    fi
    
    local resources_dir="${VROOLI_ROOT:-$HOME/Vrooli}/resources"
    local resource_dir="$resources_dir/$resource_name"
    
    if [[ ! -d "$resource_dir" ]]; then
        echo -e "${RED}Error: Resource not found: $resource_name${NC}"
        return 1
    fi
    
    echo -e "${BLUE}Discovering potential secrets in ${resource_name}...${NC}"
    echo
    
    local found_issues=false
    
    # Common secret patterns to look for
    local patterns=(
        "API_KEY.*=.*['\"].*['\"]"
        "SECRET.*=.*['\"].*['\"]"
        "TOKEN.*=.*['\"].*['\"]"
        "PASSWORD.*=.*['\"].*['\"]"
        "PRIVATE_KEY.*=.*['\"].*['\"]"
        "sk-[a-zA-Z0-9]+"
        "ey[A-Za-z0-9-_]+"  # JWT tokens
    )
    
    for pattern in "${patterns[@]}"; do
        local matches=$(grep -r -E "$pattern" "$resource_dir" \
            --exclude-dir=.git \
            --exclude-dir=node_modules \
            --exclude="*.md" \
            --exclude="secrets.yaml" \
            2>/dev/null | head -5)
        
        if [[ -n "$matches" ]]; then
            if [[ "$found_issues" == "false" ]]; then
                echo -e "${YELLOW}Potential secrets found:${NC}"
                found_issues=true
            fi
            echo "$matches" | while IFS= read -r match; do
                echo "  ${match}"
            done
            echo
        fi
    done
    
    if [[ "$found_issues" == "false" ]]; then
        echo -e "${GREEN}✓ No obvious hardcoded secrets found${NC}"
    else
        echo -e "${YELLOW}Recommendation:${NC}"
        echo "1. Move these secrets to Vault"
        echo "2. Create config/secrets.yaml to declare them"
        echo "3. Use environment variables or Vault API to access them"
    fi
    
    return 0
}

# Helper function to check if Vault is ready
vault_is_ready() {
    # First check if container is running
    if ! docker ps --format "table {{.Names}}" | grep -q "^vault$"; then
        return 1
    fi
    
    # For dev mode, we just need the container to be healthy
    # Dev mode is always unsealed
    local vault_addr="${VAULT_ADDR:-http://localhost:8200}"
    local vault_token="${VAULT_TOKEN:-myroot}"
    
    # Simple health check with timeout
    if timeout 2 curl -sf "${vault_addr}/v1/sys/health" &>/dev/null; then
        return 0
    fi
    
    return 1
}

# Main secrets command dispatcher
vault_secrets_command() {
    local subcommand="${1:-}"
    shift || true
    
    case "$subcommand" in
        scan)
            secrets_scan "$@"
            ;;
        check)
            secrets_check "$@"
            ;;
        init)
            secrets_init "$@"
            ;;
        validate)
            secrets_validate "$@"
            ;;
        export)
            secrets_export "$@"
            ;;
        create-template)
            secrets_create_template "$@"
            ;;
        discover)
            secrets_discover "$@"
            ;;
        help|"")
            cat << EOF
Vault Secrets Management Commands

Usage: resource-vault secrets <subcommand> [options]

Subcommands:
  scan                    Scan all resources for secrets.yaml files
  check <resource>        Check secrets status for a specific resource
  init <resource>         Initialize secrets for a resource (interactive)
  validate                Validate all resource secrets are configured
  export <resource>       Export secrets as environment variables
  create-template <res>   Create a secrets.yaml template for a resource
  discover <resource>     Find potentially hardcoded secrets in resource code
  help                    Show this help message

Examples:
  resource-vault secrets scan
  resource-vault secrets check openrouter
  resource-vault secrets init postgres
  resource-vault secrets export n8n > n8n-env.sh
  resource-vault secrets create-template my-resource

For more information, see: resources/vault/docs/SECRETS-STANDARD.md
EOF
            ;;
        *)
            echo -e "${RED}Unknown subcommand: $subcommand${NC}"
            echo "Run 'resource-vault secrets help' for usage"
            return 1
            ;;
    esac
}

# Export the main function for use in CLI
export -f vault_secrets_command