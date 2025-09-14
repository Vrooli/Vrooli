#!/usr/bin/env bash
################################################################################
# Cline Resource CLI - v2.0 Universal Contract Compliant
# 
# AI coding assistant for VS Code with provider flexibility
#
# Usage:
#   resource-cline <command> [options]
#   resource-cline <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    CLINE_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${CLINE_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
CLINE_CLI_DIR="${APP_ROOT}/resources/cline"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${CLINE_CLI_DIR}/config/defaults.sh"

# Source Cline libraries
for lib in common status install start stop logs config inject test content agents; do
    lib_file="${CLINE_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "cline" "Cline AI coding assistant management" "v2"

# Override default handlers to point directly to cline implementations
CLI_COMMAND_HANDLERS["manage::install"]="cline::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="cline::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="cline::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="cline::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="cline::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="cline::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="cline::test::integration"
CLI_COMMAND_HANDLERS["test::all"]="cline::test::all"

# Override content handlers for Cline-specific functionality
CLI_COMMAND_HANDLERS["content::add"]="cline::content::add"
CLI_COMMAND_HANDLERS["content::list"]="cline::content::list" 
CLI_COMMAND_HANDLERS["content::get"]="cline::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="cline::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="cline::content::execute"

# Add Cline-specific content subcommands
cli::register_subcommand "content" "configure" "Configure Cline provider settings" "cline::configure"
cli::register_subcommand "content" "provider" "Manage API provider settings" "cline::config_provider"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "cline::status"
cli::register_command "logs" "Show Cline logs" "cline::logs"
cli::register_command "config" "View/update configuration" "cline::config"

# Agent management commands
cli::register_command "agents" "Manage running Cline agents" "cline::agents::command"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi