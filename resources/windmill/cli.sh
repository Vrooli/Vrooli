#!/usr/bin/env bash
################################################################################
# Windmill Resource CLI - v2.0 Universal Contract Compliant
# 
# Developer platform for workflows, scripts, and automation
#
# Usage:
#   resource-windmill <command> [options]
#   resource-windmill <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    WINDMILL_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${WINDMILL_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
WINDMILL_CLI_DIR="${APP_ROOT}/resources/windmill"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${WINDMILL_CLI_DIR}/config/defaults.sh"

# Source Windmill libraries
for lib in state common database docker status install apps api workers content; do
    lib_file="${WINDMILL_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "windmill" "Windmill developer platform and workflow automation" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="windmill::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="windmill::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="windmill::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="windmill::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="windmill::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="windmill::quick_status"

# Content handlers - for Windmill's business functionality (apps, workflows)
CLI_COMMAND_HANDLERS["content::add"]="windmill::content::add"
CLI_COMMAND_HANDLERS["content::list"]="windmill::content::list"
CLI_COMMAND_HANDLERS["content::get"]="windmill::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="windmill::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="windmill::content::execute"

# Add Windmill-specific content subcommands
cli::register_subcommand "content" "prepare" "Prepare app for deployment" "windmill::prepare_app" "modifies-system"
cli::register_subcommand "content" "deploy" "Deploy app to workspace" "windmill::deploy_app" "modifies-system"
cli::register_subcommand "content" "backup" "Backup Windmill data" "windmill::backup" "modifies-system"
cli::register_subcommand "content" "restore" "Restore from backup" "windmill::restore" "modifies-system"

# Add worker management as content operations
cli::register_subcommand "content" "scale-workers" "Scale worker containers" "windmill::scale_workers" "modifies-system"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed resource status" "windmill::status"
cli::register_command "logs" "Show Windmill logs" "windmill::show_logs"
cli::register_command "credentials" "Show Windmill credentials for integration" "windmill::credentials"

# Only execute if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi