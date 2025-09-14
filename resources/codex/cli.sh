#!/usr/bin/env bash
################################################################################
# Codex Resource CLI - v2.0 Universal Contract Compliant
# 
# AI-powered code completion and generation via OpenAI Codex API
#
# Usage:
#   resource-codex <command> [options]
#   resource-codex <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    CODEX_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${CODEX_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
CODEX_CLI_DIR="${APP_ROOT}/resources/codex"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${CODEX_CLI_DIR}/config/defaults.sh"

# Source Codex libraries
for lib in common core status install docker test content inject codex-cli agents; do
    lib_file="${CODEX_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "codex" "AI-powered code completion and generation via OpenAI Codex" "v2"

# Override default handlers to point directly to codex implementations
CLI_COMMAND_HANDLERS["manage::install"]="codex::install::execute"
CLI_COMMAND_HANDLERS["manage::uninstall"]="codex::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="codex::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="codex::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="codex::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="codex::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="codex::test::integration"
CLI_COMMAND_HANDLERS["test::all"]="codex::test::all"

# Override content handlers for Codex-specific script management functionality
CLI_COMMAND_HANDLERS["content::add"]="codex::content::add"
CLI_COMMAND_HANDLERS["content::list"]="codex::content::list" 
CLI_COMMAND_HANDLERS["content::get"]="codex::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="codex::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="codex::content::execute"

# Add Codex-specific content subcommands not in the standard framework
cli::register_subcommand "content" "run" "Run a script with Codex API" "codex::run"
cli::register_subcommand "content" "inject" "Inject a script for Codex processing" "codex::inject"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "codex::status"
cli::register_command "logs" "Show Codex logs" "codex::docker::logs"

# Codex CLI management commands
cli::register_subcommand "manage" "install-cli" "Install OpenAI Codex CLI tool" "codex::cli::install"
cli::register_subcommand "manage" "update-cli" "Update Codex CLI to latest version" "codex::cli::update"
cli::register_subcommand "manage" "configure-cli" "Configure Codex CLI with API key" "codex::cli::configure"

# Agent commands (using Codex CLI when available)
cli::register_command "agent" "Run Codex agent on a task" "codex::cli::execute"
cli::register_command "fix" "Fix code issues using agent" "codex::cli::fix"
cli::register_command "generate-tests" "Generate tests for code" "codex::cli::test"
cli::register_command "refactor" "Refactor code using agent" "codex::cli::refactor"

# Agent management commands
cli::register_command "agents" "Manage running Codex agents" "codex::agents::command"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi