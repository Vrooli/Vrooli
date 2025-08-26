#!/usr/bin/env bash
################################################################################
# Odoo Resource CLI - v2.0 Universal Contract Compliant
# 
# Comprehensive ERP platform with integrated business applications
#
# Usage:
#   resource-odoo <command> [options]
#   resource-odoo <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    ODOO_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${ODOO_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
ODOO_CLI_DIR="${APP_ROOT}/resources/odoo"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${ODOO_CLI_DIR}/config/defaults.sh"

# Source Odoo libraries
for lib in common install start stop status docker content test; do
    lib_file="${ODOO_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "odoo" "Odoo Community Edition ERP platform" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="odoo::install::execute"
CLI_COMMAND_HANDLERS["manage::uninstall"]="odoo::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="odoo::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="odoo::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="odoo::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="odoo::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="odoo::test::integration"

# Content handlers for ERP business functionality
CLI_COMMAND_HANDLERS["content::add"]="odoo::content::add"
CLI_COMMAND_HANDLERS["content::list"]="odoo::content::list" 
CLI_COMMAND_HANDLERS["content::get"]="odoo::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="odoo::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="odoo::content::execute"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed Odoo status" "odoo::status"
cli::register_command "logs" "Show Odoo logs" "odoo::docker::logs"

# ==============================================================================
# ODOO-SPECIFIC CONTENT OPERATIONS
# ==============================================================================
# Add ERP-specific content subcommands
cli::register_subcommand "content" "modules" "List installed modules" "odoo::content::list modules"
cli::register_subcommand "content" "users" "List system users" "odoo::content::list users"
cli::register_subcommand "content" "companies" "List companies" "odoo::content::list companies"
cli::register_subcommand "content" "databases" "List databases" "odoo::content::list databases"
cli::register_subcommand "content" "backup" "Create database backup" "odoo::content::execute backup" "modifies-system"
cli::register_subcommand "content" "shell" "Open interactive Odoo shell" "odoo::content::execute shell"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi