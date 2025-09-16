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

# Determine script directory properly handling both direct and sourced execution
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
APP_ROOT="${APP_ROOT:-$(builtin cd "${SCRIPT_DIR}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    KEYCLOAK_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    SCRIPT_DIR="$( cd "$( dirname "${KEYCLOAK_CLI_SCRIPT}" )" && pwd )"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${SCRIPT_DIR}/../.." && builtin pwd)"
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
for lib in common install lifecycle status inject content social-providers ldap-federation multi-realm backup monitor theme tls mfa password-policy; do
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
# Test runner functions - must be defined before handler registration
keycloak::test::smoke() {
    "${KEYCLOAK_CLI_DIR}/test/run-tests.sh" smoke
}

keycloak::test::integration() {
    "${KEYCLOAK_CLI_DIR}/test/run-tests.sh" integration
}

keycloak::test::unit() {
    "${KEYCLOAK_CLI_DIR}/test/run-tests.sh" unit
}

keycloak::test::all() {
    "${KEYCLOAK_CLI_DIR}/test/run-tests.sh" all
}

keycloak::test::multirealm() {
    "${KEYCLOAK_CLI_DIR}/test/run-tests.sh" multi-realm
}

# Test handlers - delegate to test runner
CLI_COMMAND_HANDLERS["test::smoke"]="keycloak::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="keycloak::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="keycloak::test::unit"
CLI_COMMAND_HANDLERS["test::multi-realm"]="keycloak::test::multirealm"
CLI_COMMAND_HANDLERS["test::all"]="keycloak::test::all"

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

# Register social command group
CLI_COMMAND_GROUPS["social"]="true"
CLI_GROUP_DESCRIPTIONS["social"]="🔐 Social login providers"

# Social provider management commands
CLI_COMMAND_HANDLERS["social::add-github"]="keycloak::social::add_github"
CLI_COMMAND_HANDLERS["social::add-google"]="keycloak::social::add_google"
CLI_COMMAND_HANDLERS["social::add-facebook"]="keycloak::social::add_facebook"
CLI_COMMAND_HANDLERS["social::list"]="keycloak::social::list"
CLI_COMMAND_HANDLERS["social::remove"]="keycloak::social::remove"
CLI_COMMAND_HANDLERS["social::test"]="keycloak::social::test"

cli::register_subcommand "social" "add-github" "Add GitHub login provider" "keycloak::social::add_github"
cli::register_subcommand "social" "add-google" "Add Google login provider" "keycloak::social::add_google"
cli::register_subcommand "social" "add-facebook" "Add Facebook login provider" "keycloak::social::add_facebook"
cli::register_subcommand "social" "list" "List configured social providers" "keycloak::social::list"
cli::register_subcommand "social" "remove" "Remove a social provider" "keycloak::social::remove"
cli::register_subcommand "social" "test" "Test social provider configuration" "keycloak::social::test"

# Register ldap command group
CLI_COMMAND_GROUPS["ldap"]="true"
CLI_GROUP_DESCRIPTIONS["ldap"]="🏢 LDAP/AD federation"

# LDAP/AD federation management commands
CLI_COMMAND_HANDLERS["ldap::add"]="keycloak::ldap::add"
CLI_COMMAND_HANDLERS["ldap::list"]="keycloak::ldap::list"
CLI_COMMAND_HANDLERS["ldap::remove"]="keycloak::ldap::remove"
CLI_COMMAND_HANDLERS["ldap::test"]="keycloak::ldap::test"
CLI_COMMAND_HANDLERS["ldap::sync"]="keycloak::ldap::sync"

cli::register_subcommand "ldap" "add" "Add LDAP/AD federation provider" "keycloak::ldap::add"
cli::register_subcommand "ldap" "list" "List configured LDAP/AD providers" "keycloak::ldap::list"
cli::register_subcommand "ldap" "remove" "Remove an LDAP/AD provider" "keycloak::ldap::remove"
cli::register_subcommand "ldap" "test" "Test LDAP/AD connection" "keycloak::ldap::test"
cli::register_subcommand "ldap" "sync" "Sync users from LDAP/AD" "keycloak::ldap::sync"

# Register realm command group for multi-tenant management
CLI_COMMAND_GROUPS["realm"]="true"
CLI_GROUP_DESCRIPTIONS["realm"]="🏢 Multi-realm tenant isolation"

# Realm management commands
CLI_COMMAND_HANDLERS["realm::create-tenant"]="keycloak::realm::create_tenant"
CLI_COMMAND_HANDLERS["realm::list-tenants"]="keycloak::realm::list_tenants"
CLI_COMMAND_HANDLERS["realm::get-tenant"]="keycloak::realm::get_tenant"
CLI_COMMAND_HANDLERS["realm::delete-tenant"]="keycloak::realm::delete_tenant"
CLI_COMMAND_HANDLERS["realm::export-tenant"]="keycloak::realm::export_tenant"

cli::register_subcommand "realm" "create-tenant" "Create isolated tenant realm" "keycloak::realm::create_tenant"
cli::register_subcommand "realm" "list-tenants" "List all tenant realms" "keycloak::realm::list_tenants"
cli::register_subcommand "realm" "get-tenant" "Get tenant realm details" "keycloak::realm::get_tenant"
cli::register_subcommand "realm" "delete-tenant" "Delete tenant realm" "keycloak::realm::delete_tenant"
cli::register_subcommand "realm" "export-tenant" "Export tenant realm configuration" "keycloak::realm::export_tenant"

# Register backup command group for backup and restore operations
CLI_COMMAND_GROUPS["backup"]="true"
CLI_GROUP_DESCRIPTIONS["backup"]="💾 Backup and restore operations"

# Backup management commands
CLI_COMMAND_HANDLERS["backup::create"]="backup::create"
CLI_COMMAND_HANDLERS["backup::list"]="backup::list"
CLI_COMMAND_HANDLERS["backup::restore"]="backup::restore"
CLI_COMMAND_HANDLERS["backup::cleanup"]="backup::cleanup"
CLI_COMMAND_HANDLERS["backup::schedule"]="backup::schedule"

cli::register_subcommand "backup" "create" "Create backup of a realm" "backup::create"
cli::register_subcommand "backup" "list" "List available backups" "backup::list"
cli::register_subcommand "backup" "restore" "Restore realm from backup" "backup::restore"
cli::register_subcommand "backup" "cleanup" "Remove old backups" "backup::cleanup"
cli::register_subcommand "backup" "schedule" "Schedule automatic backups" "backup::schedule"

# Register monitor command group for performance monitoring
CLI_COMMAND_GROUPS["monitor"]="true"
CLI_GROUP_DESCRIPTIONS["monitor"]="📊 Performance monitoring and metrics"

# Monitor commands
CLI_COMMAND_HANDLERS["monitor::health"]="monitor::health"
CLI_COMMAND_HANDLERS["monitor::metrics"]="monitor::metrics"
CLI_COMMAND_HANDLERS["monitor::performance"]="monitor::performance"
CLI_COMMAND_HANDLERS["monitor::realms"]="monitor::realms"
CLI_COMMAND_HANDLERS["monitor::dashboard"]="monitor::dashboard"

cli::register_subcommand "monitor" "health" "Check health status" "monitor::health"
cli::register_subcommand "monitor" "metrics" "Show performance metrics" "monitor::metrics"
cli::register_subcommand "monitor" "performance" "Analyze performance" "monitor::performance"
cli::register_subcommand "monitor" "realms" "Show realm statistics" "monitor::realms"
cli::register_subcommand "monitor" "dashboard" "Full monitoring dashboard" "monitor::dashboard"

# Register theme command group for theme customization
CLI_COMMAND_GROUPS["theme"]="true"
CLI_GROUP_DESCRIPTIONS["theme"]="🎨 Theme customization and branding"

# Theme management commands
CLI_COMMAND_HANDLERS["theme::create"]="theme::create"
CLI_COMMAND_HANDLERS["theme::list"]="theme::list"
CLI_COMMAND_HANDLERS["theme::deploy"]="theme::deploy"
CLI_COMMAND_HANDLERS["theme::apply"]="theme::apply"
CLI_COMMAND_HANDLERS["theme::remove"]="theme::remove"
CLI_COMMAND_HANDLERS["theme::customize"]="theme::customize"
CLI_COMMAND_HANDLERS["theme::export"]="theme::export"
CLI_COMMAND_HANDLERS["theme::import"]="theme::import"

cli::register_subcommand "theme" "create" "Create a new theme" "theme::create"
cli::register_subcommand "theme" "list" "List available themes" "theme::list"
cli::register_subcommand "theme" "deploy" "Deploy theme to container" "theme::deploy"
cli::register_subcommand "theme" "apply" "Apply theme to a realm" "theme::apply"
cli::register_subcommand "theme" "remove" "Remove a theme" "theme::remove"
cli::register_subcommand "theme" "customize" "Customize theme properties" "theme::customize"
cli::register_subcommand "theme" "export" "Export theme as archive" "theme::export"
cli::register_subcommand "theme" "import" "Import theme from archive" "theme::import"

# Register TLS command group for HTTPS/certificate management
CLI_COMMAND_GROUPS["tls"]="true"
CLI_GROUP_DESCRIPTIONS["tls"]="🔒 TLS/HTTPS certificate management"

# TLS management commands
CLI_COMMAND_HANDLERS["tls::generate"]="keycloak::tls::generate_self_signed"
CLI_COMMAND_HANDLERS["tls::import"]="keycloak::tls::import_certificate"
CLI_COMMAND_HANDLERS["tls::enable"]="keycloak::tls::enable_https"
CLI_COMMAND_HANDLERS["tls::disable"]="keycloak::tls::disable_https"
CLI_COMMAND_HANDLERS["tls::check"]="keycloak::tls::check_expiry"
CLI_COMMAND_HANDLERS["tls::show"]="keycloak::tls::show_certificate"
CLI_COMMAND_HANDLERS["tls::renew"]="keycloak::tls::renew_certificate"

cli::register_subcommand "tls" "generate" "Generate self-signed certificate" "keycloak::tls::generate_self_signed"
cli::register_subcommand "tls" "import" "Import existing certificate" "keycloak::tls::import_certificate"
cli::register_subcommand "tls" "enable" "Enable HTTPS for Keycloak" "keycloak::tls::enable_https"
cli::register_subcommand "tls" "disable" "Disable HTTPS (HTTP only)" "keycloak::tls::disable_https"
cli::register_subcommand "tls" "check" "Check certificate expiry" "keycloak::tls::check_expiry"
cli::register_subcommand "tls" "show" "Show certificate details" "keycloak::tls::show_certificate"
cli::register_subcommand "tls" "renew" "Renew certificate" "keycloak::tls::renew_certificate"

# Register MFA command group for multi-factor authentication
CLI_COMMAND_GROUPS["mfa"]="true"
CLI_GROUP_DESCRIPTIONS["mfa"]="🔐 Multi-factor authentication management"

# MFA management commands
CLI_COMMAND_HANDLERS["mfa::enable"]="keycloak::mfa::enable"
CLI_COMMAND_HANDLERS["mfa::disable"]="keycloak::mfa::disable"
CLI_COMMAND_HANDLERS["mfa::configure"]="keycloak::mfa::configure_policy"
CLI_COMMAND_HANDLERS["mfa::enable-user"]="keycloak::mfa::enable_for_user"
CLI_COMMAND_HANDLERS["mfa::status"]="keycloak::mfa::status"
CLI_COMMAND_HANDLERS["mfa::list-users"]="keycloak::mfa::list_users"
CLI_COMMAND_HANDLERS["mfa::backup-codes"]="keycloak::mfa::generate_backup_codes"

cli::register_subcommand "mfa" "enable" "Enable MFA for realm" "keycloak::mfa::enable"
cli::register_subcommand "mfa" "disable" "Disable MFA for realm" "keycloak::mfa::disable"
cli::register_subcommand "mfa" "configure" "Configure MFA policy" "keycloak::mfa::configure_policy"
cli::register_subcommand "mfa" "enable-user" "Enable MFA for specific user" "keycloak::mfa::enable_for_user"
cli::register_subcommand "mfa" "status" "Show MFA status" "keycloak::mfa::status"
cli::register_subcommand "mfa" "list-users" "List users with MFA status" "keycloak::mfa::list_users"
cli::register_subcommand "mfa" "backup-codes" "Generate backup codes" "keycloak::mfa::generate_backup_codes"

# Register password-policy command group
CLI_COMMAND_GROUPS["password-policy"]="true"
CLI_GROUP_DESCRIPTIONS["password-policy"]="🔑 Password policy management"

# Password policy commands
CLI_COMMAND_HANDLERS["password-policy::set"]="keycloak::password_policy::set"
CLI_COMMAND_HANDLERS["password-policy::get"]="keycloak::password_policy::get"
CLI_COMMAND_HANDLERS["password-policy::clear"]="keycloak::password_policy::clear"
CLI_COMMAND_HANDLERS["password-policy::preset"]="keycloak::password_policy::apply_preset"
CLI_COMMAND_HANDLERS["password-policy::validate"]="keycloak::password_policy::validate"
CLI_COMMAND_HANDLERS["password-policy::force-reset"]="keycloak::password_policy::force_reset"

cli::register_subcommand "password-policy" "set" "Set password policy" "keycloak::password_policy::set"
cli::register_subcommand "password-policy" "get" "Show current policy" "keycloak::password_policy::get"
cli::register_subcommand "password-policy" "clear" "Clear password policy" "keycloak::password_policy::clear"
cli::register_subcommand "password-policy" "preset" "Apply preset policy (basic/moderate/strong/paranoid)" "keycloak::password_policy::apply_preset"
cli::register_subcommand "password-policy" "validate" "Validate password against policy" "keycloak::password_policy::validate"
cli::register_subcommand "password-policy" "force-reset" "Force password reset for users" "keycloak::password_policy::force_reset"

# Let's Encrypt certificate automation
# Register Let's Encrypt command group
CLI_COMMAND_GROUPS["letsencrypt"]="true"
CLI_GROUP_DESCRIPTIONS["letsencrypt"]="🔓 Let's Encrypt certificate automation"

# Source Let's Encrypt library with proper guards
if [[ -f "${KEYCLOAK_CLI_DIR}/lib/letsencrypt.sh" ]]; then
    # Only source if functions aren't already defined
    if ! declare -f keycloak::letsencrypt::init &>/dev/null; then
        # shellcheck disable=SC1090
        source "${KEYCLOAK_CLI_DIR}/lib/letsencrypt.sh" 2>/dev/null || true
    fi
fi

# Register Let's Encrypt handlers if functions are available
if declare -f keycloak::letsencrypt::init &>/dev/null; then
    CLI_COMMAND_HANDLERS["letsencrypt::init"]="keycloak::letsencrypt::init"
    CLI_COMMAND_HANDLERS["letsencrypt::request"]="keycloak::letsencrypt::request_certificate"
    CLI_COMMAND_HANDLERS["letsencrypt::renew"]="keycloak::letsencrypt::renew_certificates"
    CLI_COMMAND_HANDLERS["letsencrypt::auto-renew"]="keycloak::letsencrypt::setup_auto_renewal"
    CLI_COMMAND_HANDLERS["letsencrypt::disable-auto-renew"]="keycloak::letsencrypt::remove_auto_renewal"
    CLI_COMMAND_HANDLERS["letsencrypt::status"]="keycloak::letsencrypt::check_status"
    CLI_COMMAND_HANDLERS["letsencrypt::revoke"]="keycloak::letsencrypt::revoke_certificate"
    CLI_COMMAND_HANDLERS["letsencrypt::test"]="keycloak::letsencrypt::test_challenge"

    cli::register_subcommand "letsencrypt" "init" "Initialize Let's Encrypt with email" "keycloak::letsencrypt::init"
    cli::register_subcommand "letsencrypt" "request" "Request certificate for domain" "keycloak::letsencrypt::request_certificate"
    cli::register_subcommand "letsencrypt" "renew" "Renew certificates" "keycloak::letsencrypt::renew_certificates"
    cli::register_subcommand "letsencrypt" "auto-renew" "Setup automatic renewal (daily/weekly/monthly)" "keycloak::letsencrypt::setup_auto_renewal"
    cli::register_subcommand "letsencrypt" "disable-auto-renew" "Disable automatic renewal" "keycloak::letsencrypt::remove_auto_renewal"
    cli::register_subcommand "letsencrypt" "status" "Check certificate status" "keycloak::letsencrypt::check_status"
    cli::register_subcommand "letsencrypt" "revoke" "Revoke certificate for domain" "keycloak::letsencrypt::revoke_certificate"
    cli::register_subcommand "letsencrypt" "test" "Test ACME challenge setup" "keycloak::letsencrypt::test_challenge"
fi

# Additional information commands
cli::register_command "status" "Show detailed resource status" "keycloak::status"
cli::register_command "logs" "Show Keycloak logs" "keycloak::logs"
cli::register_command "credentials" "Show Keycloak admin credentials" "keycloak::credentials"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi