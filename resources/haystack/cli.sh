#!/usr/bin/env bash
################################################################################
# Haystack Resource CLI - v2.0 Universal Contract Compliant
# 
# Open-source AI framework for building production-ready RAG applications
#
# Usage:
#   resource-haystack <command> [options]
#   resource-haystack <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    HAYSTACK_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${HAYSTACK_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
HAYSTACK_CLI_DIR="${APP_ROOT}/resources/haystack"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Source Haystack libraries
for lib in lifecycle status install inject common; do
    lib_file="${HAYSTACK_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "haystack" "Haystack AI framework for RAG applications" "v2"

# Override default handlers to point directly to haystack implementations
CLI_COMMAND_HANDLERS["manage::install"]="haystack::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="haystack::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="haystack::start"
CLI_COMMAND_HANDLERS["manage::stop"]="haystack::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="haystack::restart"

# Test handlers (health checks only)
CLI_COMMAND_HANDLERS["test::smoke"]="haystack::test_smoke"

# Content handlers for Haystack-specific RAG functionality
CLI_COMMAND_HANDLERS["content::add"]="haystack::inject"
CLI_COMMAND_HANDLERS["content::list"]="haystack::list_injected"
CLI_COMMAND_HANDLERS["content::remove"]="haystack::clear_data"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "haystack::status"
cli::register_command "logs" "Show Haystack logs" "haystack::show_logs"

# Custom test wrapper for smoke test
haystack::test_smoke() {
    log::info "Running Haystack smoke test..."
    if haystack::is_running; then
        log::success "Haystack is running and healthy"
        return 0
    else
        log::error "Haystack is not running"
        return 1
    fi
}

# Wrapper for logs command
haystack::show_logs() {
    local log_file="${HAYSTACK_LOG_FILE:-${var_ROOT_DIR}/logs/haystack/haystack.log}"
    if [[ -f "$log_file" ]]; then
        tail -f "$log_file"
    else
        log::warning "Log file not found: $log_file"
        return 1
    fi
}

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi