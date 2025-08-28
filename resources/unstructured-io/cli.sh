#!/usr/bin/env bash
################################################################################
# Unstructured.io Resource CLI - v2.0 Universal Contract Compliant
# 
# Document processing and analysis service with OCR capabilities
#
# Usage:
#   resource-unstructured-io <command> [options]
#   resource-unstructured-io <group> <subcommand> [options]
#
################################################################################

set -euo pipefail

APP_ROOT="${APP_ROOT:-$(builtin cd "${BASH_SOURCE[0]%/*}/../.." && builtin pwd)}"
# Handle symlinks for installed CLI
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    UNSTRUCTURED_IO_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
    APP_ROOT="$(builtin cd "${UNSTRUCTURED_IO_CLI_SCRIPT%/*}/../.." && builtin pwd)"
fi
UNSTRUCTURED_IO_CLI_DIR="${APP_ROOT}/resources/unstructured-io"

# Source standard variables
source "${APP_ROOT}/scripts/lib/utils/var.sh"

# Source utilities
source "${var_LOG_FILE}"
source "${var_RESOURCES_COMMON_FILE}"

# Source v2.0 CLI Command Framework
source "${APP_ROOT}/scripts/resources/lib/cli-command-framework-v2.sh"

# Source resource configuration
source "${UNSTRUCTURED_IO_CLI_DIR}/config/defaults.sh"

# Source resource libraries (only what exists)
for lib in common core install status api process cache-simple validate test content; do
    lib_file="${UNSTRUCTURED_IO_CLI_DIR}/lib/${lib}.sh"
    [[ -f "$lib_file" ]] && source "$lib_file" 2>/dev/null || true
done

# Initialize CLI framework in v2.0 mode (auto-creates manage/test/content groups)
cli::init "unstructured-io" "Unstructured.io document processing and analysis" "v2"

# ==============================================================================
# REQUIRED HANDLERS - These MUST be mapped for v2.0 compliance
# ==============================================================================
CLI_COMMAND_HANDLERS["manage::install"]="unstructured_io::install"
CLI_COMMAND_HANDLERS["manage::uninstall"]="unstructured_io::uninstall"
CLI_COMMAND_HANDLERS["manage::start"]="unstructured_io::start"  
CLI_COMMAND_HANDLERS["manage::stop"]="unstructured_io::stop"
CLI_COMMAND_HANDLERS["manage::restart"]="unstructured_io::restart"
CLI_COMMAND_HANDLERS["test::smoke"]="unstructured_io::test::smoke"
CLI_COMMAND_HANDLERS["test::integration"]="unstructured_io::test::integration"
CLI_COMMAND_HANDLERS["test::unit"]="unstructured_io::test::unit"
CLI_COMMAND_HANDLERS["test::all"]="unstructured_io::test::all"

# Content handlers for document processing functionality
CLI_COMMAND_HANDLERS["content::add"]="unstructured_io::content::add"
CLI_COMMAND_HANDLERS["content::list"]="unstructured_io::content::list"
CLI_COMMAND_HANDLERS["content::get"]="unstructured_io::content::get"
CLI_COMMAND_HANDLERS["content::remove"]="unstructured_io::content::remove"
CLI_COMMAND_HANDLERS["content::execute"]="unstructured_io::process_document"

# Add document-specific content subcommands
cli::register_subcommand "content" "process" "Process document" "unstructured_io::content::process" "modifies-system"
cli::register_subcommand "content" "process-directory" "Process directory" "unstructured_io::process_directory" "modifies-system"
cli::register_subcommand "content" "extract-tables" "Extract tables" "unstructured_io::extract_tables"
cli::register_subcommand "content" "extract-metadata" "Extract metadata" "unstructured_io::extract_metadata"
cli::register_subcommand "content" "create-report" "Generate report" "unstructured_io::create_report" "modifies-system"

# Cache management subcommands
cli::register_subcommand "content" "cache-stats" "Show cache statistics" "unstructured_io::cache_stats"
cli::register_subcommand "content" "clear-cache" "Clear processing cache" "unstructured_io::content::clear_cache" "modifies-system"

# ==============================================================================
# REQUIRED INFORMATION COMMANDS
# ==============================================================================
cli::register_command "status" "Show detailed resource status" "unstructured_io::status"
cli::register_command "logs" "Show resource logs" "unstructured_io::logs"

# ==============================================================================
# OPTIONAL RESOURCE-SPECIFIC COMMANDS
# ==============================================================================
cli::register_command "credentials" "Show integration credentials" "unstructured_io::core::credentials"
cli::register_command "validate" "Validate installation" "unstructured_io::validate_installation"


# Only execute if script is run directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi