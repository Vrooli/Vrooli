#!/usr/bin/env bash
################################################################################
# Unstructured.io Resource CLI
# 
# Lightweight CLI interface for Unstructured.io using the CLI Command Framework
#
# Usage:
#   resource-unstructured-io <command> [options]
#
################################################################################

set -euo pipefail

# Get script directory
UNSTRUCTURED_IO_CLI_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
VROOLI_ROOT="${VROOLI_ROOT:-$(cd "$UNSTRUCTURED_IO_CLI_DIR/../../../.." && pwd)}"
export VROOLI_ROOT
export RESOURCE_DIR="$UNSTRUCTURED_IO_CLI_DIR"

# Source utilities first
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/lib/utils/var.sh" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_LOG_FILE:-${VROOLI_ROOT}/scripts/lib/utils/log.sh}" 2>/dev/null || true
# shellcheck disable=SC1091
source "${var_RESOURCES_COMMON_FILE:-${VROOLI_ROOT}/scripts/resources/common.sh}" 2>/dev/null || true

# Source the CLI Command Framework
# shellcheck disable=SC1091
source "${VROOLI_ROOT}/scripts/resources/lib/cli-command-framework.sh"

# Source unstructured-io configuration
# shellcheck disable=SC1091
source "${UNSTRUCTURED_IO_CLI_DIR}/config/defaults.sh" 2>/dev/null || true

# Set defaults if not already set
UNSTRUCTURED_IO_CONTAINER_NAME="${UNSTRUCTURED_IO_CONTAINER_NAME:-unstructured-io}"
UNSTRUCTURED_IO_PORT="${UNSTRUCTURED_IO_PORT:-8000}"
UNSTRUCTURED_IO_DEFAULT_STRATEGY="${UNSTRUCTURED_IO_DEFAULT_STRATEGY:-hi_res}"

# Initialize CLI framework
cli::init "unstructured-io" "Unstructured.io document processing and analysis"

# Override help to provide Unstructured.io-specific examples
cli::register_command "help" "Show this help message with Unstructured.io examples" "resource_cli::show_help"

# Register additional Unstructured.io-specific commands
cli::register_command "inject" "Process document using Unstructured.io" "resource_cli::inject" "modifies-system"
cli::register_command "process" "Process document" "resource_cli::process" "modifies-system"
cli::register_command "strategies" "List processing strategies" "resource_cli::strategies"
cli::register_command "formats" "List supported file formats" "resource_cli::formats"
cli::register_command "credentials" "Show n8n credentials for Unstructured.io" "resource_cli::credentials"
cli::register_command "uninstall" "Uninstall Unstructured.io (requires --force)" "resource_cli::uninstall" "modifies-system"

################################################################################
# Resource-specific command implementations
################################################################################

# Process document using Unstructured.io (inject alias)
resource_cli::inject() {
    local file="${1:-}"
    
    if [[ -z "$file" ]]; then
        log::error "Document file path required for processing"
        echo "Usage: resource-unstructured-io inject <document-file>"
        echo ""
        echo "Examples:"
        echo "  resource-unstructured-io inject document.pdf"
        echo "  resource-unstructured-io inject shared:documents/report.docx"
        return 1
    fi
    
    # Handle shared: prefix
    if [[ "$file" == shared:* ]]; then
        file="${VROOLI_ROOT}/${file#shared:}"
    fi
    
    if [[ ! -f "$file" ]]; then
        log::error "Document file not found: $file"
        return 1
    fi
    
    # Delegate to manage.sh
    "${UNSTRUCTURED_IO_CLI_DIR}/manage.sh" --action inject --file "$file" "${@:2}"
}

# Process document
resource_cli::process() {
    resource_cli::inject "$@"
}

# Validate Unstructured.io configuration
resource_cli::validate() {
    "${UNSTRUCTURED_IO_CLI_DIR}/manage.sh" --action status
}

# Show Unstructured.io status
resource_cli::status() {
    "${UNSTRUCTURED_IO_CLI_DIR}/manage.sh" --action status
}

# Start Unstructured.io
resource_cli::start() {
    "${UNSTRUCTURED_IO_CLI_DIR}/manage.sh" --action start
}

# Stop Unstructured.io
resource_cli::stop() {
    "${UNSTRUCTURED_IO_CLI_DIR}/manage.sh" --action stop
}

# Install Unstructured.io
resource_cli::install() {
    "${UNSTRUCTURED_IO_CLI_DIR}/manage.sh" --action install --yes yes
}

# Uninstall Unstructured.io
resource_cli::uninstall() {
    FORCE="${FORCE:-false}"
    
    if [[ "$FORCE" != "true" ]]; then
        echo "⚠️  This will remove Unstructured.io and all its data. Use --force to confirm."
        return 1
    fi
    
    "${UNSTRUCTURED_IO_CLI_DIR}/manage.sh" --action uninstall --yes yes
}

# Get credentials for n8n integration
resource_cli::credentials() {
    source "${VROOLI_ROOT}/scripts/resources/lib/credentials-utils.sh"
    
    if ! credentials::parse_args "$@"; then
        [[ $? -eq 2 ]] && { credentials::show_help "unstructured-io"; return 0; }
        return 1
    fi
    
    local status
    status=$(credentials::get_resource_status "$UNSTRUCTURED_IO_CONTAINER_NAME")
    
    local connections_array="[]"
    if [[ "$status" == "running" ]]; then
        # Unstructured.io HTTP API connection (no authentication required)
        local connection_obj
        connection_obj=$(jq -n \
            --arg host "localhost" \
            --argjson port "$UNSTRUCTURED_IO_PORT" \
            --arg path "/general/v0/general" \
            --argjson ssl false \
            '{
                host: $host,
                port: $port,
                path: $path,
                ssl: $ssl
            }')
        
        local metadata_obj
        metadata_obj=$(jq -n \
            --arg description "Unstructured.io document processing and analysis service" \
            --arg strategy "$UNSTRUCTURED_IO_DEFAULT_STRATEGY" \
            --argjson supported_formats '["pdf", "docx", "doc", "txt", "html", "xml", "pptx", "xlsx", "jpg", "png"]' \
            '{
                description: $description,
                default_strategy: $strategy,
                supported_formats: $supported_formats
            }')
        
        local connection
        connection=$(credentials::build_connection \
            "main" \
            "Unstructured.io API" \
            "httpRequest" \
            "$connection_obj" \
            "{}" \
            "$metadata_obj")
        
        connections_array="[$connection]"
    fi
    
    local response
    response=$(credentials::build_response "unstructured-io" "$status" "$connections_array")
    credentials::format_output "$response"
}

# List processing strategies
resource_cli::strategies() {
    echo "Available Unstructured.io processing strategies:"
    echo ""
    echo "  auto        Automatically choose the best strategy"
    echo "  fast        Fast processing with basic layout detection"
    echo "  hi_res      High-resolution processing with advanced layout"
    echo "  ocr_only    OCR-based text extraction only"
    echo ""
    echo "Strategy descriptions:"
    echo "• auto:     Chooses fast or hi_res based on document complexity"
    echo "• fast:     Faster processing, good for simple documents"
    echo "• hi_res:   More accurate, better for complex layouts and tables"
    echo "• ocr_only: Text extraction without layout analysis"
    echo ""
    echo "Current default: $UNSTRUCTURED_IO_DEFAULT_STRATEGY"
}

# List supported file formats
resource_cli::formats() {
    echo "Supported file formats:"
    echo ""
    echo "Documents:"
    echo "  • PDF (.pdf)          - Portable Document Format"
    echo "  • Word (.docx, .doc)  - Microsoft Word documents"
    echo "  • Text (.txt)         - Plain text files"
    echo "  • HTML (.html)        - Web pages"
    echo "  • XML (.xml)          - Structured markup"
    echo ""
    echo "Presentations:"
    echo "  • PowerPoint (.pptx)  - Microsoft PowerPoint"
    echo ""
    echo "Spreadsheets:"
    echo "  • Excel (.xlsx)       - Microsoft Excel"
    echo ""
    echo "Images:"
    echo "  • JPEG (.jpg, .jpeg)  - Images with text (via OCR)"
    echo "  • PNG (.png)          - Images with text (via OCR)"
    echo ""
    echo "Processing features:"
    echo "  • Text extraction and chunking"
    echo "  • Table detection and extraction"
    echo "  • Layout analysis and element classification"
    echo "  • OCR for scanned documents and images"
}

# Custom help function with Unstructured.io-specific examples
resource_cli::show_help() {
    # Show standard framework help first
    cli::_handle_help
    
    # Add Unstructured.io-specific examples
    echo ""
    echo "📄 Unstructured.io Document Processing Examples:"
    echo ""
    echo "Document Processing:"
    echo "  resource-unstructured-io process document.pdf          # Process PDF document"
    echo "  resource-unstructured-io inject report.docx            # Same as process"
    echo "  resource-unstructured-io inject shared:docs/data.xlsx  # Process shared file"
    echo ""
    echo "Information & Configuration:"
    echo "  resource-unstructured-io strategies                    # List processing strategies"
    echo "  resource-unstructured-io formats                       # List supported formats"
    echo ""
    echo "Management:"
    echo "  resource-unstructured-io status                        # Check service status"
    echo "  resource-unstructured-io credentials                   # Get API details"
    echo ""
    echo "Document Processing Features:"
    echo "  • Advanced text extraction and chunking"
    echo "  • Table detection and structured extraction"
    echo "  • Layout analysis with element classification"
    echo "  • OCR support for scanned documents and images"
    echo ""
    echo "Default Port: $UNSTRUCTURED_IO_PORT"
    echo "Default Strategy: $UNSTRUCTURED_IO_DEFAULT_STRATEGY"
    echo "API Endpoint: http://localhost:$UNSTRUCTURED_IO_PORT/general/v0/general"
}

################################################################################
# Main execution - dispatch to framework
################################################################################

# Only execute if script is run directly (not sourced)
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    cli::dispatch "$@"
fi