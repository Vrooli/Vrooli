#!/usr/bin/env bash
################################################################################
# Vault Resource CLI
# 
# Lightweight CLI interface for Vault that delegates to existing lib functions.
#
# Usage:
#   resource-vault <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory
VAULT_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$VAULT_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$VAULT_CLI_DIR"
export VAULT_SCRIPT_DIR="$VAULT_CLI_DIR"  # For compatibility with existing libs

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}" 2>/dev/null || true

# Source the CLI template
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/resources/cli/resource-cli-template.sh"

# Source Vault configuration
# shellcheck disable=SC1091
source "${VAULT_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
vault::export_config 2>/dev/null || true

# Source Vault libraries
for lib in common docker api status install; do
    lib_file="${VAULT_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize with resource name
resource_cli::init "vault"

################################################################################
# Delegate to existing Vault functions
################################################################################

# Inject secrets into Vault
resource_cli::inject() {
    local path="${1:-}"
    local value="${2:-}"
    local key="${3:-value}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ -z "$path" || -z "$value" ]]; then
        log::error "Both path and value required for injection"
        echo "Usage: resource-vault inject <path> <value> [key]"
        echo "Example: resource-vault inject secret/myapp/db password123"
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would inject secret at path: $path"
        return 0
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
resource_cli::validate() {
    if command -v vault::show_status &>/dev/null; then
        vault::show_status
    elif command -v vault::common::is_healthy &>/dev/null; then
        vault::common::is_healthy
    else
        # Basic validation
        log::header "Validating Vault"
        docker ps --format '{{.Names}}' 2>/dev/null | grep -q "vault" || {
            log::error "Vault container not running"
            return 1
        }
        log::success "Vault is running"
    fi
}

# Show Vault status
resource_cli::status() {
    if command -v vault::show_status &>/dev/null; then
        vault::show_status
    else
        # Basic status
        log::header "Vault Status"
        if docker ps --format '{{.Names}}' 2>/dev/null | grep -q "vault"; then
            echo "Container: ✅ Running"
            docker ps --filter "name=vault" --format "table {{.Status}}\t{{.Ports}}" | tail -n 1
        else
            echo "Container: ❌ Not running"
        fi
    fi
}

# Start Vault
resource_cli::start() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would start Vault"
        return 0
    fi
    
    if command -v vault::docker::start_container &>/dev/null; then
        vault::docker::start_container
    else
        docker start vault || log::error "Failed to start Vault"
    fi
}

# Stop Vault
resource_cli::stop() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would stop Vault"
        return 0
    fi
    
    if command -v vault::docker::stop_container &>/dev/null; then
        vault::docker::stop_container
    else
        docker stop vault || log::error "Failed to stop Vault"
    fi
}

# Install Vault
resource_cli::install() {
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would install Vault"
        return 0
    fi
    
    if command -v vault::install &>/dev/null; then
        vault::install
    else
        log::error "vault::install not available"
        return 1
    fi
}

# Uninstall Vault
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    DRY_RUN="${DRY_RUN:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Vault and all its data. Use --force to confirm."
        return 1
    fi
    
    if [[ "$DRY_RUN" == "true" ]]; then
        log::info "[DRY RUN] Would uninstall Vault"
        return 0
    fi
    
    if command -v vault::uninstall &>/dev/null; then
        vault::uninstall
    else
        docker stop vault 2>/dev/null || true
        docker rm vault 2>/dev/null || true
        log::success "Vault uninstalled"
    fi
}

# Get credentials for n8n integration
resource_cli::credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    # Parse arguments
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "vault"; return 0; }
        return 1
    fi
    
    # Get resource status
    local status
    status=$(credentials::get_resource_status "$VAULT_CONTAINER_NAME")
    
    # Build connections array
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Get token for authentication
        local token=""
        if [[ "$VAULT_MODE" == "dev" ]]; then
            # Use development root token
            token="$VAULT_DEV_ROOT_TOKEN_ID"
        elif [[ -f "$VAULT_TOKEN_FILE" ]]; then
            # Try to read token from file for production mode
            token=$(cat "$VAULT_TOKEN_FILE" 2>/dev/null || echo "")
        fi
        
        if [[ -n "$token" ]]; then
            # Vault HTTP API connection
            local connection_obj
            connection_obj=$(jq -n \
                --arg host "localhost" \
                --argjson port "$VAULT_PORT" \
                --arg path "/v1" \
                --argjson ssl "$([[ "$VAULT_TLS_DISABLE" == "1" ]] && echo false || echo true)" \
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
                --arg mode "$VAULT_MODE" \
                --arg engine "$VAULT_SECRET_ENGINE" \
                --arg version "$VAULT_SECRET_VERSION" \
                --argjson default_paths "$(printf '%s\n' "${VAULT_DEFAULT_PATHS[@]}" | jq -R . | jq -s .)" \
                '{
                    description: $description,
                    mode: $mode,
                    secret_engine: $engine,
                    kv_version: $version,
                    default_paths: $default_paths
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
    
    # Build and validate response
    local response
    response=$(credentials::build_response "vault" "$status" "$connections_array")
    
    if credentials::validate_json "$response"; then
        credentials::format_output "$response"
    else
        log::error "Invalid credentials JSON generated"
        return 1
    fi
}

################################################################################
# Vault-specific commands (if functions exist)
################################################################################

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
        echo "Usage: resource-vault delete-secret <path>"
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
    
    if command -v vault::docker::show_logs &>/dev/null; then
        if [[ "$follow" == "true" ]]; then
            vault::docker::show_logs "$lines" "follow"
        else
            vault::docker::show_logs "$lines"
        fi
    else
        if [[ "$follow" == "true" ]]; then
            docker logs --tail "$lines" -f vault
        else
            docker logs --tail "$lines" vault
        fi
    fi
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

# Show help
resource_cli::show_help() {
    cat << EOF
🚀 Vault Resource CLI

USAGE:
    resource-vault <command> [options]

CORE COMMANDS:
    inject <path> <value> [key]     Store secret in Vault
    validate                        Validate Vault configuration
    status                          Show Vault status
    start                           Start Vault container
    stop                            Stop Vault container
    install                         Install Vault
    uninstall                       Uninstall Vault (requires --force)
    credentials                     Get connection credentials for n8n integration
    
VAULT COMMANDS:
    init-dev                        Initialize Vault for development
    init-prod                       Initialize Vault for production
    unseal                          Unseal Vault
    get-secret <path> [key] [format] Get secret from Vault
    list-secrets <path> [format]    List secrets at path
    delete-secret <path>            Delete secret (requires --force)
    auth-info                       Show authentication information
    test                            Run functional tests
    monitor [interval]              Monitor Vault (default: 5s)
    backup [file]                   Create Vault backup
    restore <file>                  Restore Vault backup
    migrate-env <env-file> <prefix> Migrate .env file to Vault
    logs [lines] [follow]           Show Vault logs (default: 50 lines)
    diagnose                        Diagnose Vault issues

OPTIONS:
    --verbose, -v                   Show detailed output
    --dry-run                       Preview actions without executing
    --force                         Force operation (skip confirmations)

EXAMPLES:
    resource-vault status
    resource-vault inject secret/myapp/db password123
    resource-vault get-secret secret/myapp/db
    resource-vault list-secrets secret/myapp
    resource-vault migrate-env .env secret/myapp
    resource-vault delete-secret secret/myapp/db --force

For more information: https://docs.vrooli.com/resources/vault
EOF
}

# Main command router
resource_cli::main() {
    # Parse common options first
    local remaining_args
    remaining_args=$(resource_cli::parse_options "$@")
    set -- $remaining_args
    
    local command="${1:-help}"
    shift || true
    
    case "$command" in
        # Standard resource commands
        inject|validate|status|start|stop|install|uninstall|credentials)
            resource_cli::$command "$@"
            ;;
            
        # Vault-specific commands
        init-dev)
            vault_init_dev "$@"
            ;;
        init-prod)
            vault_init_prod "$@"
            ;;
        unseal)
            vault_unseal "$@"
            ;;
        get-secret)
            vault_get_secret "$@"
            ;;
        list-secrets)
            vault_list_secrets "$@"
            ;;
        delete-secret)
            vault_delete_secret "$@"
            ;;
        auth-info)
            vault_auth_info "$@"
            ;;
        test)
            vault_test "$@"
            ;;
        monitor)
            vault_monitor "$@"
            ;;
        backup)
            vault_backup "$@"
            ;;
        restore)
            vault_restore "$@"
            ;;
        migrate-env)
            vault_migrate_env "$@"
            ;;
        logs)
            vault_logs "$@"
            ;;
        diagnose)
            vault_diagnose "$@"
            ;;
            
        help|--help|-h)
            resource_cli::show_help
            ;;
        *)
            log::error "Unknown command: $command"
            echo ""
            resource_cli::show_help
            exit 1
            ;;
    esac
}

# Run main if executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    resource_cli::main "$@"
fi