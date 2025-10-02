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
for lib in config docker install status inject content test workflow reporting ecommerce manufacturing multi-tenant mobile-ui crm accounting hr inventory projects; do
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

# CRM Module commands
cli::register_command_group "crm" "Customer Relationship Management operations"
cli::register_subcommand "crm" "list-customers" "List all customers" "erpnext::crm::list_customers"
cli::register_subcommand "crm" "create-customer" "Create new customer" "erpnext::crm::create_customer"
cli::register_subcommand "crm" "list-leads" "List all leads" "erpnext::crm::list_leads"
cli::register_subcommand "crm" "create-lead" "Create new lead" "erpnext::crm::create_lead"
cli::register_subcommand "crm" "list-contacts" "List all contacts" "erpnext::crm::list_contacts"
cli::register_subcommand "crm" "create-contact" "Create new contact" "erpnext::crm::create_contact"
cli::register_subcommand "crm" "list-opportunities" "List opportunities" "erpnext::crm::list_opportunities"

# Accounting Module commands
cli::register_command_group "accounting" "Financial accounting operations"
cli::register_subcommand "accounting" "list-invoices" "List invoices" "erpnext::accounting::list_invoices"
cli::register_subcommand "accounting" "create-invoice" "Create new invoice" "erpnext::accounting::create_invoice"
cli::register_subcommand "accounting" "list-journal-entries" "List journal entries" "erpnext::accounting::list_journal_entries"
cli::register_subcommand "accounting" "list-accounts" "List chart of accounts" "erpnext::accounting::list_accounts"
cli::register_subcommand "accounting" "list-payments" "List payment entries" "erpnext::accounting::list_payments"
cli::register_subcommand "accounting" "create-payment" "Create payment entry" "erpnext::accounting::create_payment"
cli::register_subcommand "accounting" "balance-sheet" "View balance sheet" "erpnext::accounting::get_balance_sheet"

# HR Module commands
cli::register_command_group "hr" "Human Resources management"
cli::register_subcommand "hr" "list-employees" "List all employees" "erpnext::hr::list_employees"
cli::register_subcommand "hr" "create-employee" "Create new employee" "erpnext::hr::create_employee"
cli::register_subcommand "hr" "list-departments" "List departments" "erpnext::hr::list_departments"
cli::register_subcommand "hr" "list-leaves" "List leave applications" "erpnext::hr::list_leave_applications"
cli::register_subcommand "hr" "create-leave" "Create leave application" "erpnext::hr::create_leave_application"
cli::register_subcommand "hr" "list-attendance" "List attendance records" "erpnext::hr::list_attendance"
cli::register_subcommand "hr" "mark-attendance" "Mark employee attendance" "erpnext::hr::mark_attendance"
cli::register_subcommand "hr" "list-salary" "List salary structures" "erpnext::hr::list_salary_structures"

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

# Multi-tenant commands
cli::register_command_group "multi-tenant" "Manage multiple companies/organizations"
cli::register_subcommand "multi-tenant" "list-companies" "List all companies" "erpnext::multi_tenant::list_companies"
cli::register_subcommand "multi-tenant" "create-company" "Create new company" "erpnext::multi_tenant::create_company"
cli::register_subcommand "multi-tenant" "assign-user" "Assign user to company" "erpnext::multi_tenant::assign_user_to_company"
cli::register_subcommand "multi-tenant" "switch-company" "Switch active company" "erpnext::multi_tenant::switch_company"
cli::register_subcommand "multi-tenant" "get-data" "Get company-specific data" "erpnext::multi_tenant::get_company_data"
cli::register_subcommand "multi-tenant" "configure" "Configure company settings" "erpnext::multi_tenant::configure_company"

# Mobile UI commands
cli::register_command_group "mobile-ui" "Configure mobile-responsive UI"
cli::register_subcommand "mobile-ui" "enable" "Enable responsive UI" "erpnext::mobile_ui::enable_responsive"
cli::register_subcommand "mobile-ui" "configure-theme" "Configure mobile theme" "erpnext::mobile_ui::configure_theme"
cli::register_subcommand "mobile-ui" "configure-pwa" "Configure Progressive Web App" "erpnext::mobile_ui::configure_pwa"
cli::register_subcommand "mobile-ui" "configure-menu" "Configure mobile menu" "erpnext::mobile_ui::configure_mobile_menu"
cli::register_subcommand "mobile-ui" "optimize-touch" "Optimize for touch devices" "erpnext::mobile_ui::optimize_touch"
cli::register_subcommand "mobile-ui" "create-dashboard" "Create mobile dashboard" "erpnext::mobile_ui::create_mobile_dashboard"

# Inventory Management Module
cli::register_command_group "inventory" "Manage inventory and stock"
cli::register_subcommand "inventory" "list-items" "List inventory items" "erpnext::inventory::cli::list_items"
cli::register_subcommand "inventory" "add-item" "Add new inventory item" "erpnext::inventory::cli::add_item"
cli::register_subcommand "inventory" "check-stock" "Check stock balance" "erpnext::inventory::cli::check_stock"
cli::register_subcommand "inventory" "list-warehouses" "List warehouses" "erpnext::inventory::cli::list_warehouses"
cli::register_subcommand "inventory" "create-po" "Create purchase order" "erpnext::inventory::cli::create_po"

# Project Management Module
cli::register_command_group "projects" "Manage projects and tasks"
cli::register_subcommand "projects" "list" "List all projects" "erpnext::projects::cli::list"
cli::register_subcommand "projects" "create" "Create new project" "erpnext::projects::cli::create"
cli::register_subcommand "projects" "add-task" "Add task to project" "erpnext::projects::cli::add_task"
cli::register_subcommand "projects" "list-tasks" "List project tasks" "erpnext::projects::cli::list_tasks"
cli::register_subcommand "projects" "update-progress" "Update project progress" "erpnext::projects::cli::update_progress"
cli::register_subcommand "projects" "log-time" "Log time to project" "erpnext::projects::cli::log_time"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi