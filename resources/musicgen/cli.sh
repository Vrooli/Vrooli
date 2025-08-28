#!/usr/bin/env bash
################################################################################
# MusicGen Resource CLI - v2.0 Universal Contract Compliant
# 
# Meta's AI music generation model for creating original music
#
# Usage:
#   resource-musicgen <command> [options]
#   resource-musicgen <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    MUSICGEN_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${MUSICGEN_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
MUSICGEN_CLI_DIR="${APP_ROOT}/resources/musicgen"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${MUSICGEN_CLI_DIR}/config/defaults.sh"

# Source MusicGen libraries
for lib in common core status docker install content test; do
    lib_file="${MUSICGEN_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "musicgen" "MusicGen AI music generation platform management" "v2"

# Override default handlers to point directly to musicgen implementations
CLI_COMMAND_HANDLERS["manage::install"]="musicgen::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="musicgen::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="musicgen::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="musicgen::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="musicgen::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="musicgen::test::smoke"

# Override content handlers for MusicGen-specific music generation functionality
CLI_COMMAND_HANDLERS["content::add"]="musicgen::content::add"
CLI_COMMAND_HANDLERS["content::list"]="musicgen::content::list" 
CLI_COMMAND_HANDLERS["content::get"]="musicgen::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="musicgen::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="musicgen::generate"

# Add MusicGen-specific content subcommands not in the standard framework
cli::register_subcommand "content" "models" "List available music generation models" "musicgen::list_models"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "musicgen::status"
cli::register_command "logs" "Show MusicGen logs" "musicgen::logs"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi