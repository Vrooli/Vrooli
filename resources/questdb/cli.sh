#!/usr/bin/env bash
################################################################################
# QuestDB Resource CLI - v2.0 Universal Contract Compliant
# 
# High-performance time-series database with SQL and real-time analytics
#
# Usage:
#   resource-questdb <command> [options]
#   resource-questdb <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    QUESTDB_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${QUESTDB_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
QUESTDB_CLI_DIR="${APP_ROOT}/resources/questdb"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${QUESTDB_CLI_DIR}/config/defaults.sh"

# Source QuestDB libraries
for lib in common docker install status api tables inject content core; do
    lib_file="${QUESTDB_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "questdb" "QuestDB high-performance time-series database management" "v2"

# Override default handlers to point directly to QuestDB implementations
CLI_COMMAND_HANDLERS["manage::install"]="questdb::install::run"
CLI_COMMAND_HANDLERS["manage::uninstall"]="questdb::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="questdb::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="questdb::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="questdb::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="questdb::status::check"

# Override content handlers for QuestDB-specific time-series functionality
CLI_COMMAND_HANDLERS["content::add"]="questdb::content::add"
CLI_COMMAND_HANDLERS["content::list"]="questdb::tables::list" 
CLI_COMMAND_HANDLERS["content::get"]="questdb::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="questdb::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="questdb::content::execute"

# Add QuestDB-specific content subcommands not in the standard framework
cli::register_subcommand "content" "query" "Execute SQL query" "questdb::content::query"
cli::register_subcommand "content" "inject" "Import data (SQL/CSV/JSON)" "questdb::content::inject" "modifies-system"
cli::register_subcommand "content" "tables" "Manage tables" "questdb::content::tables"
cli::register_subcommand "content" "api" "Make API request" "questdb::content::api"
cli::register_subcommand "content" "console" "Open web console" "questdb::content::console"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "questdb::status::check"
cli::register_command "logs" "Show QuestDB logs" "questdb::docker::logs"
cli::register_command "credentials" "Show QuestDB credentials for integration" "questdb::core::credentials"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi