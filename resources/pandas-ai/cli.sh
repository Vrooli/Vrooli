#!/usr/bin/env bash
################################################################################
# Pandas AI Resource CLI - v2.0 Universal Contract Compliant
# 
# AI-powered data analysis and manipulation platform
#
# Usage:
#   resource-pandas-ai <command> [options]
#   resource-pandas-ai <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    PANDAS_AI_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${PANDAS_AI_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
PANDAS_AI_CLI_DIR="${APP_ROOT}/resources/pandas-ai"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${PANDAS_AI_CLI_DIR}/config/defaults.sh"

# Source pandas-ai libraries
for lib in common status install lifecycle docker content test agents; do
    lib_file="${PANDAS_AI_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "pandas-ai" "AI-powered data analysis and manipulation platform" "v2"

# Override default handlers to point directly to pandas-ai implementations
CLI_COMMAND_HANDLERS["manage::install"]="pandas_ai::install::execute"
CLI_COMMAND_HANDLERS["manage::uninstall"]="pandas_ai::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="pandas_ai::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="pandas_ai::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="pandas_ai::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="pandas_ai::test::smoke"
CLI_COMMAND_HANDLERS["test::all"]="pandas_ai::test::all"
CLI_COMMAND_HANDLERS["test::integration"]="pandas_ai::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="pandas_ai::test::unit"

# Override content handlers for pandas-ai specific data analysis functionality
CLI_COMMAND_HANDLERS["content::add"]="pandas_ai::content::add"
CLI_COMMAND_HANDLERS["content::list"]="pandas_ai::content::list" 
CLI_COMMAND_HANDLERS["content::get"]="pandas_ai::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="pandas_ai::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="pandas_ai::content::execute"

# Add pandas-ai-specific content subcommands
cli::register_subcommand "content" "analyze" "Run data analysis query or script" "pandas_ai::content::execute"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "pandas_ai::status"
cli::register_command "logs" "Show pandas-ai logs" "pandas_ai::docker::logs"

# Agent management commands
cli::register_command "agents" "Manage running pandas-ai agents" "pandas_ai::agents::command"

################################################################################
# Agent cleanup function
################################################################################

#######################################
# Setup agent cleanup on signals
# Arguments:
#   $1 - Agent ID
#######################################
pandas_ai::setup_agent_cleanup() {
    local agent_id="$1"
    
    # Export the agent ID so trap can access it
    export PANDAS_AI_CURRENT_AGENT_ID="$agent_id"
    
    # Cleanup function that uses the exported variable
    pandas_ai::agent_cleanup() {
        if [[ -n "${PANDAS_AI_CURRENT_AGENT_ID:-}" ]] && type -t agents::unregister &>/dev/null; then
            agents::unregister "${PANDAS_AI_CURRENT_AGENT_ID}" >/dev/null 2>&1
        fi
        exit 0
    }
    
    # Register cleanup for common signals
    trap 'pandas_ai::agent_cleanup' EXIT SIGTERM SIGINT
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi