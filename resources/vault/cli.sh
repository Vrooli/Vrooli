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
for lib in common docker api status install; do
    lib_file="${VAULT_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "vault" "HashiCorp Vault secrets management" "v2"

# Override default handlers to point directly to vault implementations
CLI_COMMAND_HANDLERS["manage::install"]="vault::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="vault::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="vault::docker::start_container"  
CLI_COMMAND_HANDLERS["manage::stop"]="vault::docker::stop_container"
CLI_COMMAND_HANDLERS["manage::restart"]="vault::docker::restart_container"
CLI_COMMAND_HANDLERS["test::smoke"]="vault::status"

# Override content handlers for Vault-specific secret management functionality
CLI_COMMAND_HANDLERS["content::add"]="vault::put_secret"
CLI_COMMAND_HANDLERS["content::list"]="vault::list_secrets" 
CLI_COMMAND_HANDLERS["content::get"]="vault::get_secret"
CLI_COMMAND_HANDLERS["content::remove"]="vault::delete_secret"
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

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi