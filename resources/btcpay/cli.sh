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
for lib in common core docker install status content test inject lightning multicurrency pos crowdfunding paybutton; do
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
CLI_COMMAND_HANDLERS["test::integration"]="btcpay::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="btcpay::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="btcpay::test::all"

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

# ==============================================================================
# MULTI-CURRENCY COMMANDS
# ==============================================================================
# Register Multi-currency command group
cli::register_command_group "multicurrency" "Multi-currency management"

# Register Multi-currency subcommands
cli::register_subcommand "multicurrency" "configure" "Configure multi-currency support" "btcpay::multicurrency::configure"
cli::register_subcommand "multicurrency" "enable" "Enable a specific currency" "btcpay::multicurrency::enable_currency"
cli::register_subcommand "multicurrency" "disable" "Disable a specific currency" "btcpay::multicurrency::disable_currency"
cli::register_subcommand "multicurrency" "list" "List supported currencies" "btcpay::multicurrency::list"
cli::register_subcommand "multicurrency" "status" "Show multi-currency status" "btcpay::multicurrency::status"
cli::register_subcommand "multicurrency" "apply" "Apply configuration and restart" "btcpay::multicurrency::apply"

# Register Multi-currency command handlers for v2.0 framework
CLI_COMMAND_HANDLERS["multicurrency::configure"]="btcpay::multicurrency::configure"
CLI_COMMAND_HANDLERS["multicurrency::enable"]="btcpay::multicurrency::enable_currency"
CLI_COMMAND_HANDLERS["multicurrency::disable"]="btcpay::multicurrency::disable_currency"
CLI_COMMAND_HANDLERS["multicurrency::list"]="btcpay::multicurrency::list"
CLI_COMMAND_HANDLERS["multicurrency::status"]="btcpay::multicurrency::status"
CLI_COMMAND_HANDLERS["multicurrency::apply"]="btcpay::multicurrency::apply"

# ==============================================================================
# POINT OF SALE COMMANDS
# ==============================================================================
# Register POS command group
cli::register_command_group "pos" "Point of Sale management"

# Register POS subcommands
cli::register_subcommand "pos" "configure" "Configure POS system" "btcpay::pos::configure"
cli::register_subcommand "pos" "add-item" "Add item to inventory" "btcpay::pos::add_item"
cli::register_subcommand "pos" "list-items" "List all POS items" "btcpay::pos::list_items"
cli::register_subcommand "pos" "remove-item" "Remove item from inventory" "btcpay::pos::remove_item"
cli::register_subcommand "pos" "update-item" "Update item property" "btcpay::pos::update_item"
cli::register_subcommand "pos" "import" "Import items from CSV" "btcpay::pos::import_items"
cli::register_subcommand "pos" "generate" "Generate POS terminal" "btcpay::pos::generate_terminal"
cli::register_subcommand "pos" "status" "Show POS status" "btcpay::pos::status"

# Register POS command handlers for v2.0 framework
CLI_COMMAND_HANDLERS["pos::configure"]="btcpay::pos::configure"
CLI_COMMAND_HANDLERS["pos::add-item"]="btcpay::pos::add_item"
CLI_COMMAND_HANDLERS["pos::list-items"]="btcpay::pos::list_items"
CLI_COMMAND_HANDLERS["pos::remove-item"]="btcpay::pos::remove_item"
CLI_COMMAND_HANDLERS["pos::update-item"]="btcpay::pos::update_item"
CLI_COMMAND_HANDLERS["pos::import"]="btcpay::pos::import_items"
CLI_COMMAND_HANDLERS["pos::generate"]="btcpay::pos::generate_terminal"
CLI_COMMAND_HANDLERS["pos::status"]="btcpay::pos::status"

# ==============================================================================
# LIGHTNING NETWORK COMMANDS
# ==============================================================================
# Register Lightning command group
cli::register_command_group "lightning" "Lightning Network management"

# Register Lightning subcommands
cli::register_subcommand "lightning" "setup" "Set up Lightning Network support" "btcpay::lightning::setup"
cli::register_subcommand "lightning" "status" "Show Lightning Network status" "btcpay::lightning::status"
cli::register_subcommand "lightning" "create-invoice" "Create Lightning invoice" "btcpay::lightning::create_invoice"
cli::register_subcommand "lightning" "check-invoice" "Check Lightning invoice status" "btcpay::lightning::check_invoice"
cli::register_subcommand "lightning" "balance" "Show Lightning wallet balance" "btcpay::lightning::balance"
cli::register_subcommand "lightning" "channels" "List Lightning channels" "btcpay::lightning::list_channels"
cli::register_subcommand "lightning" "open-channel" "Open a Lightning channel" "btcpay::lightning::open_channel"

# Register Lightning command handlers for v2.0 framework
CLI_COMMAND_HANDLERS["lightning::setup"]="btcpay::lightning::setup"
CLI_COMMAND_HANDLERS["lightning::status"]="btcpay::lightning::status"
CLI_COMMAND_HANDLERS["lightning::create-invoice"]="btcpay::lightning::create_invoice"
CLI_COMMAND_HANDLERS["lightning::check-invoice"]="btcpay::lightning::check_invoice"
CLI_COMMAND_HANDLERS["lightning::balance"]="btcpay::lightning::balance"
CLI_COMMAND_HANDLERS["lightning::channels"]="btcpay::lightning::list_channels"
CLI_COMMAND_HANDLERS["lightning::open-channel"]="btcpay::lightning::open_channel"

# ==============================================================================
# CROWDFUNDING COMMANDS
# ==============================================================================
# Register Crowdfunding command group
cli::register_command_group "crowdfunding" "Crowdfunding campaign management"

# Register Crowdfunding subcommands
cli::register_subcommand "crowdfunding" "configure" "Configure crowdfunding system" "btcpay::crowdfunding::configure"
cli::register_subcommand "crowdfunding" "create" "Create new campaign" "btcpay::crowdfunding::create_campaign"
cli::register_subcommand "crowdfunding" "list" "List all campaigns" "btcpay::crowdfunding::list_campaigns"
cli::register_subcommand "crowdfunding" "update" "Update campaign property" "btcpay::crowdfunding::update_campaign"
cli::register_subcommand "crowdfunding" "contribute" "Record contribution" "btcpay::crowdfunding::contribute"
cli::register_subcommand "crowdfunding" "status" "Show campaign status" "btcpay::crowdfunding::status"
cli::register_subcommand "crowdfunding" "close" "Close campaign" "btcpay::crowdfunding::close_campaign"
cli::register_subcommand "crowdfunding" "export" "Export campaign data" "btcpay::crowdfunding::export_campaign"
cli::register_subcommand "crowdfunding" "widget" "Generate embeddable widget" "btcpay::crowdfunding::generate_widget"

# Register Crowdfunding command handlers
CLI_COMMAND_HANDLERS["crowdfunding::configure"]="btcpay::crowdfunding::configure"
CLI_COMMAND_HANDLERS["crowdfunding::create"]="btcpay::crowdfunding::create_campaign"
CLI_COMMAND_HANDLERS["crowdfunding::list"]="btcpay::crowdfunding::list_campaigns"
CLI_COMMAND_HANDLERS["crowdfunding::update"]="btcpay::crowdfunding::update_campaign"
CLI_COMMAND_HANDLERS["crowdfunding::contribute"]="btcpay::crowdfunding::contribute"
CLI_COMMAND_HANDLERS["crowdfunding::status"]="btcpay::crowdfunding::status"
CLI_COMMAND_HANDLERS["crowdfunding::close"]="btcpay::crowdfunding::close_campaign"
CLI_COMMAND_HANDLERS["crowdfunding::export"]="btcpay::crowdfunding::export_campaign"
CLI_COMMAND_HANDLERS["crowdfunding::widget"]="btcpay::crowdfunding::generate_widget"

# ==============================================================================
# PAYMENT BUTTON COMMANDS
# ==============================================================================
# Register Payment Button command group
cli::register_command_group "paybutton" "Payment button management"

# Register Payment Button subcommands
cli::register_subcommand "paybutton" "create" "Create payment button" "btcpay::paybutton::create"
cli::register_subcommand "paybutton" "list" "List all buttons" "btcpay::paybutton::list"
cli::register_subcommand "paybutton" "get-code" "Get embed code" "btcpay::paybutton::get_code"
cli::register_subcommand "paybutton" "update" "Update button property" "btcpay::paybutton::update"
cli::register_subcommand "paybutton" "delete" "Delete button" "btcpay::paybutton::delete"
cli::register_subcommand "paybutton" "stats" "Show button statistics" "btcpay::paybutton::stats"
cli::register_subcommand "paybutton" "generate-styles" "Generate button styles" "btcpay::paybutton::generate_styles"
cli::register_subcommand "paybutton" "bulk-create" "Create from CSV" "btcpay::paybutton::bulk_create"

# Register Payment Button command handlers
CLI_COMMAND_HANDLERS["paybutton::create"]="btcpay::paybutton::create"
CLI_COMMAND_HANDLERS["paybutton::list"]="btcpay::paybutton::list"
CLI_COMMAND_HANDLERS["paybutton::get-code"]="btcpay::paybutton::get_code"
CLI_COMMAND_HANDLERS["paybutton::update"]="btcpay::paybutton::update"
CLI_COMMAND_HANDLERS["paybutton::delete"]="btcpay::paybutton::delete"
CLI_COMMAND_HANDLERS["paybutton::stats"]="btcpay::paybutton::stats"
CLI_COMMAND_HANDLERS["paybutton::generate-styles"]="btcpay::paybutton::generate_styles"
CLI_COMMAND_HANDLERS["paybutton::bulk-create"]="btcpay::paybutton::bulk_create"

# Only execute if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi