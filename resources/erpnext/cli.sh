#!/usr/bin/env bash
################################################################################
# ERPNext Resource CLI - v2.0 Universal Contract Compliant
# 
# Open-source ERP system for managing business operations
#
# Usage:
#   resource-erpnext <command> [options]
#   resource-erpnext <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    ERPNEXT_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${ERPNEXT_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
ERPNEXT_CLI_DIR="${APP_ROOT}/resources/erpnext"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${ERPNEXT_CLI_DIR}/config/defaults.sh"

# Source main library for compatibility
if [[ -f "${ERPNEXT_CLI_DIR}/lib/main.sh" ]]; then
    # shellcheck disable=SC1090
    source "${ERPNEXT_CLI_DIR}/lib/main.sh" 2>/dev/null || true
fi

# Source ERPNext libraries
for lib in config docker install status inject content test; do
    lib_file="${ERPNEXT_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "erpnext" "ERPNext ERP system management" "v2"

# Override default handlers to point directly to ERPNext implementations
CLI_COMMAND_HANDLERS["manage::install"]="erpnext::install::execute"
CLI_COMMAND_HANDLERS["manage::uninstall"]="erpnext::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="erpnext::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="erpnext::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="erpnext::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="erpnext::test::smoke"

# Override content handlers for ERPNext-specific functionality
CLI_COMMAND_HANDLERS["content::add"]="erpnext::content::add"
CLI_COMMAND_HANDLERS["content::list"]="erpnext::content::list" 
CLI_COMMAND_HANDLERS["content::get"]="erpnext::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="erpnext::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="erpnext::content::execute"

# Add custom test phases for ERPNext
cli::register_subcommand "test" "performance" "Run performance tests on ERPNext" "erpnext::test::performance"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "erpnext::status"
cli::register_command "logs" "Show ERPNext logs" "erpnext::docker::logs"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi