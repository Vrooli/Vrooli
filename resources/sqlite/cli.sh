#!/usr/bin/env bash
################################################################################
# SQLite Resource CLI - v2.0 Universal Contract Compliant
# 
# Lightweight, serverless SQL database engine for local data persistence
#
# Usage:
#   resource-sqlite <command> [options]
#   resource-sqlite <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    SQLITE_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${SQLITE_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
SQLITE_CLI_DIR="${APP_ROOT}/resources/sqlite"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${SQLITE_CLI_DIR}/config/defaults.sh"

# Source SQLite libraries
for lib in core test; do
    lib_file="${SQLITE_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "sqlite" "Lightweight serverless SQL database engine" "v2"

# Register status and logs as flat commands (required by v2.0 contract)
cli::register_command "status" "Show resource status" "cli::_handle_status"
cli::register_command "logs" "Show resource logs" "cli::_handle_logs"

# Override default handlers to point directly to SQLite implementations
CLI_COMMAND_HANDLERS["manage::install"]="sqlite::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="sqlite::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="sqlite::start"
CLI_COMMAND_HANDLERS["manage::stop"]="sqlite::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="sqlite::restart"

CLI_COMMAND_HANDLERS["test::smoke"]="sqlite::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="sqlite::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="sqlite::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="sqlite::test::all"

CLI_COMMAND_HANDLERS["content::create"]="sqlite::content::create"
CLI_COMMAND_HANDLERS["content::execute"]="sqlite::content::execute"
CLI_COMMAND_HANDLERS["content::list"]="sqlite::content::list"
CLI_COMMAND_HANDLERS["content::get"]="sqlite::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="sqlite::content::remove"
CLI_COMMAND_HANDLERS["content::backup"]="sqlite::content::backup"
CLI_COMMAND_HANDLERS["content::restore"]="sqlite::content::restore"

CLI_COMMAND_HANDLERS["status"]="sqlite::status"
CLI_COMMAND_HANDLERS["logs"]="sqlite::logs"
CLI_COMMAND_HANDLERS["info"]="sqlite::info"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi