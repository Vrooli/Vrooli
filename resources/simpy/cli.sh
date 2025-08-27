#!/usr/bin/env bash
################################################################################
# SimPy Resource CLI - v2.0 Universal Contract Compliant
# 
# Discrete-event simulation library for Python
#
# Usage:
#   resource-simpy <command> [options]
#   resource-simpy <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    SIMPY_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${SIMPY_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
SIMPY_CLI_DIR="${APP_ROOT}/resources/simpy"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${SIMPY_CLI_DIR}/config/defaults.sh"

# Source SimPy libraries
for lib in core docker install status content test; do
    lib_file="${SIMPY_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "simpy" "SimPy discrete-event simulation platform management" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================

# Management handlers
CLI_COMMAND_HANDLERS["manage::install"]="simpy::install::execute"
CLI_COMMAND_HANDLERS["manage::uninstall"]="simpy::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="simpy::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="simpy::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="simpy::docker::restart"

# Test handlers (focused on resource health, not business functionality)
CLI_COMMAND_HANDLERS["test::smoke"]="simpy::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="simpy::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="simpy::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="simpy::test::all"

# Content handlers (business functionality - simulation management)
CLI_COMMAND_HANDLERS["content::add"]="simpy::content::add"
CLI_COMMAND_HANDLERS["content::list"]="simpy::content::list" 
CLI_COMMAND_HANDLERS["content::get"]="simpy::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="simpy::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="simpy::content::execute"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed resource status" "simpy::status"
cli::register_command "logs" "Show SimPy logs" "simpy::docker::logs"

# ==============================================================================
# OPTIONAL SIMPY-SPECIFIC COMMANDS
# ==============================================================================

# Add SimPy-specific content subcommands for simulation management
cli::register_subcommand "content" "examples" "List example simulations" "simpy::list_examples"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi