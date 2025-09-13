#!/usr/bin/env bash
################################################################################
# Home Assistant Resource CLI - v2.0 Universal Contract Compliant
# 
# Open source home automation platform
#
# Usage:
#   resource-home-assistant <command> [options]
#   resource-home-assistant <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    HOME_ASSISTANT_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${HOME_ASSISTANT_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
HOME_ASSISTANT_CLI_DIR="${APP_ROOT}/resources/home-assistant"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${HOME_ASSISTANT_CLI_DIR}/config/defaults.sh"

# Source Home Assistant libraries
for lib in core health install status inject test components; do
    lib_file="${HOME_ASSISTANT_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "home-assistant" "Home Assistant automation platform management" "v2"

# Override default handlers to point directly to home-assistant implementations
CLI_COMMAND_HANDLERS["manage::install"]="home_assistant::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="home_assistant::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="home_assistant::docker::start"
CLI_COMMAND_HANDLERS["manage::stop"]="home_assistant::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="home_assistant::docker::restart"

# Test handlers for Home Assistant health checks
CLI_COMMAND_HANDLERS["test::smoke"]="home_assistant::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="home_assistant::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="home_assistant::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="home_assistant::test::all"

# Content handlers for Home Assistant automation functionality
CLI_COMMAND_HANDLERS["content::add"]="home_assistant::inject"
CLI_COMMAND_HANDLERS["content::list"]="home_assistant::inject::list"
CLI_COMMAND_HANDLERS["content::remove"]="home_assistant::inject::clear"
CLI_COMMAND_HANDLERS["content::execute"]="home_assistant::reload_automations"

# Home Assistant doesn't have a single "get" for content, but we can show config
CLI_COMMAND_HANDLERS["content::get"]="home_assistant::export_config"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "home_assistant::status"
cli::register_command "logs" "Show Home Assistant logs" "home_assistant::docker::logs"

# Custom Home Assistant commands for API access and configuration
cli::register_command "api-info" "Show Home Assistant API information" "home_assistant::get_api_info"
cli::register_subcommand "content" "reload" "Reload automations without restart" "home_assistant::reload_automations"
cli::register_subcommand "content" "export" "Export current configuration" "home_assistant::export_config"

# Backup and restore commands - backup is a group with subcommands
cli::register_command_group "backup" "Backup and restore operations"
cli::register_subcommand "backup" "create" "Create a new backup" "home_assistant::backup"
cli::register_subcommand "backup" "list" "List available backups" "home_assistant::backup::list"
cli::register_subcommand "backup" "restore" "Restore from a backup" "home_assistant::restore"

# Custom components management commands
cli::register_command_group "components" "Custom components management"
cli::register_subcommand "components" "list" "List installed custom components" "home_assistant::components::list"
cli::register_subcommand "components" "install-hacs" "Install Home Assistant Community Store" "home_assistant::components::install_hacs"
cli::register_subcommand "components" "add" "Install component from GitHub" "home_assistant::components::install_from_github"
cli::register_subcommand "components" "remove" "Remove a custom component" "home_assistant::components::remove"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi