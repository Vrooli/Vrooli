#!/usr/bin/env bash
################################################################################
# Keycloak Resource CLI - v2.0 Universal Contract Compliant
# 
# Open Source Identity and Access Management for modern applications
#
# Usage:
#   resource-keycloak <command> [options]
#   resource-keycloak <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    KEYCLOAK_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${KEYCLOAK_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
KEYCLOAK_CLI_DIR="${APP_ROOT}/resources/keycloak"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${KEYCLOAK_CLI_DIR}/config/defaults.sh"

# Source Keycloak libraries
for lib in common install lifecycle status inject content; do
    lib_file="${KEYCLOAK_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "keycloak" "Identity and Access Management platform" "v2"

# Override default handlers to point directly to keycloak implementations
CLI_COMMAND_HANDLERS["manage::install"]="keycloak::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="keycloak::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="keycloak::start"
CLI_COMMAND_HANDLERS["manage::stop"]="keycloak::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="keycloak::restart"
# Test handlers - delegate to test runner
CLI_COMMAND_HANDLERS["test::smoke"]="keycloak::test::run_phase smoke"
CLI_COMMAND_HANDLERS["test::integration"]="keycloak::test::run_phase integration"
CLI_COMMAND_HANDLERS["test::unit"]="keycloak::test::run_phase unit"
CLI_COMMAND_HANDLERS["test::all"]="keycloak::test::run_phase all"

# Test runner function
keycloak::test::run_phase() {
    local phase="${1:-all}"
    "${KEYCLOAK_CLI_DIR}/test/run-tests.sh" "$phase"
}

# Override content handlers for Keycloak-specific realm/user management
CLI_COMMAND_HANDLERS["content::add"]="keycloak::content::add"
CLI_COMMAND_HANDLERS["content::list"]="keycloak::content::list"
CLI_COMMAND_HANDLERS["content::get"]="keycloak::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="keycloak::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="keycloak::content::execute"

# Add Keycloak-specific custom commands for realm management
cli::register_subcommand "content" "inject" "Inject realm configuration from JSON file" "keycloak::inject"
cli::register_subcommand "content" "realms" "List configured realms and users" "keycloak::list_injected"
cli::register_subcommand "content" "clear" "Clear all Keycloak data" "keycloak::clear_data"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "keycloak::status"
cli::register_command "logs" "Show Keycloak logs" "keycloak::logs"
cli::register_command "credentials" "Show Keycloak admin credentials" "keycloak::credentials"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi