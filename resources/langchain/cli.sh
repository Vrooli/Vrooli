#!/usr/bin/env bash
################################################################################
# LangChain Resource CLI - v2.0 Universal Contract Compliant
# 
# Framework for developing applications with Large Language Models (LLMs)
#
# Usage:
#   resource-langchain <command> [options]
#   resource-langchain <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    LANGCHAIN_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${LANGCHAIN_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
LANGCHAIN_CLI_DIR="${APP_ROOT}/resources/langchain"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${LANGCHAIN_CLI_DIR}/config/defaults.sh"

# Source agent management (load config and manager directly)
if [[ -f "${APP_ROOT}/resources/langchain/config/agents.conf" ]]; then
    source "${APP_ROOT}/resources/langchain/config/agents.conf"
    source "${APP_ROOT}/scripts/resources/agents/agent-manager.sh"
fi

# Source LangChain libraries
for lib in core install status content agents; do
    lib_file="${LANGCHAIN_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "langchain" "LangChain LLM application framework" "v2"

#######################################
# Setup agent cleanup trap
# Arguments:
#   $1 - Agent ID
#######################################
langchain::setup_agent_cleanup() {
    local agent_id="$1"
    
    # Export the agent ID so trap can access it
    export LANGCHAIN_CURRENT_AGENT_ID="$agent_id"
    
    # Cleanup function that uses the exported variable
    langchain::agent_cleanup() {
        if [[ -n "${LANGCHAIN_CURRENT_AGENT_ID:-}" ]] && type -t agent_manager::unregister &>/dev/null; then
            agent_manager::unregister "${LANGCHAIN_CURRENT_AGENT_ID}" >/dev/null 2>&1
        fi
        exit 0
    }
    
    # Register cleanup for common signals
    trap 'langchain::agent_cleanup' EXIT SIGTERM SIGINT
}

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="langchain::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="langchain::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="langchain::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="langchain::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="langchain::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="langchain::test"

# Content handlers for LangChain-specific functionality
CLI_COMMAND_HANDLERS["content::add"]="langchain::content::add"
CLI_COMMAND_HANDLERS["content::list"]="langchain::content::list" 
CLI_COMMAND_HANDLERS["content::get"]="langchain::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="langchain::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="langchain::content::execute"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed resource status" "langchain::status"
cli::register_command "logs" "Show LangChain logs" "langchain::logs"
# Create wrapper for agents command that delegates to manager
langchain::agents::command() {
    if type -t agent_manager::load_config &>/dev/null; then
        "${APP_ROOT}/scripts/resources/agents/agent-manager.sh" --config="langchain" "$@"
    else
        log::error "Agent management not available"
        return 1
    fi
}
export -f langchain::agents::command

cli::register_command "agents" "Manage running langchain agents" "langchain::agents::command"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi