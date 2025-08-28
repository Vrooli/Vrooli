#!/usr/bin/env bash
################################################################################
# Gemini Resource CLI - v2.0 Universal Contract Compliant
# 
# Google Gemini AI API integration for text generation and chat
#
# Usage:
#   resource-gemini <command> [options]
#   resource-gemini <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    GEMINI_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${GEMINI_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
GEMINI_CLI_DIR="${APP_ROOT}/resources/gemini"

# Source standard variables
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source utilities
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"

# Source v2.0 CLI Command Framework
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Source resource configuration
source "${GEMINI_CLI_DIR}/config/defaults.sh"

# Source resource libraries (only what exists)
for lib in core install status content inject test; do
    lib_file="${GEMINI_CLI_DIR}/lib/${lib}.sh"
    [[ -f "$lib_file" ]] && source "$lib_file" 2>/dev/null || true
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "gemini" "Google Gemini AI API integration" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="gemini::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="gemini::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="gemini::docker::start_noop"  
CLI_COMMAND_HANDLERS["manage::stop"]="gemini::docker::stop_noop"
CLI_COMMAND_HANDLERS["manage::restart"]="gemini::docker::restart_noop"
CLI_COMMAND_HANDLERS["test::smoke"]="gemini::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="gemini::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="gemini::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="gemini::test::all"

# Content handlers - required but can use framework defaults if not applicable
CLI_COMMAND_HANDLERS["content::add"]="gemini::content::add"
CLI_COMMAND_HANDLERS["content::list"]="gemini::content::list"
CLI_COMMAND_HANDLERS["content::get"]="gemini::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="gemini::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="gemini::content::execute"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed resource status" "gemini::status"
cli::register_command "logs" "Show resource logs (N/A for API service)" "gemini::logs_noop"

# ==============================================================================
# OPTIONAL RESOURCE-SPECIFIC COMMANDS
# ==============================================================================
cli::register_command "list-models" "List available Gemini models" "gemini::list_models"
cli::register_command "generate" "Generate content using Gemini" "gemini::generate_cli"

# Add no-op handlers for API service (since Gemini doesn't have Docker containers)
gemini::docker::start_noop() {
    log::info "Gemini is an API service (no start needed)"
    return 0
}

gemini::docker::stop_noop() {
    log::info "Gemini is an API service (cannot be stopped)"
    return 0
}

gemini::docker::restart_noop() {
    log::info "Gemini is an API service (no restart needed)"
    return 0
}

gemini::logs_noop() {
    log::info "Gemini is an API service (no logs available)"
    return 0
}

# CLI wrapper for generate command
gemini::generate_cli() {
    local prompt="${1:-}"
    local model="${2:-}"
    
    if [[ -z "$prompt" ]]; then
        log::error "Usage: resource-gemini generate <prompt> [model]"
        return 1
    fi
    
    gemini::generate "$prompt" "$model"
}

# Only execute if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi