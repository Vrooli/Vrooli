#!/usr/bin/env bash
################################################################################
# OpenCode Resource CLI - v2.0 Universal Contract Compliant
# 
# AI-powered code completion and chat via VS Code Twinny extension
#
# Usage:
#   resource-opencode <command> [options]
#   resource-opencode <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    OPENCODE_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${OPENCODE_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
OPENCODE_CLI_DIR="${APP_ROOT}/resources/opencode"

# Source standard variables
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source utilities
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"

# Source v2.0 CLI Command Framework
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Source resource configuration
source "${OPENCODE_CLI_DIR}/config/defaults.sh"

# Source resource libraries
for lib in common install status docker content test; do
    lib_file="${OPENCODE_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "opencode" "AI-powered code completion via VS Code Twinny extension" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="opencode::install::execute"
CLI_COMMAND_HANDLERS["manage::uninstall"]="opencode::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="opencode::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="opencode::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="opencode::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="opencode::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="opencode::test::integration"
CLI_COMMAND_HANDLERS["test::all"]="opencode::test::all"

# Content handlers for AI model and configuration management
CLI_COMMAND_HANDLERS["content::add"]="opencode::content::add"
CLI_COMMAND_HANDLERS["content::list"]="opencode::content::list"
CLI_COMMAND_HANDLERS["content::get"]="opencode::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="opencode::content::remove"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed OpenCode status" "opencode::status"
cli::register_command "logs" "Show VS Code and extension logs" "opencode::docker::logs"

# ==============================================================================
# OPENCODE-SPECIFIC CUSTOM COMMANDS
# ==============================================================================
# Custom content subcommand for activating configurations
cli::register_subcommand "content" "activate" "Activate a saved configuration" "opencode::content::activate"

# Only execute if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi