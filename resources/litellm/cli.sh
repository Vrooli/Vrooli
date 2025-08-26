#!/usr/bin/env bash
################################################################################
# LiteLLM Resource CLI - v2.0 Universal Contract Compliant
# 
# Unified LLM proxy server supporting multiple AI providers
#
# Usage:
#   resource-litellm <command> [options]
#   resource-litellm <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    LITELLM_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${LITELLM_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
LITELLM_CLI_DIR="${APP_ROOT}/resources/litellm"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${LITELLM_CLI_DIR}/config/defaults.sh"

# Source LiteLLM libraries
for lib in core docker install status content test; do
    lib_file="${LITELLM_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "litellm" "LiteLLM unified LLM proxy server management" "v2"

# Override default handlers to point directly to litellm implementations
CLI_COMMAND_HANDLERS["manage::install"]="litellm::install::execute"
CLI_COMMAND_HANDLERS["manage::uninstall"]="litellm::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="litellm::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="litellm::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="litellm::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="litellm::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="litellm::test::integration"
CLI_COMMAND_HANDLERS["test::all"]="litellm::test::all"

# Override content handlers for LiteLLM-specific configuration and model management
CLI_COMMAND_HANDLERS["content::add"]="litellm::content::add"
CLI_COMMAND_HANDLERS["content::list"]="litellm::content::list" 
CLI_COMMAND_HANDLERS["content::get"]="litellm::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="litellm::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="litellm::content::execute"

# Add LiteLLM-specific content subcommands not in the standard framework
cli::register_subcommand "content" "models" "List available AI models" "litellm::list_models"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "litellm::status"
cli::register_command "logs" "Show LiteLLM logs" "litellm::docker::logs"
cli::register_command "credentials" "Show LiteLLM credentials for integration" "litellm::core::credentials"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi