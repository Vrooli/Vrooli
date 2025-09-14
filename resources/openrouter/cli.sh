#!/usr/bin/env bash
################################################################################
# OpenRouter Resource CLI - v2.0 Universal Contract Compliant
# 
# Unified API to many AI model providers with intelligent routing
#
# Usage:
#   resource-openrouter <command> [options]
#   resource-openrouter <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    OPENROUTER_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${OPENROUTER_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
OPENROUTER_CLI_DIR="${APP_ROOT}/resources/openrouter"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${OPENROUTER_CLI_DIR}/config/defaults.sh"

# Source OpenRouter libraries
for lib in core status install configure content test info models usage ratelimit benchmark agents; do
    lib_file="${OPENROUTER_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "openrouter" "OpenRouter unified API to many AI model providers" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="openrouter::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="openrouter::uninstall"

# OpenRouter is an API service - no docker containers to manage
CLI_COMMAND_HANDLERS["manage::start"]="openrouter::service::noop_start"
CLI_COMMAND_HANDLERS["manage::stop"]="openrouter::service::noop_stop"
CLI_COMMAND_HANDLERS["manage::restart"]="openrouter::service::noop_restart"

# Test handlers - delegate to test library
CLI_COMMAND_HANDLERS["test::smoke"]="openrouter::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="openrouter::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="openrouter::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="openrouter::test::all"

# Content handlers
CLI_COMMAND_HANDLERS["content::add"]="openrouter::content::add"
CLI_COMMAND_HANDLERS["content::list"]="openrouter::content::list"
CLI_COMMAND_HANDLERS["content::get"]="openrouter::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="openrouter::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="openrouter::content::execute"

# Usage tracking handlers
CLI_COMMAND_HANDLERS["usage::today"]="openrouter::usage::today"
CLI_COMMAND_HANDLERS["usage::week"]="openrouter::usage::week"
CLI_COMMAND_HANDLERS["usage::month"]="openrouter::usage::month"
CLI_COMMAND_HANDLERS["usage::all"]="openrouter::usage::all"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "info" "Show structured resource information" "openrouter::info"
cli::register_command "usage" "Show usage analytics and costs" "openrouter::usage"
cli::register_command "status" "Show detailed resource status" "openrouter::status"

# OpenRouter is API service - no logs to show
cli::register_command "logs" "Show OpenRouter logs (API service - no logs)" "openrouter::service::noop_logs"

# ==============================================================================
# OPTIONAL RESOURCE-SPECIFIC COMMANDS
# ==============================================================================
# Add custom top-level commands for OpenRouter functionality
cli::register_command "configure" "Configure OpenRouter API key" "openrouter::configure"
cli::register_command "show-config" "Show current configuration" "openrouter::show_config"

# Add custom content subcommands for OpenRouter-specific operations
cli::register_subcommand "content" "models" "List available models" "openrouter::list_models"
cli::register_subcommand "content" "usage" "Show usage and credits" "openrouter::get_usage"

# Add model management commands
cli::register_command "test-model" "Test a specific model" "openrouter::models::test"
cli::register_command "list-models" "List models by category" "openrouter::models::list_by_category"

# Add benchmark commands for performance testing
cli::register_command "benchmark" "Run model performance benchmarks" "openrouter::benchmark::main"

# Add agent management commands
cli::register_command "agents" "Manage running openrouter agents" "openrouter::agents::command"

# API service helper functions for manage group (compact)
openrouter::service::noop_start() { echo "OpenRouter is an API service (no start needed)"; }
openrouter::service::noop_stop() { echo "OpenRouter is an API service (cannot be stopped)"; }
openrouter::service::noop_restart() { echo "OpenRouter is an API service (no restart needed)"; }
openrouter::service::noop_logs() { echo "OpenRouter is an API service (no logs available)"; echo "Check status with: resource-openrouter status"; }

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi