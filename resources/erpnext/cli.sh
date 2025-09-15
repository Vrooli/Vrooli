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
for lib in config docker install status inject content test workflow reporting ecommerce manufacturing; do
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
cli::register_command "credentials" "Show access credentials and instructions" "erpnext::show_credentials"

# Workflow management commands
cli::register_command_group "workflow" "Manage ERPNext workflows"
cli::register_subcommand "workflow" "list" "List available workflows" "erpnext::workflow::cli::list"
cli::register_subcommand "workflow" "get" "Get workflow details" "erpnext::workflow::cli::get"
cli::register_subcommand "workflow" "create" "Create new workflow" "erpnext::workflow::cli::create"
cli::register_subcommand "workflow" "transition" "Execute workflow transition" "erpnext::workflow::cli::transition"

# Reporting commands
cli::register_command_group "report" "Manage ERPNext reports"
cli::register_subcommand "report" "list" "List available reports" "erpnext::report::cli::list"
cli::register_subcommand "report" "get" "Get report details" "erpnext::report::cli::get"
cli::register_subcommand "report" "execute" "Execute report with filters" "erpnext::report::cli::execute"
cli::register_subcommand "report" "create" "Create new report" "erpnext::report::cli::create"
cli::register_subcommand "report" "export" "Export report data" "erpnext::report::cli::export"

# E-commerce commands
cli::register_command_group "ecommerce" "Manage e-commerce functionality"
cli::register_subcommand "ecommerce" "list-products" "List online products" "erpnext::ecommerce::list_products"
cli::register_subcommand "ecommerce" "add-product" "Add product to online store" "erpnext::ecommerce::add_product"
cli::register_subcommand "ecommerce" "get-cart" "Get shopping cart contents" "erpnext::ecommerce::get_cart"
cli::register_subcommand "ecommerce" "add-to-cart" "Add item to shopping cart" "erpnext::ecommerce::add_to_cart"
cli::register_subcommand "ecommerce" "configure" "Configure store settings" "erpnext::ecommerce::configure_store"

# Manufacturing commands
cli::register_command_group "manufacturing" "Manage manufacturing operations"
cli::register_subcommand "manufacturing" "list-boms" "List bill of materials" "erpnext::manufacturing::list_boms"
cli::register_subcommand "manufacturing" "create-bom" "Create new BOM" "erpnext::manufacturing::create_bom"
cli::register_subcommand "manufacturing" "add-bom-item" "Add item to BOM" "erpnext::manufacturing::add_bom_item"
cli::register_subcommand "manufacturing" "list-work-orders" "List work orders" "erpnext::manufacturing::list_work_orders"
cli::register_subcommand "manufacturing" "create-work-order" "Create work order" "erpnext::manufacturing::create_work_order"
cli::register_subcommand "manufacturing" "production-plan" "Get production plan" "erpnext::manufacturing::get_production_plan"
cli::register_subcommand "manufacturing" "stock-entries" "Get stock entries" "erpnext::manufacturing::get_stock_entry"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi