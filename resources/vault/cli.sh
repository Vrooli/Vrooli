#!/usr/bin/env bash
################################################################################
# Vault Resource CLI
# 
# Lightweight CLI interface for Vault using the CLI Command Framework
#
# Usage:
#   resource-vault <command> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    VAULT_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "$(dirname "$VAULT_CLI_SCRIPT")/../.." && builtin pwd)"
fi
VAULT_CLI_DIR="${APP_ROOT}/resources/vault"

# Source standard variables
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${var_LIB_SERVICE_DIR}/secrets.sh"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"
source "${VAULT_CLI_DIR}/lib/status.sh"

# Source Vault configuration
# shellcheck disable=SC1091
source "${VAULT_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${VAULT_CLI_DIR}/config/messages.sh" 2>/dev/null || true
vault::export_config 2>/dev/null || true
vault::messages::init 2>/dev/null || true

# Source Vault libraries
for lib in common docker api status install; do
    lib_file="${VAULT_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework
cli::init "vault" "HashiCorp Vault secrets management"

# Override help to provide Vault-specific examples
cli::register_command "help" "Show this help message with Vault examples" "vault_show_help"

# Override status command to use vault-specific handler
cli::register_command "status" "Show Vault status" "vault_status"
cli::register_command "start" "Start Vault" "vault_start" "modifies-system"
cli::register_command "stop" "Stop Vault" "vault_stop" "modifies-system"
cli::register_command "install" "Install Vault" "vault_install" "modifies-system"
cli::register_command "validate" "Validate Vault configuration" "vault_validate"

# Register additional Vault-specific commands
cli::register_command "inject" "Store secret in Vault" "vault_inject" "modifies-system"
cli::register_command "init-dev" "Initialize Vault for development" "vault_init_dev" "modifies-system"
cli::register_command "init-prod" "Initialize Vault for production" "vault_init_prod" "modifies-system"
cli::register_command "unseal" "Unseal Vault" "vault_unseal" "modifies-system"
cli::register_command "get-secret" "Get secret from Vault" "vault_get_secret"
cli::register_command "list-secrets" "List secrets at path" "vault_list_secrets"
cli::register_command "delete-secret" "Delete secret (requires --force)" "vault_delete_secret" "modifies-system"
cli::register_command "auth-info" "Show authentication information" "vault_auth_info"
cli::register_command "test" "Run functional tests" "vault_test"
cli::register_command "monitor" "Monitor Vault (default: 5s)" "vault_monitor"
cli::register_command "backup" "Create Vault backup" "vault_backup" "modifies-system"
cli::register_command "restore" "Restore Vault backup" "vault_restore" "modifies-system"
cli::register_command "migrate-env" "Migrate .env file to Vault" "vault_migrate_env" "modifies-system"
cli::register_command "logs" "Show Vault logs (default: 50 lines)" "vault_logs"
cli::register_command "diagnose" "Diagnose Vault issues" "vault_diagnose"
cli::register_command "credentials" "Show n8n credentials for Vault" "vault_credentials"
cli::register_command "uninstall" "Uninstall Vault (requires --force)" "vault_uninstall" "modifies-system"

################################################################################
# Resource-specific command implementations
################################################################################

# Inject secrets into Vault
vault_inject() {
    local path="${1:-}"
    local value="${2:-}"
    local key="${3:-value}"
    
    if [[ -z "$path" || -z "$value" ]]; then
        log::error "Both path and value required for injection"
        echo "Usage: resource-vault inject <path> <value> [key]"
        echo ""
        echo "Examples:"
        echo "  resource-vault inject secret/myapp/db password123"
        echo "  resource-vault inject secret/myapp/api apikey123 api_key"
        echo "  resource-vault inject secret/shared/jwt supersecret token"
        return 1
    fi
    
    # Use existing Vault secret function
    if command -v vault::put_secret &>/dev/null; then
        vault::put_secret "$path" "$value" "$key"
    else
        log::error "vault::put_secret not available"
        return 1
    fi
}

# Validate Vault configuration
vault_validate() {
    if command -v vault::show_status &>/dev/null; then
        vault::show_status
    elif command -v vault::common::is_healthy &>/dev/null; then
        vault::common::is_healthy
    else
        # Basic validation
        log::header "Validating Vault"
        local container_name="${VAULT_CONTAINER_NAME:-vault}"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "$container_name" || {
            log::error "Vault container not running"
            return 1
        }
        log::success "Vault is running"
    fi
}

# Show Vault status
vault_status() {
    if command -v vault::status &>/dev/null; then
        vault::status "$@"
    elif command -v vault::show_status &>/dev/null; then
        vault::show_status "$@"
    else
        # Basic status fallback
        log::header "Vault Status"
        local container_name="${VAULT_CONTAINER_NAME:-vault}"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "$container_name"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=$container_name" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start Vault
vault_start() {
    if command -v vault::docker::start_container &>/dev/null; then
        vault::docker::start_container
    else
        local container_name="${VAULT_CONTAINER_NAME:-vault}"
        docker start "$container_name" || log::error "Failed to start Vault"
    fi
}

# Stop Vault
vault_stop() {
    if command -v vault::docker::stop_container &>/dev/null; then
        vault::docker::stop_container
    else
        local container_name="${VAULT_CONTAINER_NAME:-vault}"
        docker stop "$container_name" || log::error "Failed to stop Vault"
    fi
}

# Install Vault
vault_install() {
    if command -v vault::install &>/dev/null; then
        vault::install
    else
        log::error "vault::install not available"
        return 1
    fi
}

# Uninstall Vault
vault_uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Vault and all its data. Use --force to confirm."
        return 1
    fi
    
    if command -v vault::uninstall &>/dev/null; then
        vault::uninstall
    else
        local container_name="${VAULT_CONTAINER_NAME:-vault}"
        docker stop "$container_name" 2>/dev/null || true
        docker rm "$container_name" 2>/dev/null || true
        log::success "Vault uninstalled"
    fi
}

# Show credentials for n8n integration
vault_credentials() {
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "vault"; return 0; }
        return 1
    fi
    
    local status
    status=$(credentials::get_resource_status "${VAULT_CONTAINER_NAME:-vault}")
    
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Get token for authentication
        local token=""
        if [[ "${VAULT_MODE:-}" == "dev" ]]; then
            # Use development root token
            token="${VAULT_DEV_ROOT_TOKEN_ID:-root}"
        elif [[ -f "${VAULT_TOKEN_FILE:-}" ]]; then
            # Try to read token from file for production mode
            token=$(cat "${VAULT_TOKEN_FILE}" 2>/dev/null || echo "")
        fi
        
        if [[ -n "$token" ]]; then
            # Vault HTTP API connection
            local connection_obj
            connection_obj=$(jq -n \
                --arg host "localhost" \
                --argjson port "${VAULT_PORT:-8200}" \
                --arg path "/v1" \
                --argjson ssl "$([[ "${VAULT_TLS_DISABLE:-1}" == "1" ]] && echo false || echo true)" \
                '{
                    host: $host,
                    port: $port,
                    path: $path,
                    ssl: $ssl
                }')
            
            local auth_obj
            auth_obj=$(jq -n \
                --arg header_name "X-Vault-Token" \
                --arg header_value "$token" \
                '{
                    header_name: $header_name,
                    header_value: $header_value
                }')
            
            local metadata_obj
            metadata_obj=$(jq -n \
                --arg description "HashiCorp Vault secrets management" \
                --arg mode "${VAULT_MODE:-dev}" \
                --arg engine "${VAULT_SECRET_ENGINE:-secret}" \
                --arg version "${VAULT_SECRET_VERSION:-v2}" \
                '{
                    description: $description,
                    mode: $mode,
                    secret_engine: $engine,
                    kv_version: $version
                }')
            
            local connection
            connection=$(credentials::build_connection \
                "main" \
                "Vault API" \
                "httpHeaderAuth" \
                "$connection_obj" \
                "$auth_obj" \
                "$metadata_obj")
            
            connections_array="[$connection]"
        fi
    fi
    
    local response
    response=$(credentials::build_response "vault" "$status" "$connections_array")
    credentials::format_output "$response"
}

# Initialize Vault for development
vault_init_dev() {
    if command -v vault::init_dev &>/dev/null; then
        vault::init_dev
    else
        log::error "Vault dev initialization not available"
        return 1
    fi
}

# Initialize Vault for production
vault_init_prod() {
    if command -v vault::init_prod &>/dev/null; then
        vault::init_prod
    else
        log::error "Vault production initialization not available"
        return 1
    fi
}

# Unseal Vault
vault_unseal() {
    if command -v vault::unseal &>/dev/null; then
        vault::unseal
    else
        log::error "Vault unseal not available"
        return 1
    fi
}

# Get secret from Vault
vault_get_secret() {
    local path="${1:-}"
    local key="${2:-value}"
    local format="${3:-raw}"
    
    if [[ -z "$path" ]]; then
        log::error "Secret path required"
        echo "Usage: resource-vault get-secret <path> [key] [format]"
        echo ""
        echo "Examples:"
        echo "  resource-vault get-secret secret/myapp/db"
        echo "  resource-vault get-secret secret/myapp/api api_key"
        echo "  resource-vault get-secret secret/shared/jwt token json"
        return 1
    fi
    
    if command -v vault::get_secret &>/dev/null; then
        vault::get_secret "$path" "$key" "$format"
    else
        log::error "vault::get_secret not available"
        return 1
    fi
}

# List secrets at path
vault_list_secrets() {
    local path="${1:-}"
    local format="${2:-list}"
    
    if [[ -z "$path" ]]; then
        log::error "Secret path required"
        echo "Usage: resource-vault list-secrets <path> [format]"
        echo ""
        echo "Examples:"
        echo "  resource-vault list-secrets secret/myapp"
        echo "  resource-vault list-secrets secret/shared json"
        return 1
    fi
    
    if command -v vault::list_secrets &>/dev/null; then
        vault::list_secrets "$path" "$format"
    else
        log::error "vault::list_secrets not available"
        return 1
    fi
}

# Delete secret from Vault
vault_delete_secret() {
    local path="${1:-}"
    
    if [[ -z "$path" ]]; then
        log::error "Secret path required"
        echo "Usage: resource-vault delete-secret <path> --force"
        echo ""
        echo "Examples:"
        echo "  resource-vault delete-secret secret/myapp/old-key --force"
        echo "  resource-vault delete-secret secret/temp/test-data --force"
        return 1
    fi
    
    FORCE="${FORCE:-false}"
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will permanently delete the secret. Use --force to confirm."
        return 1
    fi
    
    if command -v vault::delete_secret &>/dev/null; then
        vault::delete_secret "$path"
    else
        log::error "vault::delete_secret not available"
        return 1
    fi
}

# Show authentication info
vault_auth_info() {
    if command -v vault::show_auth_info &>/dev/null; then
        vault::show_auth_info
    else
        log::error "Vault auth info not available"
        return 1
    fi
}

# Run functional tests
vault_test() {
    if command -v vault::test_functional &>/dev/null; then
        vault::test_functional
    else
        log::error "Vault functional tests not available"
        return 1
    fi
}

# Monitor Vault
vault_monitor() {
    local interval="${1:-5}"
    
    if command -v vault::monitor &>/dev/null; then
        vault::monitor "$interval"
    else
        log::error "Vault monitoring not available"
        return 1
    fi
}

# Create backup
vault_backup() {
    local backup_file="${1:-}"
    
    if command -v vault::docker::backup &>/dev/null; then
        vault::docker::backup "$backup_file"
    else
        log::error "Vault backup not available"
        return 1
    fi
}

# Restore backup
vault_restore() {
    local backup_file="${1:-}"
    
    if [[ -z "$backup_file" ]]; then
        log::error "Backup file required for restore"
        echo "Usage: resource-vault restore <backup-file>"
        echo ""
        echo "Examples:"
        echo "  resource-vault restore /backups/vault-2024-01-01.tar"
        echo "  resource-vault restore vault-backup.tar"
        return 1
    fi
    
    if command -v vault::docker::restore &>/dev/null; then
        vault::docker::restore "$backup_file"
    else
        log::error "Vault restore not available"
        return 1
    fi
}

# Migrate environment file
vault_migrate_env() {
    local env_file="${1:-}"
    local vault_prefix="${2:-}"
    
    if [[ -z "$env_file" || -z "$vault_prefix" ]]; then
        log::error "Both env-file and vault-prefix required"
        echo "Usage: resource-vault migrate-env <env-file> <vault-prefix>"
        echo ""
        echo "Examples:"
        echo "  resource-vault migrate-env .env secret/myapp"
        echo "  resource-vault migrate-env production.env secret/prod"
        return 1
    fi
    
    if command -v vault::migrate_env_file &>/dev/null; then
        vault::migrate_env_file "$env_file" "$vault_prefix"
    else
        log::error "vault::migrate_env_file not available"
        return 1
    fi
}

# Show logs
vault_logs() {
    local lines="${1:-50}"
    local follow="${2:-false}"
    local container_name="${VAULT_CONTAINER_NAME:-vault}"
    
    # Use shared utility with follow support
    docker_resource::show_logs_with_follow "$container_name" "$lines" "$follow"
}

# Diagnose Vault issues
vault_diagnose() {
    if command -v vault::diagnose &>/dev/null; then
        vault::diagnose
    else
        log::error "Vault diagnostics not available"
        return 1
    fi
}

# Custom help function with Vault-specific examples
vault_show_help() {
    # Show standard framework help first
    cli::_handle_help
    
    # Add Vault-specific examples
    echo ""
    echo "🔐 HashiCorp Vault Secrets Management Examples:"
    echo ""
    echo "Secrets Management:"
    echo "  resource-vault inject secret/myapp/db password123     # Store secret"
    echo "  resource-vault get-secret secret/myapp/db             # Retrieve secret"
    echo "  resource-vault list-secrets secret/myapp             # List secrets"
    echo "  resource-vault delete-secret secret/old/key --force  # Delete secret"
    echo ""
    echo "Initialization & Operations:"
    echo "  resource-vault init-dev                              # Setup development mode"
    echo "  resource-vault init-prod                             # Setup production mode"
    echo "  resource-vault unseal                                # Unseal vault"
    echo "  resource-vault auth-info                             # Show auth status"
    echo ""
    echo "Environment Migration:"
    echo "  resource-vault migrate-env .env secret/myapp         # Import .env file"
    echo "  resource-vault migrate-env production.env secret/prod # Import prod env"
    echo ""
    echo "Backup & Monitoring:"
    echo "  resource-vault backup /backups/vault-backup.tar      # Create backup"
    echo "  resource-vault restore /backups/vault-backup.tar     # Restore backup"
    echo "  resource-vault monitor 10                            # Monitor every 10s"
    echo "  resource-vault diagnose                              # Run diagnostics"
    echo ""
    echo "Security Features:"
    echo "  • Token-based authentication"
    echo "  • Secret versioning and rollback"
    echo "  • Audit logging and compliance"
    echo "  • Dynamic secrets generation"
    echo ""
    echo "Default Port: 8200"
    echo "Modes: dev (unsealed), prod (requires initialization)"
    echo "Web UI: http://localhost:8200/ui"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi