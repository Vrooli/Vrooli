#!/usr/bin/env bash
################################################################################
# AutoGen Studio Resource CLI - v2.0 Universal Contract Compliant
# 
# Multi-agent conversation framework for complex task orchestration
#
# Usage:
#   resource-autogen-studio <command> [options]
#   resource-autogen-studio <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    AUTOGEN_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${AUTOGEN_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
AUTOGEN_CLI_DIR="${APP_ROOT}/resources/autogen-studio"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${AUTOGEN_CLI_DIR}/config/defaults.sh"

# Source AutoGen Studio libraries
for lib in core agents docker install status content test; do
    lib_file="${AUTOGEN_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "autogen-studio" "AutoGen Studio multi-agent conversation framework" "v2"

# Override default handlers to point directly to autogen implementations
CLI_COMMAND_HANDLERS["manage::install"]="autogen::install::execute"
CLI_COMMAND_HANDLERS["manage::uninstall"]="autogen::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="autogen::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="autogen::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="autogen::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="autogen::status::check"

# Override content handlers for AutoGen-specific functionality
CLI_COMMAND_HANDLERS["content::add"]="autogen::content::add"
CLI_COMMAND_HANDLERS["content::list"]="autogen::content::list" 
CLI_COMMAND_HANDLERS["content::get"]="autogen::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="autogen::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="autogen::content::execute"

# AutoGen-specific functionality is handled through standard content commands
# (inject functionality has been moved to content::add for v2.0 compliance)

# Additional information commands
cli::register_command "status" "Show detailed resource status" "autogen::status"
cli::register_command "logs" "Show AutoGen Studio logs" "autogen::docker::logs"
cli::register_command "agents" "Manage running autogen-studio agents" "autogen_studio::agents::command"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi