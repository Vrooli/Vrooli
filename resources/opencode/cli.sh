#!/usr/bin/env bash
################################################################################
# OpenCode Resource CLI - v2.0 Universal Contract Compliant
#
# Terminal-first AI assistant powered by the OpenCode CLI
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    OPENCODE_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${OPENCODE_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
OPENCODE_CLI_DIR="${APP_ROOT}/resources/opencode"

source "${APP_ROOT}/scripts/lib/utils/var.sh"
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

for lib in common install status docker content test; do
    lib_file="${OPENCODE_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "${lib_file}" ]]; then
        # shellcheck disable=SC1090
        source "${lib_file}" 2>/dev/null || true
    fi
done

cli::init "opencode" "OpenCode AI CLI" "v2"

CLI_COMMAND_HANDLERS["manage::install"]="opencode::install::execute"
CLI_COMMAND_HANDLERS["manage::uninstall"]="opencode::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="opencode::status"
CLI_COMMAND_HANDLERS["manage::stop"]="opencode::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="opencode::docker::restart"

CLI_COMMAND_HANDLERS["test::smoke"]="opencode::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="opencode::test::integration"
CLI_COMMAND_HANDLERS["test::all"]="opencode::test::all"

CLI_COMMAND_HANDLERS["content::add"]="opencode::content::add"
CLI_COMMAND_HANDLERS["content::list"]="opencode::content::list"
CLI_COMMAND_HANDLERS["content::get"]="opencode::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="opencode::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="opencode::content::activate"

opencode::cli::dispatch() {
    if [[ $# -eq 0 ]]; then
        echo "Usage: resource-opencode run <args>"
        echo "Try 'resource-opencode run --help' for CLI options."
        return 1
    fi
    opencode::run_cli "$@"
}

cli::register_command "status" "Show OpenCode status" "opencode::status"
cli::register_command "run" "Execute raw OpenCode CLI commands" "opencode::cli::dispatch"
cli::register_command "logs" "Show log directory" "opencode::docker::logs"

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi
