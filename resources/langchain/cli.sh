#!/bin/bash
set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    LANGCHAIN_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    LANGCHAIN_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
LANGCHAIN_CLI_DIR="$(cd "$(dirname "$LANGCHAIN_CLI_SCRIPT")" && pwd)"

# Source dependencies
source "${LANGCHAIN_CLI_DIR}/../../../lib/utils/var.sh"
source "${LANGCHAIN_CLI_DIR}/../../../lib/utils/format.sh"
source "${LANGCHAIN_CLI_DIR}/../../lib/cli-command-framework.sh"
source "${LANGCHAIN_CLI_DIR}/lib/core.sh"
source "${LANGCHAIN_CLI_DIR}/lib/status.sh"
source "${LANGCHAIN_CLI_DIR}/lib/install.sh"
source "${LANGCHAIN_CLI_DIR}/lib/inject.sh"

# Register commands
cli::register_command "status" "Check LangChain status" "langchain::status"
cli::register_command "install" "Install LangChain framework" "langchain::install"
cli::register_command "start" "Start LangChain services" "langchain::start"
cli::register_command "stop" "Stop LangChain services" "langchain::stop"
cli::register_command "inject" "Inject workflow or chain" "langchain::inject"
cli::register_command "test" "Test LangChain connectivity" "langchain::test"
cli::register_command "help" "Show this help message" "cli::_handle_help"

# Main dispatcher
cli::dispatch "$@"