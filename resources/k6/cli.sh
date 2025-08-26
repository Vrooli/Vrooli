#!/usr/bin/env bash
################################################################################
# K6 Resource CLI - v2.0 Universal Contract Compliant
# 
# Modern load testing tool with JavaScript scripting
#
# Usage:
#   resource-k6 <command> [options]
#   resource-k6 <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    K6_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${K6_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
K6_CLI_DIR="${APP_ROOT}/resources/k6"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${K6_CLI_DIR}/config/defaults.sh"

# Source K6 libraries
for lib in core docker install status inject content test; do
    lib_file="${K6_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "k6" "K6 load testing platform management" "v2"

# Override default handlers to point directly to k6 implementations
CLI_COMMAND_HANDLERS["manage::install"]="k6::install::execute"
CLI_COMMAND_HANDLERS["manage::uninstall"]="k6::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="k6::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="k6::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="k6::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="k6::test::smoke"

# Override content handlers for K6-specific performance testing functionality
CLI_COMMAND_HANDLERS["content::add"]="k6::content::add"
CLI_COMMAND_HANDLERS["content::list"]="k6::content::list" 
CLI_COMMAND_HANDLERS["content::get"]="k6::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="k6::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="k6::content::execute"

# Add K6-specific content subcommands not in the standard framework
cli::register_subcommand "content" "results" "Show recent performance test results" "k6::content::show_results"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "k6::status"
cli::register_command "logs" "Show K6 logs" "k6::docker::logs"
cli::register_command "credentials" "Show K6 credentials for integration" "k6::core::credentials"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi