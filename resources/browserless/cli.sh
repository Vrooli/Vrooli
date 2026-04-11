#!/usr/bin/env bash
################################################################################
# Browserless Resource CLI - thin compatibility surface
#
# Browserless remains as a shared compatibility resource for consumers that still
# require Browserless-shaped behavior. Standard lifecycle is native; the shell CLI
# only retains a small command surface for screenshots and diagnostics.
#
# Usage:
#   resource-browserless <command> [options]
#   resource-browserless <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

# Determine the script directory
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this script is a symlink, resolve it
    BROWSERLESS_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    SCRIPT_DIR="$(dirname "${BROWSERLESS_CLI_SCRIPT}")"
else
    # Get the directory of the script
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

# Set APP_ROOT relative to the script
APP_ROOT="${APP_ROOT:-$(cd "${SCRIPT_DIR}/../.." && pwd)}"
BROWSERLESS_CLI_DIR="${APP_ROOT}/resources/browserless"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${BROWSERLESS_CLI_DIR}/config/defaults.sh"

# Export configuration for subprocesses
browserless::export_config

# Source browserless libraries
for lib in common core docker install start stop status uninstall test health actions; do
    lib_file="${BROWSERLESS_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "browserless" "Browserless headless Chrome automation service" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="install_browserless"
CLI_COMMAND_HANDLERS["manage::uninstall"]="uninstall_browserless"
CLI_COMMAND_HANDLERS["manage::start"]="start_browserless"
CLI_COMMAND_HANDLERS["manage::stop"]="stop_browserless"
CLI_COMMAND_HANDLERS["manage::restart"]="browserless::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="browserless::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="browserless::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="browserless::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="browserless::test::all"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed browserless status" "status"
cli::register_command "logs" "Show browserless logs" "browserless::logs"

# ==============================================================================
# BROWSERLESS-SPECIFIC COMMANDS - retained compatibility surface
# ==============================================================================

cli::register_command "screenshot" "Take screenshots of URLs" "browserless::screenshot"
cli::register_command "diagnostics" "Collect all diagnostics in one browser session (console, network, performance, HTML)" "browserless::diagnostics"

# Dispatcher functions to preserve all original browserless functionality
browserless::screenshot() { actions::dispatch "screenshot" "$@"; }
browserless::diagnostics() { actions::dispatch "diagnostics" "$@"; }

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi
