#!/usr/bin/env bash
################################################################################
# Unstructured.io Resource CLI
# 
# Ultra-thin CLI wrapper that delegates directly to library functions
#
# Usage:
#   resource-unstructured-io <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory (resolving symlinks for installed CLI)
if [[ -L "${BASH_SOURCE[0]}" ]]; then
    # If this is a symlink, resolve it
    UNSTRUCTURED_IO_CLI_SCRIPT="$(readlink -f "${BASH_SOURCE[0]}")"
else
    UNSTRUCTURED_IO_CLI_SCRIPT="${BASH_SOURCE[0]}"
fi
UNSTRUCTURED_IO_CLI_DIR="$(cd "$(dirname "$UNSTRUCTURED_IO_CLI_SCRIPT")" && pwd)"

# Source standard variables
# shellcheck disable=SC1091
source "${UNSTRUCTURED_IO_CLI_DIR}/../../../lib/utils/var.sh"

# Source utilities using var_ variables
# shellcheck disable=SC1091
source "${var_LOG_FILE}"
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE}"

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${var_SCRIPTS_RESOURCES_LIB_DIR}/cli-command-framework.sh"

# Source unstructured-io configuration
# shellcheck disable=SC1091
source "${UNSTRUCTURED_IO_CLI_DIR}/config/defaults.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${UNSTRUCTURED_IO_CLI_DIR}/config/messages.sh" 2>/dev/null || true

# Export configuration
unstructured_io::export_config 2>/dev/null || true

# Source all library modules directly - these contain the actual functionality
# shellcheck disable=SC1091
source "${UNSTRUCTURED_IO_CLI_DIR}/lib/common.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${UNSTRUCTURED_IO_CLI_DIR}/lib/install.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${UNSTRUCTURED_IO_CLI_DIR}/lib/status.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${UNSTRUCTURED_IO_CLI_DIR}/lib/api.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${UNSTRUCTURED_IO_CLI_DIR}/lib/process.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${UNSTRUCTURED_IO_CLI_DIR}/lib/cache-simple.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${UNSTRUCTURED_IO_CLI_DIR}/lib/validate.sh" 2>/dev/null || true

# Initialize CLI framework
cli::init "unstructured-io" "Unstructured.io document processing and analysis"

# Override help to provide Unstructured.io-specific examples
cli::register_command "help" "Show this help message with examples" "unstructured_io::show_help"

# Register core commands - direct library function calls
cli::register_command "install" "Install Unstructured.io" "unstructured_io::install" "modifies-system"
cli::register_command "uninstall" "Uninstall Unstructured.io" "unstructured_io::uninstall" "modifies-system"
cli::register_command "start" "Start Unstructured.io" "unstructured_io::start" "modifies-system"
cli::register_command "stop" "Stop Unstructured.io" "unstructured_io::stop" "modifies-system"
cli::register_command "restart" "Restart Unstructured.io" "unstructured_io::restart" "modifies-system"
cli::register_command "status" "Show service status" "unstructured_io::status"
cli::register_command "validate" "Validate installation" "unstructured_io::validate_installation"
cli::register_command "logs" "Show container logs" "unstructured_io::logs"
cli::register_command "info" "Show system information" "unstructured_io::info"

# Register processing commands - direct library function calls
cli::register_command "process" "Process document" "unstructured_io::cli_process" "modifies-system"
cli::register_command "inject" "Process document (alias)" "unstructured_io::cli_process" "modifies-system"
cli::register_command "extract-tables" "Extract tables from document" "unstructured_io::cli_extract_tables"
cli::register_command "extract-metadata" "Extract metadata from document" "unstructured_io::cli_extract_metadata"
cli::register_command "process-directory" "Process directory of documents" "unstructured_io::cli_process_directory" "modifies-system"
cli::register_command "create-report" "Generate document report" "unstructured_io::cli_create_report" "modifies-system"

# Register cache commands
cli::register_command "cache-stats" "Show cache statistics" "unstructured_io::cache_stats"
cli::register_command "clear-cache" "Clear processing cache" "unstructured_io::cli_clear_cache" "modifies-system"

# Register utility commands
cli::register_command "test-api" "Test API connectivity" "unstructured_io::test_api"
cli::register_command "strategies" "List processing strategies" "unstructured_io::show_strategies"
cli::register_command "formats" "List supported formats" "unstructured_io::show_formats"
cli::register_command "credentials" "Get n8n credentials" "unstructured_io::show_credentials"

################################################################################
# CLI wrapper functions - minimal wrappers for commands that need argument handling
################################################################################

# Process document with proper argument handling
unstructured_io::cli_process() {
    local file="${1:-}"
    local strategy="${2:-$UNSTRUCTURED_IO_DEFAULT_STRATEGY}"
    local output="${3:-}"
    local languages="${4:-}"
    
    if [[ -z "$file" ]]; then
        log::error "Document file path required"
        echo "Usage: resource-unstructured-io process <file> [strategy] [output] [languages]"
        return 1
    fi
    
    # Handle shared: prefix
    [[ "$file" == shared:* ]] && file="${var_ROOT_DIR}/${file#shared:}"
    
    unstructured_io::process_document "$file" "$strategy" "$output" "$languages"
}

# Extract tables with file argument
unstructured_io::cli_extract_tables() {
    local file="${1:-}"
    [[ -z "$file" ]] && { log::error "File required"; return 1; }
    [[ "$file" == shared:* ]] && file="${var_ROOT_DIR}/${file#shared:}"
    unstructured_io::extract_tables "$file"
}

# Extract metadata with file argument
unstructured_io::cli_extract_metadata() {
    local file="${1:-}"
    [[ -z "$file" ]] && { log::error "File required"; return 1; }
    [[ "$file" == shared:* ]] && file="${var_ROOT_DIR}/${file#shared:}"
    unstructured_io::extract_metadata "$file"
}

# Process directory with arguments
unstructured_io::cli_process_directory() {
    local directory="${1:-}"
    local strategy="${2:-$UNSTRUCTURED_IO_DEFAULT_STRATEGY}"
    local output="${3:-}"
    local recursive="${4:-no}"
    
    [[ -z "$directory" ]] && { log::error "Directory required"; return 1; }
    unstructured_io::process_directory "$directory" "$strategy" "$output" "$recursive"
}

# Create report with file argument
unstructured_io::cli_create_report() {
    local file="${1:-}"
    local report_file="${2:-}"
    [[ -z "$file" ]] && { log::error "File required"; return 1; }
    [[ "$file" == shared:* ]] && file="${var_ROOT_DIR}/${file#shared:}"
    unstructured_io::create_report "$file" "$report_file"
}

# Clear cache with optional file
unstructured_io::cli_clear_cache() {
    local file="${1:-}"
    if [[ -n "$file" ]]; then
        unstructured_io::clear_cache "$file"
    else
        unstructured_io::clear_all_cache
    fi
}

# Show processing strategies
unstructured_io::show_strategies() {
    echo "Available processing strategies:"
    echo "  • auto     - Automatically choose the best strategy"
    echo "  • fast     - Fast processing with basic layout detection"
    echo "  • hi_res   - High-resolution processing with advanced layout"
    echo "  • ocr_only - OCR-based text extraction only"
    echo ""
    echo "Default: $UNSTRUCTURED_IO_DEFAULT_STRATEGY"
}

# Show supported formats
unstructured_io::show_formats() {
    echo "Supported file formats:"
    echo ""
    echo "Documents: PDF, Word (docx/doc), Text, HTML, XML"
    echo "Presentations: PowerPoint (pptx)"
    echo "Spreadsheets: Excel (xlsx)"
    echo "Images: JPEG, PNG (with OCR)"
}

# Show credentials for n8n integration
unstructured_io::show_credentials() {
    # Source credentials utilities
    # shellcheck disable=SC1091
    source "${var_SCRIPTS_RESOURCES_LIB_DIR}/credentials-utils.sh"
    
    credentials::parse_args "$@" || return $?
    
    local status
    status=$(credentials::get_resource_status "$UNSTRUCTURED_IO_CONTAINER_NAME")
    
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "$UNSTRUCTURED_IO_PORT" \
            --arg path "/general/v0/general" \
            '{host: $host, port: $port, path: $path, ssl: false}')
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "Document processing service" \
            --arg strategy "$UNSTRUCTURED_IO_DEFAULT_STRATEGY" \
            '{description: $description, default_strategy: $strategy}')
        
        local connection
        connection=$(credentials::build_connection \
            "main" "Unstructured.io API" "httpRequest" \
            "$connection_obj" "{}" "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    credentials::format_output "$(credentials::build_response "unstructured-io" "$status" "$connections_array")"
}

# Custom help function with examples
unstructured_io::show_help() {
    cli::_handle_help
    
    echo ""
    echo "📄 Examples:"
    echo ""
    echo "  # Process documents"
    echo "  resource-unstructured-io process document.pdf"
    echo "  resource-unstructured-io process report.docx hi_res output.json"
    echo "  resource-unstructured-io process-directory ./docs/"
    echo ""
    echo "  # Extract data"
    echo "  resource-unstructured-io extract-tables financial.pdf"
    echo "  resource-unstructured-io extract-metadata paper.docx"
    echo ""
    echo "  # Management"
    echo "  resource-unstructured-io status"
    echo "  resource-unstructured-io cache-stats"
    echo "  resource-unstructured-io clear-cache"
    echo ""
    echo "Default Port: $UNSTRUCTURED_IO_PORT"
    echo "API Endpoint: http://localhost:$UNSTRUCTURED_IO_PORT/general/v0/general"
}

################################################################################
# Main execution
################################################################################

if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi