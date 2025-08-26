#!/usr/bin/env bash
################################################################################
# BTCPay Resource CLI - v2.0 Universal Contract Compliant
# 
# Self-hosted, open-source cryptocurrency payment processor
#
# Usage:
#   resource-btcpay <command> [options]
#   resource-btcpay <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    BTCPAY_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${BTCPAY_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
BTCPAY_CLI_DIR="${APP_ROOT}/resources/btcpay"

# Source standard variables
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# Source utilities
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"
# Source v2.0 CLI Command Framework
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# Source resource configuration
source "${BTCPAY_CLI_DIR}/config/defaults.sh"

# Source BTCPay libraries
for lib in common core docker install status content test inject; do
    lib_file="${BTCPAY_CLI_DIR}/lib/${lib}.sh"
    [[ -f "$lib_file" ]] && source "$lib_file" 2>/dev/null || true
done

# Initialize CLI framework in v2.0 mode
cli::init "btcpay" "BTCPay Server - Self-hosted cryptocurrency payment processor" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="btcpay::install::execute"
CLI_COMMAND_HANDLERS["manage::uninstall"]="btcpay::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="btcpay::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="btcpay::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="btcpay::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="btcpay::test::smoke"

# Content handlers
CLI_COMMAND_HANDLERS["content::add"]="btcpay::content::add"
CLI_COMMAND_HANDLERS["content::list"]="btcpay::content::list"
CLI_COMMAND_HANDLERS["content::get"]="btcpay::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="btcpay::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="btcpay::content::execute"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed BTCPay Server status" "btcpay::status"
cli::register_command "logs" "Show BTCPay Server logs" "btcpay::docker::logs"

# ==============================================================================
# OPTIONAL BTCPAY-SPECIFIC COMMANDS
# ==============================================================================
cli::register_command "credentials" "Show BTCPay credentials and API info" "btcpay::core::credentials"
cli::register_command "validate" "Validate BTCPay configuration" "btcpay::core::validate_config"

# Custom content subcommands for payment operations
cli::register_subcommand "content" "create-invoice" "Create a new payment invoice" "btcpay::create_invoice"
cli::register_subcommand "content" "check-payment" "Check payment status" "btcpay::check_payment"
cli::register_subcommand "content" "generate-address" "Generate crypto address" "btcpay::generate_address"

# Custom test phase for performance testing
cli::register_subcommand "test" "performance" "Test API performance" "btcpay::test::performance"

# Only execute if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi