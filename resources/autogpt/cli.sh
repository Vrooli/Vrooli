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

# Source AutoGPT libraries
for lib in common install start stop status inject; do
    [[ -f "${AUTOGPT_CLI_DIR}/lib/${lib}.sh" ]] && source "${AUTOGPT_CLI_DIR}/lib/${lib}.sh"
done

# Initialize CLI framework in v2.0 mode
cli::init "autogpt" "Autonomous AI agent framework for task automation" "v2"

# Required manage handlers
CLI_COMMAND_HANDLERS["manage::install"]="autogpt_install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="autogpt_uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="autogpt_start"
CLI_COMMAND_HANDLERS["manage::stop"]="autogpt_stop"
CLI_COMMAND_HANDLERS["manage::restart"]="cli::framework::restart"

# Test handlers - smoke test checks health
autogpt::test::smoke() {
    autogpt_container_running && curl -s "http://localhost:${AUTOGPT_PORT_API}/health" >/dev/null 2>&1
}
CLI_COMMAND_HANDLERS["test::smoke"]="autogpt::test::smoke"

# Content handlers - agent management functionality
CLI_COMMAND_HANDLERS["content::add"]="autogpt_inject"
CLI_COMMAND_HANDLERS["content::list"]="autogpt_list_agents"
CLI_COMMAND_HANDLERS["content::execute"]="autogpt_run_agent"

# Get agent details
autogpt::content::get() {
    [[ -n "${1:-}" ]] && [[ -f "$AUTOGPT_AGENTS_DIR/${1}.yaml" ]] && cat "$AUTOGPT_AGENTS_DIR/${1}.yaml"
}
CLI_COMMAND_HANDLERS["content::get"]="autogpt::content::get"

# Remove agent
autogpt::content::remove() {
    [[ -n "${1:-}" ]] && rm -f "$AUTOGPT_AGENTS_DIR/${1}.yaml"
}
CLI_COMMAND_HANDLERS["content::remove"]="autogpt::content::remove"

# Required information commands
cli::register_command "status" "Show detailed resource status" "autogpt_status"

# Logs handler
autogpt::logs() {
    autogpt_container_exists && docker logs "${1:---tail=100}" "$AUTOGPT_CONTAINER_NAME"
}
cli::register_command "logs" "Show AutoGPT logs" "autogpt::logs"

# AutoGPT-specific agent management
cli::register_subcommand "content" "create-agent" "Create a new AI agent" "autogpt_create_agent" "modifies-system"

# Dispatch if run directly
[[ "${BASH_SOURCE[0]}" == "${0}" ]] && cli::dispatch "$@"