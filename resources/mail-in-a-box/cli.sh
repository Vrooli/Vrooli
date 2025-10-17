#!/usr/bin/env bash
################################################################################
# Mail-in-a-Box Resource CLI - v2.0 Universal Contract Compliant
# Open source email & groupware server with easy setup
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    MAILINABOX_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${MAILINABOX_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
MAILINABOX_CLI_DIR="${APP_ROOT}/resources/mail-in-a-box"

# Source core dependencies
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Source config and libraries
[[ -f "${MAILINABOX_CLI_DIR}/config/defaults.sh" ]] && source "${MAILINABOX_CLI_DIR}/config/defaults.sh"
for lib in core install start stop status inject test content caldav domains autoconfig; do
    lib_file="${MAILINABOX_CLI_DIR}/lib/${lib}.sh"
    [[ -f "$lib_file" ]] && source "$lib_file" 2>/dev/null || true
done

# Initialize CLI framework in v2.0 mode
cli::init "mail-in-a-box" "Mail-in-a-Box email server management" "v2"

# Manage handlers (REQUIRED)
CLI_COMMAND_HANDLERS["manage::install"]="mailinabox_install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="mailinabox_uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="mailinabox_start"
CLI_COMMAND_HANDLERS["manage::stop"]="mailinabox_stop"
CLI_COMMAND_HANDLERS["manage::restart"]="mailinabox_restart"

# Restart handler function
mailinabox_restart() {
    mailinabox_stop
    mailinabox_start
}

# Test handlers (REQUIRED) - comprehensive test suite
CLI_COMMAND_HANDLERS["test::smoke"]="run_test_smoke"
CLI_COMMAND_HANDLERS["test::unit"]="run_test_unit"
CLI_COMMAND_HANDLERS["test::integration"]="run_test_integration"
CLI_COMMAND_HANDLERS["test::all"]="run_test_all"

# Content handlers (REQUIRED) - business functionality
CLI_COMMAND_HANDLERS["content::add"]="mailinabox_add_account"
CLI_COMMAND_HANDLERS["content::list"]="mailinabox_list_accounts"
CLI_COMMAND_HANDLERS["content::get"]="mailinabox_get_config"
CLI_COMMAND_HANDLERS["content::remove"]="mailinabox_delete_account"

# Custom content subcommands for email management
cli::register_subcommand "content" "add-alias" "Add email alias" "mailinabox_add_alias"
cli::register_subcommand "content" "add-domain" "Add custom domain" "mailinabox_add_domain"
cli::register_subcommand "content" "list-domains" "List configured domains" "mailinabox_list_domains"
cli::register_subcommand "content" "remove-domain" "Remove custom domain" "mailinabox_remove_domain"
cli::register_subcommand "content" "get-dkim" "Get DKIM key for domain" "mailinabox_get_dkim"
cli::register_subcommand "content" "verify-domain" "Verify domain DNS configuration" "mailinabox_verify_domain"
cli::register_subcommand "content" "setup-autoconfig" "Setup email client auto-configuration" "mailinabox_setup_autoconfig"
cli::register_subcommand "content" "test-autoconfig" "Test email client auto-configuration" "mailinabox_test_autoconfig"
cli::register_subcommand "content" "import" "Import email configuration from file" "mailinabox_inject_file"

# Information commands (REQUIRED)
cli::register_command "status" "Show detailed resource status" "mailinabox_simple_status"
cli::register_command "logs" "Show Mail-in-a-Box logs" "mailinabox_show_logs"
cli::register_command "version" "Show Mail-in-a-Box version" "mailinabox_get_version"

# Extended functionality commands
cli::register_command "monitor" "Monitor email server metrics" "mailinabox_monitor_wrapper"
cli::register_command "api" "REST API operations" "mailinabox_api_wrapper"
cli::register_command "credentials" "Show access information" "mailinabox_show_credentials"
cli::register_command "caldav" "CalDAV/CardDAV management" "mailinabox_caldav_wrapper"

# Simple status wrapper
mailinabox_simple_status() {
    echo -e "📧 Mail-in-a-Box Status\n"
    mailinabox_is_installed && echo "✅ Installed: Yes" || echo "❌ Installed: No"
    mailinabox_is_running && echo "✅ Running: Yes" || echo "⚠️  Running: No"
    [[ "$(mailinabox_get_health)" == "healthy" ]] && echo "✅ Health: Healthy" || echo "⚠️  Health: Unhealthy"
    echo -e "\nVersion: $(mailinabox_get_version)\nDetails: $(mailinabox_get_status_details)"
}

# Simple logs function
mailinabox_show_logs() {
    mailinabox_is_running && docker logs "$MAILINABOX_CONTAINER_NAME" --tail 50 || echo "Mail-in-a-Box is not running"
}

# Monitor wrapper
mailinabox_monitor_wrapper() {
    source "${MAILINABOX_CLI_DIR}/lib/monitor.sh" 2>/dev/null || {
        echo "Monitor module not available"
        return 1
    }
    mailinabox_monitor "$@"
}

# API wrapper
mailinabox_api_wrapper() {
    source "${MAILINABOX_CLI_DIR}/lib/api.sh" 2>/dev/null || {
        echo "API module not available"
        return 1
    }
    local subcmd="${1:-}"
    shift || true
    case "$subcmd" in
        serve)
            mailinabox_api_serve "${1:-8543}"
            ;;
        health|accounts|aliases|stats)
            mailinabox_api_handler "GET" "/$subcmd" "$@"
            ;;
        *)
            echo "Available API commands: serve, health, accounts, aliases, stats"
            ;;
    esac
}

# CalDAV wrapper
mailinabox_caldav_wrapper() {
    source "${MAILINABOX_CLI_DIR}/lib/caldav.sh" 2>/dev/null || {
        echo "CalDAV module not available"
        return 1
    }
    local subcmd="${1:-}"
    shift || true
    case "$subcmd" in
        add-user)
            caldav_create_user "$@"
            ;;
        list-users)
            caldav_list_users
            ;;
        delete-user)
            caldav_delete_user "$@"
            ;;
        test)
            caldav_test_connection "$@"
            ;;
        health)
            caldav_health
            ;;
        info)
            caldav_info
            ;;
        *)
            echo "Available CalDAV commands: add-user, list-users, delete-user, test, health, info"
            echo ""
            echo "Examples:"
            echo "  vrooli resource mail-in-a-box caldav add-user john@example.com password123"
            echo "  vrooli resource mail-in-a-box caldav list-users"
            echo "  vrooli resource mail-in-a-box caldav test john@example.com password123"
            ;;
    esac
}

# Credentials display
mailinabox_show_credentials() {
    echo -e "📧 Mail-in-a-Box Access Information\n"
    echo "Primary hostname: ${MAILINABOX_PRIMARY_HOSTNAME}"
    echo "Admin email: ${MAILINABOX_ADMIN_EMAIL}"
    echo "Data directory: ${MAILINABOX_DATA_DIR}"
    echo ""
    if docker inspect mailinabox-webmail &>/dev/null && [[ "$(docker inspect -f '{{.State.Running}}' mailinabox-webmail 2>/dev/null)" == "true" ]]; then
        echo "Webmail: http://localhost:8080"
    else
        echo "Webmail: Not installed (use docker-compose to enable)"
    fi
    if docker inspect mailinabox-caldav &>/dev/null && [[ "$(docker inspect -f '{{.State.Running}}' mailinabox-caldav 2>/dev/null)" == "true" ]]; then
        echo "CalDAV/CardDAV: http://localhost:5232"
    else
        echo "CalDAV/CardDAV: Not installed (use docker-compose to enable)"
    fi
    echo "API endpoint: http://localhost:8543/health"
    echo ""
    echo "Create your first email account with:"
    echo "  vrooli resource mail-in-a-box content add user@example.com"
}

# Test wrapper functions
run_test_smoke() {
    "${MAILINABOX_CLI_DIR}/test/run-tests.sh" smoke
}

run_test_unit() {
    "${MAILINABOX_CLI_DIR}/test/run-tests.sh" unit
}

run_test_integration() {
    "${MAILINABOX_CLI_DIR}/test/run-tests.sh" integration
}

run_test_all() {
    "${MAILINABOX_CLI_DIR}/test/run-tests.sh" all
}

# Execute if run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi