#!/usr/bin/env bash
################################################################################
# Judge0 Resource CLI - v2.0 Universal Contract Compliant
# 
# Secure code execution service for running untrusted code safely
#
# Usage:
#   resource-judge0 <command> [options]
#   resource-judge0 <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    JUDGE0_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${JUDGE0_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
JUDGE0_CLI_DIR="${APP_ROOT}/resources/judge0"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/docker-utils.sh"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${JUDGE0_CLI_DIR}/config/defaults.sh"
# shellcheck disable=SC1091
[[ -f "${JUDGE0_CLI_DIR}/config/messages.sh" ]] && source "${JUDGE0_CLI_DIR}/config/messages.sh"
judge0::export_config 2>/dev/null || true
judge0::export_messages 2>/dev/null || true

# Source Judge0 libraries
for lib in common docker status install api languages security usage content batch analytics security-dashboard webhooks cache custom-languages; do
    lib_file="${JUDGE0_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "judge0" "Secure code execution service management" "v2"

# Override default handlers to point directly to judge0 implementations
CLI_COMMAND_HANDLERS["manage::install"]="judge0::install::main"
CLI_COMMAND_HANDLERS["manage::uninstall"]="judge0::install::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="judge0::docker::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="judge0::docker::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="judge0::docker::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="judge0::status::is_healthy"

# Override content handlers for Judge0-specific code execution functionality
CLI_COMMAND_HANDLERS["content::add"]="judge0::content::add"
CLI_COMMAND_HANDLERS["content::list"]="judge0::content::list"
CLI_COMMAND_HANDLERS["content::get"]="judge0::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="judge0::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="judge0::content::execute"

# Add Judge0-specific content subcommands
cli::register_subcommand "content" "submit" "Submit code for execution" "judge0::api::submit"
cli::register_subcommand "content" "languages" "List supported languages" "judge0::languages::list"
cli::register_subcommand "content" "usage" "Show usage statistics" "judge0::usage::show"
cli::register_subcommand "content" "batch" "Submit batch of codes" "judge0::batch::submit"
cli::register_subcommand "content" "analytics" "View execution analytics" "judge0::analytics::dashboard"

# Add Judge0-specific test subcommands
cli::register_subcommand "test" "api" "Test API connectivity" "judge0::api::test"
cli::register_subcommand "test" "security" "Run security monitoring" "judge0::security::monitor"
cli::register_subcommand "test" "security-dashboard" "View security dashboard" "judge0::security::dashboard::show"

# Register additional command groups and commands
cli::register_command "cache-stats" "Show cache statistics" "judge0::cache::stats"
cli::register_command "cache-clear" "Clear all cached results" "judge0::cache::clear"
cli::register_command "cache-warm" "Warm cache with common tests" "judge0::cache::warm"
cli::register_command "webhook-list" "List registered webhooks" "judge0::webhooks::list"
cli::register_command "webhook-test" "Test webhook endpoint" "judge0::webhooks::test"
cli::register_command "custom-languages" "List custom languages" "judge0::custom_langs::list"
cli::register_command "add-language-presets" "Add common language presets" "judge0::custom_langs::add_presets"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "judge0::status"
cli::register_command "logs" "Show Judge0 logs" "judge0::docker::logs"
cli::register_command "credentials" "Show Judge0 credentials for integration" "judge0::core::credentials"
cli::register_command "info" "Get system information" "judge0::api::system_info"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi