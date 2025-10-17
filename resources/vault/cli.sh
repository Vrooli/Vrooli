#!/usr/bin/env bash
################################################################################
# Vault Resource CLI - v2.0 Universal Contract Compliant
# 
# HashiCorp Vault secrets management platform
#
# Usage:
#   resource-vault <command> [options]
#   resource-vault <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    VAULT_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${VAULT_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
VAULT_CLI_DIR="${APP_ROOT}/resources/vault"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${VAULT_CLI_DIR}/config/defaults.sh"

# Source Vault libraries
for lib in common docker api status install security-health audit secrets-manager; do
    lib_file="${VAULT_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Wrapper functions for v2.0 content command compliance
vault::content_add_wrapper() {
    # Convert v2.0 content add flags to vault::put_secret format
    local path="" value="" key="value" file="" name=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --file)
                file="$2"
                shift 2
                ;;
            --name)
                path="$2"
                shift 2
                ;;
            --type)
                # Ignored for vault
                shift 2
                ;;
            --path)
                path="$2"
                shift 2
                ;;
            --value)
                value="$2"
                shift 2
                ;;
            --data)
                value="$2"
                shift 2
                ;;
            *)
                # Pass through to original function
                break
                ;;
        esac
    done
    
    # If file is provided, read its content
    if [[ -n "$file" && -f "$file" ]]; then
        value=$(cat "$file")
    fi
    
    # Use name as path if path not set
    if [[ -z "$path" && -n "$name" ]]; then
        path="$name"
    fi
    
    vault::put_secret --path "$path" --value "$value" --key "$key" "$@"
}

vault::content_list_wrapper() {
    # v2.0 content list should work without arguments
    vault::list_secrets "$@"
}

vault::content_get_wrapper() {
    # Convert v2.0 content get flags to vault::get_secret format
    local path="" output=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                path="$2"
                shift 2
                ;;
            --path)
                path="$2"
                shift 2
                ;;
            --output)
                output="$2"
                shift 2
                ;;
            *)
                # Pass through
                break
                ;;
        esac
    done
    
    if [[ -n "$output" ]]; then
        vault::get_secret --path "$path" --format raw "$@" > "$output"
    else
        vault::get_secret --path "$path" "$@"
    fi
}

vault::content_remove_wrapper() {
    # Convert v2.0 content remove flags to vault::delete_secret format
    local path=""
    
    while [[ $# -gt 0 ]]; do
        case "$1" in
            --name)
                path="$2"
                shift 2
                ;;
            --path)
                path="$2"
                shift 2
                ;;
            --force)
                # Already handled by delete_secret
                shift
                ;;
            *)
                break
                ;;
        esac
    done
    
    vault::delete_secret --path "$path" "$@"
}

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "vault" "HashiCorp Vault secrets management" "v2"

# Register additional command groups
cli::register_command_group "audit" "Audit logging management"
cli::register_command_group "access" "Access control and policy management"
cli::register_command_group "secrets" "Resource secrets management"

# Override default handlers to point directly to vault implementations
CLI_COMMAND_HANDLERS["manage::install"]="vault::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="vault::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="vault::docker::start_container"  
CLI_COMMAND_HANDLERS["manage::stop"]="vault::docker::stop_container"
CLI_COMMAND_HANDLERS["manage::restart"]="vault::docker::restart_container"
CLI_COMMAND_HANDLERS["test::smoke"]="vault::status"

# Override content handlers for Vault-specific secret management functionality
CLI_COMMAND_HANDLERS["content::add"]="vault::content_add_wrapper"
CLI_COMMAND_HANDLERS["content::list"]="vault::content_list_wrapper" 
CLI_COMMAND_HANDLERS["content::get"]="vault::content_get_wrapper"
CLI_COMMAND_HANDLERS["content::remove"]="vault::content_remove_wrapper"
CLI_COMMAND_HANDLERS["content::execute"]="vault::test_functional"

# Add Vault-specific content subcommands not in the standard framework
cli::register_subcommand "content" "init-dev" "Initialize development mode" "vault::init_dev" "modifies-system"
cli::register_subcommand "content" "init-prod" "Initialize production mode" "vault::init_prod" "modifies-system"
cli::register_subcommand "content" "unseal" "Unseal vault instance" "vault::unseal" "modifies-system"
cli::register_subcommand "content" "backup" "Create vault backup" "vault::docker::backup" "modifies-system"
cli::register_subcommand "content" "restore" "Restore vault backup" "vault::docker::restore" "modifies-system"
cli::register_subcommand "content" "migrate-env" "Migrate .env file to vault" "vault::migrate_env_file" "modifies-system"

# Add Vault-specific test subcommands beyond standard smoke/integration
cli::register_subcommand "test" "auth" "Test authentication status" "vault::show_auth_info"
cli::register_subcommand "test" "secrets" "Test secret operations" "vault::test_secret_operations"

# Additional information commands
cli::register_command "status" "Show detailed vault status" "vault::show_status"
cli::register_command "logs" "Show Vault logs" "vault::docker::show_logs"
cli::register_command "credentials" "Show Vault credentials for integration" "vault::show_integration_info"

cli::register_command "monitor" "Monitor Vault status" "vault::monitor"
cli::register_command "diagnose" "Diagnose Vault issues" "vault::diagnose"

# Security-specific commands
cli::register_command "security-health" "Comprehensive security health check" "vault::security::health_check"
cli::register_command "security-monitor" "Monitor security events in real-time" "vault::security::monitor"
cli::register_command "security-audit" "Generate security audit report" "vault::security::audit_report"

# Audit and access control commands
cli::register_subcommand "audit" "enable" "Enable audit logging" "vault::audit::enable" "modifies-system"
cli::register_subcommand "audit" "disable" "Disable audit logging" "vault::audit::disable" "modifies-system"
cli::register_subcommand "audit" "list" "List audit devices" "vault::audit::list"
cli::register_subcommand "audit" "analyze" "Analyze audit logs" "vault::audit::analyze"

cli::register_subcommand "access" "create-policy" "Create access control policy" "vault::access::create_policy" "modifies-system"
cli::register_subcommand "access" "delete-policy" "Delete access control policy" "vault::access::delete_policy" "modifies-system"
cli::register_subcommand "access" "list-policies" "List access control policies" "vault::access::list_policies"
cli::register_subcommand "access" "create-token" "Create token with policy" "vault::access::create_token" "modifies-system"
cli::register_subcommand "access" "setup-standard" "Setup standard Vrooli policies" "vault::access::setup_standard_policies" "modifies-system"

# Resource Secrets Management commands
vault::secrets_scan_wrapper() { vault_secrets_command scan "$@"; }
vault::secrets_check_wrapper() { vault_secrets_command check "$@"; }
vault::secrets_init_wrapper() { vault_secrets_command init "$@"; }
vault::secrets_validate_wrapper() { vault_secrets_command validate "$@"; }
vault::secrets_export_wrapper() { vault_secrets_command export "$@"; }
vault::secrets_create_template_wrapper() { vault_secrets_command create-template "$@"; }
vault::secrets_discover_wrapper() { vault_secrets_command discover "$@"; }

cli::register_subcommand "secrets" "scan" "Scan all resources for secrets declarations" "vault::secrets_scan_wrapper"
cli::register_subcommand "secrets" "check" "Check secrets status for a resource" "vault::secrets_check_wrapper"
cli::register_subcommand "secrets" "init" "Initialize secrets for a resource" "vault::secrets_init_wrapper" "modifies-system"
cli::register_subcommand "secrets" "validate" "Validate all resource secrets" "vault::secrets_validate_wrapper"
cli::register_subcommand "secrets" "export" "Export secrets as environment variables" "vault::secrets_export_wrapper"
cli::register_subcommand "secrets" "create-template" "Create secrets.yaml template" "vault::secrets_create_template_wrapper" "modifies-system"
cli::register_subcommand "secrets" "discover" "Find hardcoded secrets in resource" "vault::secrets_discover_wrapper"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi