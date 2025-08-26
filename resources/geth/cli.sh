#!/usr/bin/env bash
################################################################################
# Geth Resource CLI - v2.0 Universal Contract Compliant
# 
# Ethereum blockchain client for running nodes and smart contracts
#
# Usage:
#   resource-geth <command> [options]
#   resource-geth <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    GETH_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${GETH_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
GETH_CLI_DIR="${APP_ROOT}/resources/geth"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${GETH_CLI_DIR}/config/defaults.sh"

# Source Geth libraries
for lib in common core docker install install_wrapper status inject content test; do
    lib_file="${GETH_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "geth" "Ethereum blockchain client management" "v2"

# Override default handlers to point directly to geth implementations
CLI_COMMAND_HANDLERS["manage::install"]="geth::install::execute"
CLI_COMMAND_HANDLERS["manage::uninstall"]="geth::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="geth::docker::start"
CLI_COMMAND_HANDLERS["manage::stop"]="geth::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="geth::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="geth::test::smoke"

# Override content handlers for Geth-specific blockchain functionality
CLI_COMMAND_HANDLERS["content::add"]="geth::content::add"
CLI_COMMAND_HANDLERS["content::list"]="geth::content::list"
CLI_COMMAND_HANDLERS["content::get"]="geth::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="geth::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="geth::content::execute"

# Add Geth-specific content subcommands
cli::register_subcommand "content" "console" "Open Geth interactive console" "geth::content::console"
cli::register_subcommand "content" "block" "Show current block number" "geth::content::block"
cli::register_subcommand "content" "peers" "Show connected peers" "geth::content::peers"
cli::register_subcommand "content" "sync" "Show sync status" "geth::content::sync"

# Additional test phases
cli::register_subcommand "test" "integration" "Run integration tests" "geth::test::integration"
cli::register_subcommand "test" "performance" "Run performance tests" "geth::test::performance"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "geth::status"
cli::register_command "logs" "Show Geth logs" "geth::docker::logs"
cli::register_command "credentials" "Show Geth connection info" "geth::core::credentials"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi