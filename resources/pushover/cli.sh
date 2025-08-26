#!/usr/bin/env bash
################################################################################
# Pushover Resource CLI - v2.0 Universal Contract Compliant
# 
# Cloud notification service integration for sending push notifications
#
# Usage:
#   resource-pushover <command> [options]
#   resource-pushover <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    PUSHOVER_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${PUSHOVER_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
PUSHOVER_CLI_DIR="${APP_ROOT}/resources/pushover"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Source Pushover libraries
for lib in core install configure start inject status; do
    lib_file="${PUSHOVER_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "pushover" "Pushover notification service management" "v2"

# Override default handlers to point directly to pushover implementations
CLI_COMMAND_HANDLERS["manage::install"]="pushover::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="pushover::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="pushover::start"
CLI_COMMAND_HANDLERS["manage::stop"]="pushover::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="pushover::restart"

# Test group handlers (health checks)
CLI_COMMAND_HANDLERS["test::smoke"]="pushover::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="pushover::test::integration"

# Content group handlers (business functionality)
CLI_COMMAND_HANDLERS["content::add"]="pushover::inject"  # Add notification templates
CLI_COMMAND_HANDLERS["content::list"]="pushover::list_templates"
CLI_COMMAND_HANDLERS["content::get"]="pushover::get_template"
CLI_COMMAND_HANDLERS["content::remove"]="pushover::remove_template"

# Additional Pushover-specific commands
cli::register_command "status" "Show detailed resource status" "pushover::status"
cli::register_command "logs" "Show Pushover logs" "pushover::show_logs"

# Pushover-specific custom commands
cli::register_command "configure" "Configure API credentials" "pushover::configure"
cli::register_command "send" "Send a notification" "pushover::send"
cli::register_command "clear-credentials" "Clear stored credentials" "pushover::clear_credentials"
cli::register_command "enable-demo" "Enable demo mode for testing" "pushover::enable_demo_mode"
cli::register_command "disable-demo" "Disable demo mode" "pushover::disable_demo_mode"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi