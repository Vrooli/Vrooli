#!/usr/bin/env bash
################################################################################
# Wiki.js Resource CLI - v2.0 Universal Contract Compliant
# 
# Modern wiki platform with Git storage backend
#
# Usage:
#   resource-wikijs <command> [options]
#   resource-wikijs <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    WIKIJS_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${WIKIJS_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
WIKIJS_CLI_DIR="${APP_ROOT}/resources/wikijs"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Source Wiki.js libraries
for lib in common install uninstall lifecycle status logs test api search backup git; do
    lib_file="${WIKIJS_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "wikijs" "Wiki.js platform management" "v2"

# Override default handlers to point directly to wikijs implementations
CLI_COMMAND_HANDLERS["manage::install"]="install_wikijs"
CLI_COMMAND_HANDLERS["manage::uninstall"]="uninstall_wikijs"
CLI_COMMAND_HANDLERS["manage::start"]="start_wikijs"  
CLI_COMMAND_HANDLERS["manage::stop"]="stop_wikijs"
CLI_COMMAND_HANDLERS["manage::restart"]="restart_wikijs"

# Test handlers - health checks only
CLI_COMMAND_HANDLERS["test::smoke"]="status_wikijs"
CLI_COMMAND_HANDLERS["test::integration"]="run_tests"
CLI_COMMAND_HANDLERS["test::all"]="run_tests"

# Content handlers for Wiki.js business functionality
CLI_COMMAND_HANDLERS["content::add"]="inject_content"
CLI_COMMAND_HANDLERS["content::list"]="search_content"
CLI_COMMAND_HANDLERS["content::get"]="api_command"
CLI_COMMAND_HANDLERS["content::remove"]="api_command"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "status_wikijs"
cli::register_command "logs" "Show Wiki.js logs" "show_logs"

# Add Wiki.js-specific content subcommands
cli::register_subcommand "content" "api" "Direct API access to Wiki.js" "api_command"
cli::register_subcommand "content" "search" "Search wiki content" "search_content"
cli::register_subcommand "content" "backup" "Create Wiki.js backup" "backup_wikijs"
cli::register_subcommand "content" "restore" "Restore Wiki.js backup" "restore_wikijs"
cli::register_subcommand "content" "git-sync" "Sync with Git repository" "sync_git"
cli::register_subcommand "content" "rebuild-search" "Rebuild search index" "rebuild_search_index"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi