#!/usr/bin/env bash
################################################################################
# Mifos X Resource CLI - v2.0 Universal Contract Compliant
# 
# Digital finance platform for microfinance and financial inclusion
# Powered by Apache Fineract core banking engine
#
# Usage:
#   resource-mifos <command> [options]
#   resource-mifos <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    MIFOS_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${MIFOS_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
MIFOS_CLI_DIR="${APP_ROOT}/resources/mifos"

# Source standard variables
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# Source utilities
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"
# Source v2.0 CLI Command Framework
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# Source resource configuration
source "${MIFOS_CLI_DIR}/config/defaults.sh"

# Source Mifos libraries
for lib in common core docker install status content test clients loans; do
    lib_file="${MIFOS_CLI_DIR}/lib/${lib}.sh"
    [[ -f "$lib_file" ]] && source "$lib_file" 2>/dev/null || true
done

# Initialize CLI framework in v2.0 mode
cli::init "mifos" "Mifos X - Digital finance platform for microfinance" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="mifos::install::execute"
CLI_COMMAND_HANDLERS["manage::uninstall"]="mifos::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="mifos::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="mifos::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="mifos::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="mifos::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="mifos::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="mifos::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="mifos::test::all"

# Content handlers
CLI_COMMAND_HANDLERS["content::add"]="mifos::content::add"
CLI_COMMAND_HANDLERS["content::list"]="mifos::content::list"
CLI_COMMAND_HANDLERS["content::get"]="mifos::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="mifos::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="mifos::content::execute"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed Mifos X status" "mifos::status"
cli::register_command "logs" "Show Mifos X logs" "mifos::docker::logs"

# ==============================================================================
# OPTIONAL MIFOS-SPECIFIC COMMANDS
# ==============================================================================
cli::register_command "credentials" "Show Mifos credentials and API info" "mifos::core::credentials"
cli::register_command "validate" "Validate Mifos configuration" "mifos::core::validate_config"

# Custom content subcommands for banking operations
cli::register_subcommand "content" "create-client" "Create a new client" "mifos::clients::create"
cli::register_subcommand "content" "open-account" "Open a savings/loan account" "mifos::clients::open_account"
cli::register_subcommand "content" "disburse-loan" "Disburse a loan" "mifos::loans::disburse"
cli::register_subcommand "content" "payment-status" "Check payment status" "mifos::loans::payment_status"

# Custom test phase for performance testing
cli::register_subcommand "test" "performance" "Test API performance" "mifos::test::performance"

# ==============================================================================
# FINANCIAL PRODUCTS COMMANDS
# ==============================================================================
# Register Products command group
cli::register_command_group "products" "Financial products management"

# Register Products subcommands
cli::register_subcommand "products" "list" "List all financial products" "mifos::products::list"
cli::register_subcommand "products" "create-loan" "Create loan product" "mifos::products::create_loan"
cli::register_subcommand "products" "create-savings" "Create savings product" "mifos::products::create_savings"
cli::register_subcommand "products" "configure" "Configure product settings" "mifos::products::configure"
cli::register_subcommand "products" "status" "Show products status" "mifos::products::status"

# Register Products command handlers for v2.0 framework
CLI_COMMAND_HANDLERS["products::list"]="mifos::products::list"
CLI_COMMAND_HANDLERS["products::create-loan"]="mifos::products::create_loan"
CLI_COMMAND_HANDLERS["products::create-savings"]="mifos::products::create_savings"
CLI_COMMAND_HANDLERS["products::configure"]="mifos::products::configure"
CLI_COMMAND_HANDLERS["products::status"]="mifos::products::status"

# ==============================================================================
# MAIN DISPATCH - Required by v2.0 framework
# ==============================================================================
# Only execute if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi