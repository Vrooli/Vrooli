#!/usr/bin/env bash
################################################################################
# AutoGPT Resource CLI - v2.0 Universal Contract Compliant
# Autonomous AI agent framework for task automation
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    AUTOGPT_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${AUTOGPT_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
AUTOGPT_CLI_DIR="${APP_ROOT}/resources/autogpt"

# Source required components
source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Source agent management (load config and manager directly)
if [[ -f "${APP_ROOT}/resources/autogpt/config/agents.conf" ]]; then
    source "${APP_ROOT}/resources/autogpt/config/agents.conf"
    source "${APP_ROOT}/scripts/resources/agents/agent-manager.sh"
fi

# Source configuration
[[ -f "${AUTOGPT_CLI_DIR}/config/defaults.sh" ]] && source "${AUTOGPT_CLI_DIR}/config/defaults.sh"

# Source AutoGPT libraries
for lib in common core test install start stop status inject agents; do
    [[ -f "${AUTOGPT_CLI_DIR}/lib/${lib}.sh" ]] && source "${AUTOGPT_CLI_DIR}/lib/${lib}.sh"
done

# Initialize CLI framework in v2.0 mode
cli::init "autogpt" "Autonomous AI agent framework for task automation" "v2"

# Required manage handlers - use core functions
CLI_COMMAND_HANDLERS["manage::install"]="autogpt::core::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="autogpt::core::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="autogpt::core::start"
CLI_COMMAND_HANDLERS["manage::stop"]="autogpt::core::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="autogpt::core::restart"

# Test handlers - v2.0 compliant
CLI_COMMAND_HANDLERS["test::smoke"]="autogpt::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="autogpt::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="autogpt::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="autogpt::test::all"

# Content handlers - use core functions
CLI_COMMAND_HANDLERS["content::add"]="autogpt::core::content_add"
CLI_COMMAND_HANDLERS["content::list"]="autogpt::core::content_list"
CLI_COMMAND_HANDLERS["content::get"]="autogpt::core::content_get"
CLI_COMMAND_HANDLERS["content::remove"]="autogpt::core::content_remove"
CLI_COMMAND_HANDLERS["content::execute"]="autogpt::core::content_execute"

# Required information commands
cli::register_command "status" "Show detailed resource status" "autogpt::core::status"
cli::register_command "info" "Show runtime configuration" "cli::framework::info"
# Create wrapper for agents command that delegates to manager
autogpt::agents::command() {
    if type -t agent_manager::load_config &>/dev/null; then
        "${APP_ROOT}/scripts/resources/agents/agent-manager.sh" --config="autogpt" "$@"
    else
        log::error "Agent management not available"
        return 1
    fi
}
export -f autogpt::agents::command

cli::register_command "agents" "Manage running autogpt agents" "autogpt::agents::command"

# Logs handler
cli::register_command "logs" "Show AutoGPT logs" "autogpt::core::logs"

# AutoGPT-specific agent management
cli::register_subcommand "content" "create-agent" "Create a new AI agent" "autogpt::core::content_add" "modifies-system"

# Dispatch if run directly
[[ "${BASH_SOURCE[0]}" == "${0}" ]] && cli::dispatch "$@"