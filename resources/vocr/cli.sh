#!/usr/bin/env bash
################################################################################
# VOCR Resource CLI - v2.0 Universal Contract Compliant
# 
# Vision OCR and AI-powered screen recognition
#
# Usage:
#   resource-vocr <command> [options]
#   resource-vocr <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    VOCR_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    # Recalculate APP_ROOT from resolved symlink location
    APP_ROOT="$(builtin cd "${VOCR_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
VOCR_CLI_DIR="${APP_ROOT}/resources/vocr"

# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"
# shellcheck disable=SC1091
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"
# shellcheck disable=SC1091
source "${VOCR_CLI_DIR}/config/defaults.sh"

# Source VOCR libraries
for lib in core status install capture test; do
    lib_file="${VOCR_CLI_DIR}/lib/${lib}.sh"
    if [[ -f "$lib_file" ]]; then
        # shellcheck disable=SC1090
        source "$lib_file" 2>/dev/null || true
    fi
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "vocr" "Vision OCR and AI-powered screen recognition" "v2"

# Override default handlers to point directly to vocr implementations
CLI_COMMAND_HANDLERS["manage::install"]="vocr::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="vocr::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="vocr::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="vocr::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="vocr::restart"

# Test handlers - delegate to test library
CLI_COMMAND_HANDLERS["test::smoke"]="vocr::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="vocr::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="vocr::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="vocr::test::all"

# Content handlers for VOCR business functionality
CLI_COMMAND_HANDLERS["content::add"]="vocr::ocr::extract"      # Add OCR scan result
CLI_COMMAND_HANDLERS["content::list"]="vocr::list_models"      # List OCR models
CLI_COMMAND_HANDLERS["content::get"]="vocr::capture::screen"   # Get screen capture
CLI_COMMAND_HANDLERS["content::remove"]="vocr::clear_captures" # Remove captures

# Add VOCR-specific content subcommands for business functionality
cli::register_subcommand "content" "capture" "Capture screen region" "vocr::capture::screen"
cli::register_subcommand "content" "ocr" "Extract text from screen" "vocr::ocr::extract"  
cli::register_subcommand "content" "monitor" "Monitor screen region" "vocr::monitor::region"
cli::register_subcommand "content" "configure" "Configure VOCR settings" "vocr::configure"

# Additional information commands
cli::register_command "status" "Show detailed resource status" "vocr::status"
cli::register_command "logs" "Show VOCR logs" "vocr::logs"
cli::register_command "credentials" "Show VOCR credentials for integration" "vocr::credentials"

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi