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

# Source agent management (load config and manager directly)
if [[ -f "${APP_ROOT}/resources/opencode/config/agents.conf" ]]; then
    source "${APP_ROOT}/resources/opencode/config/agents.conf"
    source "${APP_ROOT}/scripts/resources/agents/agent-manager.sh"
fi

#######################################
# Setup agent cleanup on signals
# Arguments:
#   $1 - Agent ID
#######################################
opencode::setup_agent_cleanup() {
    local agent_id="$1"
    
    # Export the agent ID so trap can access it
    export OPENCODE_CURRENT_AGENT_ID="$agent_id"
    
    # Cleanup function that uses the exported variable
    opencode::agent_cleanup() {
        if [[ -n "${OPENCODE_CURRENT_AGENT_ID:-}" ]] && type -t agent_manager::unregister &>/dev/null; then
            agent_manager::unregister "${OPENCODE_CURRENT_AGENT_ID}" >/dev/null 2>&1
        fi
        exit 0
    }
    
    # Register cleanup for common signals
    trap 'opencode::agent_cleanup' EXIT SIGTERM SIGINT
}

# Source resource configuration
source "${OPENCODE_CLI_DIR}/config/defaults.sh"

# Source resource libraries
for lib in common install status docker content test agents; do
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
# Create wrapper for agents command that delegates to manager
opencode::agents::command() {
    if type -t agent_manager::load_config &>/dev/null; then
        "${APP_ROOT}/scripts/resources/agents/agent-manager.sh" --config="opencode" "$@"
    else
        log::error "Agent management not available"
        return 1
    fi
}
export -f opencode::agents::command

cli::register_command "agents" "Manage running opencode agents" "opencode::agents::command"

# ==============================================================================
# OPENCODE-SPECIFIC CUSTOM COMMANDS
# ==============================================================================
# Custom content subcommand for activating configurations
cli::register_subcommand "content" "activate" "Activate a saved configuration" "opencode::content::activate"

# Only execute if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi