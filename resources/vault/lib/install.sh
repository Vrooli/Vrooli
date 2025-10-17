#!/usr/bin/env bash
# Vault Installation Functions
# Installation, initialization, and setup operations

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../../.." && builtin pwd)}"
VAULT_LIB_DIR="${APP_ROOT}/resources/vault/lib"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_TRASH_FILE}"
# shellcheck disable=SC1091
source "${VAULT_LIB_DIR}/docker.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/resources/vault/config/messages.sh"
vault::messages::init

#######################################
# Install Vault
#######################################
vault::install() {
    if vault::is_installed; then
        vault::message "info" "MSG_VAULT_ALREADY_INSTALLED"
        return 0
    fi
    
    vault::message "info" "MSG_VAULT_INSTALL_START"
    
    # Check prerequisites
    if ! vault::docker::check_prerequisites; then
        vault::message "error" "MSG_VAULT_INSTALL_FAILED"
        return 1
    fi
    
    # Start container (which will pull image if needed)
    vault::docker::start_container
    
    if [[ $? -eq 0 ]]; then
        vault::message "success" "MSG_VAULT_INSTALL_SUCCESS"
        
        # Show mode-specific information
        if [[ "$VAULT_MODE" == "dev" ]]; then
            vault::message "info" "MSG_VAULT_DEV_TOKEN_INFO"
            vault::message "info" "MSG_VAULT_DEV_UNSEAL_INFO"
        fi
        
        # Auto-install CLI if available
        # shellcheck disable=SC1091
        "${var_SCRIPTS_RESOURCES_LIB_DIR}/install-resource-cli.sh" "${APP_ROOT}/resources/vault" 2>/dev/null || true
        
        return 0
    else
        vault::message "error" "MSG_VAULT_INSTALL_FAILED"
        return 1
    fi
}

#######################################
# Uninstall Vault
#######################################
vault::uninstall() {
    if ! vault::is_installed; then
        vault::message "info" "MSG_VAULT_NOT_INSTALLED"
        return 0
    fi
    
    vault::message "info" "MSG_VAULT_UNINSTALL_UNINSTALLING"
    
    # Stop and remove container
    vault::docker::cleanup
    
    # Ask about data removal
    if [[ "${VAULT_REMOVE_DATA:-}" == "yes" ]] || resources::confirm "Remove Vault data directories?"; then
        log::info "Removing Vault data directories..."
        trash::safe_remove "$VAULT_DATA_DIR" --production 2>/dev/null || true
        trash::safe_remove "$VAULT_CONFIG_DIR" --production 2>/dev/null || true
        trash::safe_remove "$VAULT_LOGS_DIR" --production 2>/dev/null || true
    fi
    
    vault::message "success" "MSG_VAULT_UNINSTALL_SUCCESS"
}

#######################################
# Initialize Vault in development mode
#######################################
vault::init_dev() {
    # Set development mode
    export VAULT_MODE="dev"
    vault::export_config
    
    # Install if not already installed
    if ! vault::is_installed; then
        vault::install
    elif ! vault::is_running; then
        vault::docker::start_container
    fi
    
    # Wait for Vault to be ready
    if ! vault::wait_for_health; then
        vault::message "error" "MSG_VAULT_INIT_FAILED"
        return 1
    fi
    
    vault::message "info" "MSG_VAULT_INIT_STARTING"
    
    # Development mode is auto-initialized and unsealed
    if vault::is_healthy && ! vault::is_sealed; then
        vault::message "success" "MSG_VAULT_INIT_SUCCESS"
        
        # Enable secret engines and create default paths
        vault::setup_secret_engines
        vault::create_default_paths
        
        # Show integration information
        vault::show_integration_info
        
        return 0
    else
        vault::message "error" "MSG_VAULT_INIT_FAILED"
        return 1
    fi
}

#######################################
# Initialize Vault in production mode
#######################################
vault::init_prod() {
    # Set production mode
    export VAULT_MODE="prod"
    vault::export_config
    
    # Install if not already installed
    if ! vault::is_installed; then
        vault::install
    elif ! vault::is_running; then
        vault::docker::start_container
    fi
    
    # Wait for Vault to be ready
    if ! vault::wait_for_health; then
        vault::message "error" "MSG_VAULT_INIT_FAILED"
        return 1
    fi
    
    # Check if already initialized
    if vault::is_initialized; then
        vault::message "info" "MSG_VAULT_ALREADY_INITIALIZED"
        
        # Still need to setup additional features if not already done
        vault::enable_audit_logging
        vault::setup_approle_auth
        vault::setup_secret_rotation
        vault::setup_auto_unseal
        
        return 0
    fi
    
    vault::message "info" "MSG_VAULT_INIT_STARTING"
    
    # Setup auto-unseal before initialization if configured
    vault::setup_auto_unseal
    
    # Initialize Vault
    local init_response
    init_response=$(vault::api_request "POST" "/v1/sys/init" '{
        "secret_shares": 5,
        "secret_threshold": 3
    }')
    
    if [[ $? -eq 0 ]]; then
        # Save unseal keys and root token
        echo "$init_response" | jq -r '.keys[]' > "$VAULT_UNSEAL_KEYS_FILE"
        echo "$init_response" | jq -r '.root_token' > "$VAULT_TOKEN_FILE"
        
        # Secure the files
        chmod 600 "$VAULT_UNSEAL_KEYS_FILE" "$VAULT_TOKEN_FILE"
        
        vault::message "success" "MSG_VAULT_INIT_SUCCESS"
        vault::message "warn" "MSG_VAULT_SECURITY_WARNING"
        vault::message "info" "MSG_VAULT_TOKEN_LOCATION"
        vault::message "info" "MSG_VAULT_UNSEAL_KEYS_LOCATION"
        
        # Unseal Vault
        vault::unseal
        
        # Enable secret engines and create default paths
        vault::setup_secret_engines
        vault::create_default_paths
        
        # Setup production features
        vault::enable_audit_logging
        vault::setup_approle_auth
        vault::setup_secret_rotation
        
        return 0
    else
        vault::message "error" "MSG_VAULT_INIT_FAILED"
        return 1
    fi
}

#######################################
# Unseal Vault (production mode)
#######################################
vault::unseal() {
    if [[ "$VAULT_MODE" == "dev" ]]; then
        vault::message "info" "MSG_VAULT_DEV_UNSEAL_INFO"
        return 0
    fi
    
    if ! vault::is_initialized; then
        log::error "Vault is not initialized"
        return 1
    fi
    
    if ! vault::is_sealed; then
        vault::message "info" "MSG_VAULT_ALREADY_UNSEALED"
        return 0
    fi
    
    if [[ ! -f "$VAULT_UNSEAL_KEYS_FILE" ]]; then
        log::error "Unseal keys file not found: $VAULT_UNSEAL_KEYS_FILE"
        return 1
    fi
    
    vault::message "info" "MSG_VAULT_UNSEAL_STARTING"
    
    # Read first 3 unseal keys (threshold)
    local unseal_keys
    mapfile -t unseal_keys < <(head -3 "$VAULT_UNSEAL_KEYS_FILE")
    
    # Unseal with each key
    for key in "${unseal_keys[@]}"; do
        if [[ -n "$key" ]]; then
            vault::api_request "POST" "/v1/sys/unseal" "{\"key\": \"$key\"}" >/dev/null
        fi
    done
    
    # Check if unsealed
    if ! vault::is_sealed; then
        vault::message "success" "MSG_VAULT_UNSEAL_SUCCESS"
        return 0
    else
        vault::message "error" "MSG_VAULT_UNSEAL_FAILED"
        return 1
    fi
}

#######################################
# Setup secret engines
#######################################
vault::setup_secret_engines() {
    vault::message "info" "MSG_VAULT_SECRET_ENGINE_ENABLE"
    
    # Enable KV v2 secret engine
    local enable_response
    enable_response=$(vault::api_request "POST" "/v1/sys/mounts/${VAULT_SECRET_ENGINE}" "{
        \"type\": \"kv\",
        \"options\": {
            \"version\": \"${VAULT_SECRET_VERSION}\"
        }
    }" 2>/dev/null)
    
    # Check if already enabled (404 error is expected if already enabled)
    if [[ $? -eq 0 ]] || echo "$enable_response" | grep -q "existing mount"; then
        vault::message "success" "MSG_VAULT_SECRET_ENGINE_SUCCESS"
        return 0
    else
        vault::message "error" "MSG_VAULT_SECRET_ENGINE_FAILED"
        return 1
    fi
}

#######################################
# Create default secret paths
#######################################
vault::create_default_paths() {
    log::info "Creating default secret paths..."
    
    for path in "${VAULT_DEFAULT_PATHS[@]}"; do
        local full_path
        full_path=$(vault::construct_secret_path "$path/.gitkeep")
        
        vault::api_request "POST" "/v1/${full_path}" '{
            "data": {
                ".gitkeep": "This directory is used for organizing secrets"
            }
        }' >/dev/null 2>&1
        
        log::info "Created path: $path"
    done
}

#######################################
# Show integration information
#######################################
vault::show_integration_info() {
    log::info ""
    log::info "🔗 Vault Integration Information:"
    log::info "   Base URL: $VAULT_BASE_URL"
    log::info "   Secret Engine: $VAULT_SECRET_ENGINE"
    log::info "   Namespace: $VAULT_NAMESPACE_PREFIX"
    
    if [[ "$VAULT_MODE" == "dev" ]]; then
        log::info "   Root Token: $VAULT_DEV_ROOT_TOKEN_ID"
    else
        log::info "   Root Token: stored in $VAULT_TOKEN_FILE"
    fi
    
    log::info ""
    log::info "📋 Example Usage:"
    log::info "   Store secret: ./manage.sh --action put-secret --path 'environments/dev/api-key' --value 'secret123'"
    log::info "   Get secret:   ./manage.sh --action get-secret --path 'environments/dev/api-key'"
    log::info "   List secrets: ./manage.sh --action list-secrets --path 'environments/'"
    log::info ""
    
    vault::message "info" "MSG_VAULT_INTEGRATION_N8N"
    vault::message "info" "MSG_VAULT_INTEGRATION_NODERED"
    vault::message "info" "MSG_VAULT_INTEGRATION_AGENT_S2"
}

#######################################
# Migrate environment file to Vault
# Arguments:
#   $1 - environment file path
#   $2 - vault path prefix
#######################################
vault::migrate_env_file() {
    local env_file="" vault_prefix=""
    
    # Support both named and positional arguments
    if [[ "${1:-}" == --* ]]; then
        # Parse named arguments
        while [[ $# -gt 0 ]]; do
            case "$1" in
                --env-file)
                    env_file="$2"
                    shift 2
                    ;;
                --vault-prefix)
                    vault_prefix="$2"
                    shift 2
                    ;;
                *)
                    log::error "Unknown argument: $1"
                    log::error "Usage: vault::migrate_env_file --env-file <file> --vault-prefix <prefix>"
                    return 1
                    ;;
            esac
        done
    else
        # Traditional positional arguments
        env_file="$1"
        vault_prefix="$2"
    fi
    
    if [[ -z "$env_file" ]] || [[ -z "$vault_prefix" ]]; then
        log::error "Usage: vault::migrate_env_file <env_file> <vault_prefix>"
        log::error "   or: vault::migrate_env_file --env-file <file> --vault-prefix <prefix>"
        return 1
    fi
    
    if [[ ! -f "$env_file" ]]; then
        log::error "Environment file not found: $env_file"
        return 1
    fi
    
    vault::message "info" "MSG_VAULT_MIGRATION_START"
    
    local migrated_count=0
    local failed_count=0
    
    # Read and process environment file
    while IFS='=' read -r key value; do
        # Skip empty lines and comments
        [[ -z "$key" ]] || [[ "$key" =~ ^[[:space:]]*# ]] && continue
        
        # Skip lines that don't look like environment variables
        [[ "$key" =~ ^[A-Z_][A-Z0-9_]*$ ]] || continue
        
        # Remove quotes from value
        value=$(echo "$value" | sed 's/^["'\'']*//;s/["'\'']*$//')
        
        # Construct vault path
        local vault_path="${vault_prefix}/${key,,}"  # lowercase key
        
        # Store in Vault
        if vault::put_secret "$vault_path" "$value"; then
            log::info "Migrated: $key -> $vault_path"
            ((migrated_count++))
        else
            log::error "Failed to migrate: $key"
            ((failed_count++))
        fi
        
    done < <(grep -v '^[[:space:]]*$' "$env_file" | grep -v '^[[:space:]]*#')
    
    if [[ $failed_count -eq 0 ]]; then
        vault::message "success" "MSG_VAULT_MIGRATION_SUCCESS"
        log::info "Migrated $migrated_count secrets from $env_file"
        return 0
    else
        vault::message "error" "MSG_VAULT_MIGRATION_FAILED"
        log::error "Failed to migrate $failed_count secrets"
        return 1
    fi
}