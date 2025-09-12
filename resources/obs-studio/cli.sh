#!/usr/bin/env bash
################################################################################
# OBS Studio Resource CLI - v2.0 Universal Contract Compliant
# 
# Open Broadcaster Software Studio for screen recording and live streaming
#
# Usage:
#   resource-obs-studio <command> [options]
#   resource-obs-studio <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    OBS_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${OBS_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
OBS_CLI_DIR="${APP_ROOT}/resources/obs-studio"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${OBS_CLI_DIR}/config/defaults.sh"

# Source OBS Studio libraries
for lib in common core install status start stop content test; do
    lib_file="${OBS_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "obs-studio" "OBS Studio screen recording and streaming platform" "v2"

# Source install script to get main function
source "${OBS_CLI_DIR}/lib/install.sh"

# Override default handlers to point directly to OBS Studio implementations
CLI_COMMAND_HANDLERS["manage::install"]="main"  # From install.sh
CLI_COMMAND_HANDLERS["manage::uninstall"]="obs::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="obs::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="obs::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="obs::restart"

# Override test handlers to use proper test functions
CLI_COMMAND_HANDLERS["test::smoke"]="obs::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="obs::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="obs::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="obs::test::all"

# Override content handlers for OBS Studio-specific recording/streaming functionality
CLI_COMMAND_HANDLERS["content::add"]="obs::content::add"
CLI_COMMAND_HANDLERS["content::list"]="obs::content::list" 
CLI_COMMAND_HANDLERS["content::get"]="obs::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="obs::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="obs::content::execute"

# Add OBS Studio-specific content subcommands
cli::register_subcommand "content" "scenes" "Manage OBS scenes" "obs::content::scenes"
cli::register_subcommand "content" "record" "Control recording" "obs::content::record"
cli::register_subcommand "content" "stream" "Control streaming" "obs::content::stream"

# Additional information commands
cli::register_command "status" "Show detailed OBS Studio status" "obs::get_status"
cli::register_command "logs" "Show OBS Studio logs" "obs::logs"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi