#!/usr/bin/env bash
set -euo pipefail

# Vault Secrets Injection Adapter
# This script handles injection of secrets and policies into HashiCorp Vault
# Part of the Vrooli resource data injection system

DESCRIPTION="Inject secrets and policies into HashiCorp Vault"

VAULT_SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
RESOURCES_DIR="${VAULT_SCRIPT_DIR}/../.."

# Source common utilities
# shellcheck disable=SC1091
source "${RESOURCES_DIR}/common.sh"

# Source Vault configuration if available
if [[ -f "${VAULT_SCRIPT_DIR}/config/defaults.sh" ]]; then
    # shellcheck disable=SC1091
    source "${VAULT_SCRIPT_DIR}/config/defaults.sh" 2>/dev/null || true
fi

# Default Vault settings
readonly DEFAULT_VAULT_ADDR="http://localhost:8200"
readonly DEFAULT_VAULT_TOKEN="root"

# Vault settings (can be overridden by environment)
VAULT_ADDR="${VAULT_ADDR:-$DEFAULT_VAULT_ADDR}"
VAULT_TOKEN="${VAULT_TOKEN:-$DEFAULT_VAULT_TOKEN}"

# Operation tracking
declare -a VAULT_ROLLBACK_ACTIONS=()

#######################################
# Display usage information
#######################################
vault_inject::usage() {
    cat << EOF
Vault Secrets Injection Adapter

USAGE:
    $0 [OPTIONS] CONFIG_JSON

DESCRIPTION:
    Injects secrets and policies into HashiCorp Vault based on scenario configuration.
    Supports validation, injection, status checks, and rollback operations.

OPTIONS:
    --validate    Validate the injection configuration
    --inject      Perform the secret injection
    --status      Check status of injected secrets
    --rollback    Rollback injected secrets
    --help        Show this help message

CONFIGURATION FORMAT:
    {
      "secrets": [
        {
          "path": "secret/data/app/config",
          "data": {
            "api_key": "abc123",
            "database_url": "postgres://localhost/db"
          }
        }
      ],
      "policies": [
        {
          "name": "app-policy",
          "rules": "path \\"secret/data/app/*\\" { capabilities = [\\"read\\"] }"
        }
      ],
      "auth": [
        {
          "type": "userpass",
          "username": "app-user",
          "password": "secure-password",
          "policies": ["app-policy"]
        }
      ],
      "engines": [
        {
          "path": "database",
          "type": "database",
          "config": {
            "plugin_name": "postgresql-database-plugin",
            "connection_url": "postgresql://{{username}}:{{password}}@localhost:5432/mydb"
          }
        }
      ]
    }

EXAMPLES:
    # Validate configuration
    $0 --validate '{"secrets": [{"path": "secret/data/test", "data": {"key": "value"}}]}'
    
    # Inject secrets and policies
    $0 --inject '{"secrets": [{"path": "secret/data/app", "data": {"api_key": "xyz"}}], "policies": [{"name": "read-app", "rules": "path \\"secret/data/app\\" { capabilities = [\\"read\\"] }"}]}'

EOF
}

#######################################
# Check if Vault is accessible
# Returns:
#   0 if accessible, 1 otherwise
#######################################
vault_inject::check_accessibility() {
    # Check if Vault is sealed
    local health_response
    health_response=$(curl -s "${VAULT_ADDR}/v1/sys/health" 2>/dev/null)
    
    if [[ $? -ne 0 ]]; then
        log::error "Vault is not accessible at $VAULT_ADDR"
        log::info "Ensure Vault is running: ./scripts/resources/storage/vault/manage.sh --action start"
        return 1
    fi
    
    local sealed
    sealed=$(echo "$health_response" | jq -r '.sealed // true')
    
    if [[ "$sealed" == "true" ]]; then
        log::error "Vault is sealed. Please unseal it first"
        log::info "Use: vault operator unseal <unseal-key>"
        return 1
    fi
    
    log::debug "Vault is accessible and unsealed at $VAULT_ADDR"
    return 0
}

#######################################
# Add rollback action
# Arguments:
#   $1 - description
#   $2 - rollback command
#######################################
vault_inject::add_rollback_action() {
    local description="$1"
    local command="$2"
    
    VAULT_ROLLBACK_ACTIONS+=("$description|$command")
    log::debug "Added Vault rollback action: $description"
}

#######################################
# Execute rollback actions
#######################################
vault_inject::execute_rollback() {
    if [[ ${#VAULT_ROLLBACK_ACTIONS[@]} -eq 0 ]]; then
        log::info "No Vault rollback actions to execute"
        return 0
    fi
    
    log::info "Executing Vault rollback actions..."
    
    local success_count=0
    local total_count=${#VAULT_ROLLBACK_ACTIONS[@]}
    
    # Execute in reverse order
    for ((i=${#VAULT_ROLLBACK_ACTIONS[@]}-1; i>=0; i--)); do
        local action="${VAULT_ROLLBACK_ACTIONS[i]}"
        IFS='|' read -r description command <<< "$action"
        
        log::info "Rollback: $description"
        
        if eval "$command"; then
            success_count=$((success_count + 1))
            log::success "Rollback completed: $description"
        else
            log::error "Rollback failed: $description"
        fi
    done
    
    log::info "Vault rollback completed: $success_count/$total_count actions successful"
    VAULT_ROLLBACK_ACTIONS=()
}

#######################################
# Validate secrets configuration
# Arguments:
#   $1 - secrets configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
vault_inject::validate_secrets() {
    local secrets_config="$1"
    
    log::debug "Validating secrets configurations..."
    
    # Check if secrets is an array
    local secrets_type
    secrets_type=$(echo "$secrets_config" | jq -r 'type')
    
    if [[ "$secrets_type" != "array" ]]; then
        log::error "Secrets configuration must be an array, got: $secrets_type"
        return 1
    fi
    
    # Validate each secret
    local secret_count
    secret_count=$(echo "$secrets_config" | jq 'length')
    
    for ((i=0; i<secret_count; i++)); do
        local secret
        secret=$(echo "$secrets_config" | jq -c ".[$i]")
        
        # Check required fields
        local path data
        path=$(echo "$secret" | jq -r '.path // empty')
        data=$(echo "$secret" | jq -r '.data // empty')
        
        if [[ -z "$path" ]]; then
            log::error "Secret at index $i missing required 'path' field"
            return 1
        fi
        
        if [[ -z "$data" ]] || [[ "$data" == "null" ]]; then
            log::error "Secret at path '$path' missing required 'data' field"
            return 1
        fi
        
        # Validate path format
        if [[ ! "$path" =~ ^[a-zA-Z0-9/_-]+$ ]]; then
            log::error "Invalid secret path: $path (contains invalid characters)"
            return 1
        fi
        
        log::debug "Secret at path '$path' configuration is valid"
    done
    
    log::success "All secrets configurations are valid"
    return 0
}

#######################################
# Validate policies configuration
# Arguments:
#   $1 - policies configuration JSON array
# Returns:
#   0 if valid, 1 if invalid
#######################################
vault_inject::validate_policies() {
    local policies_config="$1"
    
    log::debug "Validating policies configurations..."
    
    # Check if policies is an array
    local policies_type
    policies_type=$(echo "$policies_config" | jq -r 'type')
    
    if [[ "$policies_type" != "array" ]]; then
        log::error "Policies configuration must be an array, got: $policies_type"
        return 1
    fi
    
    # Validate each policy
    local policy_count
    policy_count=$(echo "$policies_config" | jq 'length')
    
    for ((i=0; i<policy_count; i++)); do
        local policy
        policy=$(echo "$policies_config" | jq -c ".[$i]")
        
        # Check required fields
        local name rules
        name=$(echo "$policy" | jq -r '.name // empty')
        rules=$(echo "$policy" | jq -r '.rules // empty')
        
        if [[ -z "$name" ]]; then
            log::error "Policy at index $i missing required 'name' field"
            return 1
        fi
        
        if [[ -z "$rules" ]]; then
            log::error "Policy '$name' missing required 'rules' field"
            return 1
        fi
        
        # Validate policy name
        if [[ ! "$name" =~ ^[a-zA-Z0-9_-]+$ ]]; then
            log::error "Invalid policy name: $name (contains invalid characters)"
            return 1
        fi
        
        log::debug "Policy '$name' configuration is valid"
    done
    
    log::success "All policies configurations are valid"
    return 0
}

#######################################
# Validate injection configuration
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if valid, 1 if invalid
#######################################
vault_inject::validate_config() {
    local config="$1"
    
    log::info "Validating Vault injection configuration..."
    
    # Basic JSON validation
    if ! echo "$config" | jq . >/dev/null 2>&1; then
        log::error "Invalid JSON in Vault injection configuration"
        return 1
    fi
    
    # Check for at least one injection type
    local has_secrets has_policies has_auth has_engines
    has_secrets=$(echo "$config" | jq -e '.secrets' >/dev/null 2>&1 && echo "true" || echo "false")
    has_policies=$(echo "$config" | jq -e '.policies' >/dev/null 2>&1 && echo "true" || echo "false")
    has_auth=$(echo "$config" | jq -e '.auth' >/dev/null 2>&1 && echo "true" || echo "false")
    has_engines=$(echo "$config" | jq -e '.engines' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_secrets" == "false" && "$has_policies" == "false" && "$has_auth" == "false" && "$has_engines" == "false" ]]; then
        log::error "Vault injection configuration must have 'secrets', 'policies', 'auth', or 'engines'"
        return 1
    fi
    
    # Validate secrets if present
    if [[ "$has_secrets" == "true" ]]; then
        local secrets
        secrets=$(echo "$config" | jq -c '.secrets')
        
        if ! vault_inject::validate_secrets "$secrets"; then
            return 1
        fi
    fi
    
    # Validate policies if present
    if [[ "$has_policies" == "true" ]]; then
        local policies
        policies=$(echo "$config" | jq -c '.policies')
        
        if ! vault_inject::validate_policies "$policies"; then
            return 1
        fi
    fi
    
    log::success "Vault injection configuration is valid"
    return 0
}

#######################################
# Write secret to Vault
# Arguments:
#   $1 - secret configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
vault_inject::write_secret() {
    local secret_config="$1"
    
    local path data
    path=$(echo "$secret_config" | jq -r '.path')
    data=$(echo "$secret_config" | jq -c '.data')
    
    log::info "Writing secret to path: $path"
    
    # Write secret via Vault API
    local response
    response=$(curl -s -X POST \
        -H "X-Vault-Token: ${VAULT_TOKEN}" \
        -d "{\"data\": $data}" \
        "${VAULT_ADDR}/v1/${path}" 2>&1)
    
    if [[ $? -eq 0 ]]; then
        # Check for errors in response
        if echo "$response" | jq -e '.errors' >/dev/null 2>&1; then
            log::error "Failed to write secret: $(echo "$response" | jq -r '.errors[]')"
            return 1
        else
            log::success "Written secret to path: $path"
            
            # Add rollback action
            vault_inject::add_rollback_action \
                "Delete secret: $path" \
                "curl -s -X DELETE -H 'X-Vault-Token: ${VAULT_TOKEN}' '${VAULT_ADDR}/v1/${path}' >/dev/null 2>&1"
            
            return 0
        fi
    else
        log::error "Failed to write secret to path: $path"
        return 1
    fi
}

#######################################
# Create policy in Vault
# Arguments:
#   $1 - policy configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
vault_inject::create_policy() {
    local policy_config="$1"
    
    local name rules
    name=$(echo "$policy_config" | jq -r '.name')
    rules=$(echo "$policy_config" | jq -r '.rules')
    
    log::info "Creating policy: $name"
    
    # Create policy via Vault API
    local response
    response=$(curl -s -X PUT \
        -H "X-Vault-Token: ${VAULT_TOKEN}" \
        -d "{\"policy\": \"$rules\"}" \
        "${VAULT_ADDR}/v1/sys/policies/acl/${name}" 2>&1)
    
    if [[ $? -eq 0 ]]; then
        # Check for errors in response
        if echo "$response" | jq -e '.errors' >/dev/null 2>&1; then
            log::error "Failed to create policy: $(echo "$response" | jq -r '.errors[]')"
            return 1
        else
            log::success "Created policy: $name"
            
            # Add rollback action
            vault_inject::add_rollback_action \
                "Delete policy: $name" \
                "curl -s -X DELETE -H 'X-Vault-Token: ${VAULT_TOKEN}' '${VAULT_ADDR}/v1/sys/policies/acl/${name}' >/dev/null 2>&1"
            
            return 0
        fi
    else
        log::error "Failed to create policy: $name"
        return 1
    fi
}

#######################################
# Create auth method
# Arguments:
#   $1 - auth configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
vault_inject::create_auth() {
    local auth_config="$1"
    
    local type username password policies
    type=$(echo "$auth_config" | jq -r '.type')
    username=$(echo "$auth_config" | jq -r '.username // empty')
    password=$(echo "$auth_config" | jq -r '.password // empty')
    policies=$(echo "$auth_config" | jq -r '.policies // [] | join(",")')
    
    case "$type" in
        userpass)
            if [[ -z "$username" || -z "$password" ]]; then
                log::error "Userpass auth requires username and password"
                return 1
            fi
            
            log::info "Creating userpass auth for user: $username"
            
            # Enable userpass auth if not already enabled
            curl -s -X POST \
                -H "X-Vault-Token: ${VAULT_TOKEN}" \
                -d '{"type": "userpass"}' \
                "${VAULT_ADDR}/v1/sys/auth/userpass" >/dev/null 2>&1
            
            # Create user
            local response
            response=$(curl -s -X POST \
                -H "X-Vault-Token: ${VAULT_TOKEN}" \
                -d "{\"password\": \"$password\", \"policies\": \"$policies\"}" \
                "${VAULT_ADDR}/v1/auth/userpass/users/${username}" 2>&1)
            
            if [[ $? -eq 0 ]]; then
                log::success "Created userpass auth for user: $username"
                
                # Add rollback action
                vault_inject::add_rollback_action \
                    "Delete user: $username" \
                    "curl -s -X DELETE -H 'X-Vault-Token: ${VAULT_TOKEN}' '${VAULT_ADDR}/v1/auth/userpass/users/${username}' >/dev/null 2>&1"
                
                return 0
            else
                log::error "Failed to create userpass auth for user: $username"
                return 1
            fi
            ;;
        *)
            log::warn "Auth type '$type' not yet supported"
            return 0
            ;;
    esac
}

#######################################
# Mount secret engine
# Arguments:
#   $1 - engine configuration JSON object
# Returns:
#   0 if successful, 1 if failed
#######################################
vault_inject::mount_engine() {
    local engine_config="$1"
    
    local path type config
    path=$(echo "$engine_config" | jq -r '.path')
    type=$(echo "$engine_config" | jq -r '.type')
    config=$(echo "$engine_config" | jq -c '.config // {}')
    
    log::info "Mounting secret engine '$type' at path: $path"
    
    # Mount secret engine via Vault API
    local mount_data=$(jq -n \
        --arg type "$type" \
        --argjson config "$config" \
        '{type: $type, config: $config}')
    
    local response
    response=$(curl -s -X POST \
        -H "X-Vault-Token: ${VAULT_TOKEN}" \
        -d "$mount_data" \
        "${VAULT_ADDR}/v1/sys/mounts/${path}" 2>&1)
    
    if [[ $? -eq 0 ]]; then
        # Check for errors in response
        if echo "$response" | jq -e '.errors' >/dev/null 2>&1; then
            # Check if already mounted
            if echo "$response" | grep -q "path is already in use"; then
                log::warn "Secret engine already mounted at path: $path"
                return 0
            else
                log::error "Failed to mount engine: $(echo "$response" | jq -r '.errors[]')"
                return 1
            fi
        else
            log::success "Mounted secret engine '$type' at path: $path"
            
            # Add rollback action
            vault_inject::add_rollback_action \
                "Unmount engine: $path" \
                "curl -s -X DELETE -H 'X-Vault-Token: ${VAULT_TOKEN}' '${VAULT_ADDR}/v1/sys/mounts/${path}' >/dev/null 2>&1"
            
            return 0
        fi
    else
        log::error "Failed to mount secret engine at path: $path"
        return 1
    fi
}

#######################################
# Inject secrets
# Arguments:
#   $1 - secrets configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
vault_inject::inject_secrets() {
    local secrets_config="$1"
    
    log::info "Writing secrets to Vault..."
    
    local secret_count
    secret_count=$(echo "$secrets_config" | jq 'length')
    
    if [[ "$secret_count" -eq 0 ]]; then
        log::info "No secrets to write"
        return 0
    fi
    
    local failed_secrets=()
    
    for ((i=0; i<secret_count; i++)); do
        local secret
        secret=$(echo "$secrets_config" | jq -c ".[$i]")
        
        local secret_path
        secret_path=$(echo "$secret" | jq -r '.path')
        
        if ! vault_inject::write_secret "$secret"; then
            failed_secrets+=("$secret_path")
        fi
    done
    
    if [[ ${#failed_secrets[@]} -eq 0 ]]; then
        log::success "All secrets written successfully"
        return 0
    else
        log::error "Failed to write secrets: ${failed_secrets[*]}"
        return 1
    fi
}

#######################################
# Inject policies
# Arguments:
#   $1 - policies configuration JSON array
# Returns:
#   0 if successful, 1 if failed
#######################################
vault_inject::inject_policies() {
    local policies_config="$1"
    
    log::info "Creating Vault policies..."
    
    local policy_count
    policy_count=$(echo "$policies_config" | jq 'length')
    
    if [[ "$policy_count" -eq 0 ]]; then
        log::info "No policies to create"
        return 0
    fi
    
    local failed_policies=()
    
    for ((i=0; i<policy_count; i++)); do
        local policy
        policy=$(echo "$policies_config" | jq -c ".[$i]")
        
        local policy_name
        policy_name=$(echo "$policy" | jq -r '.name')
        
        if ! vault_inject::create_policy "$policy"; then
            failed_policies+=("$policy_name")
        fi
    done
    
    if [[ ${#failed_policies[@]} -eq 0 ]]; then
        log::success "All policies created successfully"
        return 0
    else
        log::error "Failed to create policies: ${failed_policies[*]}"
        return 1
    fi
}

#######################################
# Perform data injection
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
vault_inject::inject_data() {
    local config="$1"
    
    log::header "🔄 Injecting data into Vault"
    
    # Check Vault accessibility
    if ! vault_inject::check_accessibility; then
        return 1
    fi
    
    # Clear previous rollback actions
    VAULT_ROLLBACK_ACTIONS=()
    
    # Mount engines if present
    local has_engines
    has_engines=$(echo "$config" | jq -e '.engines' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_engines" == "true" ]]; then
        local engines
        engines=$(echo "$config" | jq -c '.engines')
        
        local engine_count
        engine_count=$(echo "$engines" | jq 'length')
        
        for ((i=0; i<engine_count; i++)); do
            local engine
            engine=$(echo "$engines" | jq -c ".[$i]")
            
            if ! vault_inject::mount_engine "$engine"; then
                log::error "Failed to mount secret engine"
                vault_inject::execute_rollback
                return 1
            fi
        done
    fi
    
    # Inject policies if present
    local has_policies
    has_policies=$(echo "$config" | jq -e '.policies' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_policies" == "true" ]]; then
        local policies
        policies=$(echo "$config" | jq -c '.policies')
        
        if ! vault_inject::inject_policies "$policies"; then
            log::error "Failed to create policies"
            vault_inject::execute_rollback
            return 1
        fi
    fi
    
    # Inject secrets if present
    local has_secrets
    has_secrets=$(echo "$config" | jq -e '.secrets' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_secrets" == "true" ]]; then
        local secrets
        secrets=$(echo "$config" | jq -c '.secrets')
        
        if ! vault_inject::inject_secrets "$secrets"; then
            log::error "Failed to write secrets"
            vault_inject::execute_rollback
            return 1
        fi
    fi
    
    # Create auth methods if present
    local has_auth
    has_auth=$(echo "$config" | jq -e '.auth' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_auth" == "true" ]]; then
        local auth_methods
        auth_methods=$(echo "$config" | jq -c '.auth')
        
        local auth_count
        auth_count=$(echo "$auth_methods" | jq 'length')
        
        for ((i=0; i<auth_count; i++)); do
            local auth
            auth=$(echo "$auth_methods" | jq -c ".[$i]")
            
            if ! vault_inject::create_auth "$auth"; then
                log::error "Failed to create auth method"
                vault_inject::execute_rollback
                return 1
            fi
        done
    fi
    
    log::success "✅ Vault data injection completed"
    return 0
}

#######################################
# Check injection status
# Arguments:
#   $1 - configuration JSON
# Returns:
#   0 if successful, 1 if failed
#######################################
vault_inject::check_status() {
    local config="$1"
    
    log::header "📊 Checking Vault injection status"
    
    # Check Vault accessibility
    if ! vault_inject::check_accessibility; then
        return 1
    fi
    
    # Check secrets status
    local has_secrets
    has_secrets=$(echo "$config" | jq -e '.secrets' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_secrets" == "true" ]]; then
        local secrets
        secrets=$(echo "$config" | jq -c '.secrets')
        
        log::info "Checking secrets status..."
        
        local secret_count
        secret_count=$(echo "$secrets" | jq 'length')
        
        for ((i=0; i<secret_count; i++)); do
            local secret
            secret=$(echo "$secrets" | jq -c ".[$i]")
            
            local path
            path=$(echo "$secret" | jq -r '.path')
            
            # Check if secret exists
            local response
            response=$(curl -s -H "X-Vault-Token: ${VAULT_TOKEN}" \
                "${VAULT_ADDR}/v1/${path}" 2>/dev/null)
            
            if echo "$response" | jq -e '.data' >/dev/null 2>&1; then
                log::success "✅ Secret exists at path: $path"
            else
                log::error "❌ Secret not found at path: $path"
            fi
        done
    fi
    
    # Check policies status
    local has_policies
    has_policies=$(echo "$config" | jq -e '.policies' >/dev/null 2>&1 && echo "true" || echo "false")
    
    if [[ "$has_policies" == "true" ]]; then
        local policies
        policies=$(echo "$config" | jq -c '.policies')
        
        log::info "Checking policies status..."
        
        local policy_count
        policy_count=$(echo "$policies" | jq 'length')
        
        for ((i=0; i<policy_count; i++)); do
            local policy
            policy=$(echo "$policies" | jq -c ".[$i]")
            
            local name
            name=$(echo "$policy" | jq -r '.name')
            
            # Check if policy exists
            local response
            response=$(curl -s -H "X-Vault-Token: ${VAULT_TOKEN}" \
                "${VAULT_ADDR}/v1/sys/policies/acl/${name}" 2>/dev/null)
            
            if echo "$response" | jq -e '.policy' >/dev/null 2>&1; then
                log::success "✅ Policy exists: $name"
            else
                log::error "❌ Policy not found: $name"
            fi
        done
    fi
    
    return 0
}

#######################################
# Main execution function
#######################################
vault_inject::main() {
    local action="$1"
    local config="${2:-}"
    
    if [[ -z "$config" ]]; then
        log::error "Configuration JSON required"
        vault_inject::usage
        exit 1
    fi
    
    case "$action" in
        "--validate")
            vault_inject::validate_config "$config"
            ;;
        "--inject")
            vault_inject::inject_data "$config"
            ;;
        "--status")
            vault_inject::check_status "$config"
            ;;
        "--rollback")
            vault_inject::execute_rollback
            ;;
        "--help")
            vault_inject::usage
            ;;
        *)
            log::error "Unknown action: $action"
            vault_inject::usage
            exit 1
            ;;
    esac
}

# Execute main function if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    if [[ $# -eq 0 ]]; then
        vault_inject::usage
        exit 1
    fi
    
    vault_inject::main "$@"
fi