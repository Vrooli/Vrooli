#!/usr/bin/env bash

# Unstructured.io Common Utilities
# This file contains argument parsing, usage display, and other common utility functions

# Get script directory for relative path resolution
LIB_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)

# shellcheck disable=SC1091
source "${LIB_DIR}/../../../../lib/utils/var.sh"
# shellcheck disable=SC1091
source "${var_LIB_NETWORK_DIR}/ports.sh"

#######################################
# Parse command line arguments
#######################################
unstructured_io::parse_arguments() {
    args::reset
    
    args::register_help
    args::register_yes
    
    args::register \
        --name "action" \
        --flag "a" \
        --desc "Action to perform" \
        --type "value" \
        --options "install|uninstall|start|stop|restart|status|process|info|logs|test|extract-tables|extract-metadata|process-directory|create-report|cache-stats|clear-cache" \
        --default "install"
    
    args::register \
        --name "force" \
        --flag "f" \
        --desc "Force action even if container already exists" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "file" \
        --desc "File to process (for process action)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "strategy" \
        --desc "Processing strategy: fast, hi_res, or auto" \
        --type "value" \
        --options "fast|hi_res|auto" \
        --default "$UNSTRUCTURED_IO_DEFAULT_STRATEGY"
    
    args::register \
        --name "output" \
        --desc "Output format: json, markdown, text, or elements" \
        --type "value" \
        --options "json|markdown|text|elements" \
        --default "json"
    
    args::register \
        --name "languages" \
        --desc "Comma-separated list of OCR languages (e.g., eng,fra,deu)" \
        --type "value" \
        --default "$UNSTRUCTURED_IO_DEFAULT_LANGUAGES"
    
    args::register \
        --name "batch" \
        --desc "Process multiple files in batch mode" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "follow" \
        --desc "Follow logs in real-time (for logs action)" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "quiet" \
        --flag "q" \
        --desc "Suppress status messages (only show output)" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "directory" \
        --desc "Directory to process (for process-directory action)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "recursive" \
        --desc "Process directory recursively" \
        --type "value" \
        --options "yes|no" \
        --default "no"
    
    args::register \
        --name "report-file" \
        --desc "Output file for report (for create-report action)" \
        --type "value" \
        --default ""
    
    args::register \
        --name "chunk-chars" \
        --desc "Maximum characters per chunk (default: 1500)" \
        --type "value" \
        --default "1500"
    
    args::register \
        --name "new-after-chars" \
        --desc "Start new chunk after N characters (default: 500)" \
        --type "value" \
        --default "500"
    
    args::register \
        --name "combine-under-chars" \
        --desc "Combine elements under N characters (default: 0)" \
        --type "value" \
        --default "0"
    
    args::register \
        --name "include-page-breaks" \
        --desc "Include page breaks in output" \
        --type "value" \
        --options "yes|no" \
        --default "yes"
    
    # Parse the arguments
    args::parse "$@"
    
    local ACTION
    ACTION=$(args::get "action")
    local FORCE
    FORCE=$(args::get "force")
    local YES
    YES=$(args::get "yes")
    local FILE_INPUT
    FILE_INPUT=$(args::get "file")
    local STRATEGY
    STRATEGY=$(args::get "strategy")
    local OUTPUT
    OUTPUT=$(args::get "output")
    local LANGUAGES
    LANGUAGES=$(args::get "languages")
    local BATCH
    BATCH=$(args::get "batch")
    local FOLLOW
    FOLLOW=$(args::get "follow")
    local QUIET
    QUIET=$(args::get "quiet")
    local DIRECTORY
    DIRECTORY=$(args::get "directory")
    local RECURSIVE
    RECURSIVE=$(args::get "recursive")
    local REPORT_FILE
    REPORT_FILE=$(args::get "report-file")
    local CHUNK_CHARS
    CHUNK_CHARS=$(args::get "chunk-chars")
    local NEW_AFTER_CHARS
    NEW_AFTER_CHARS=$(args::get "new-after-chars")
    local COMBINE_UNDER_CHARS
    COMBINE_UNDER_CHARS=$(args::get "combine-under-chars")
    local INCLUDE_PAGE_BREAKS
    INCLUDE_PAGE_BREAKS=$(args::get "include-page-breaks")
    export ACTION FORCE YES FILE_INPUT STRATEGY OUTPUT LANGUAGES BATCH FOLLOW QUIET DIRECTORY RECURSIVE REPORT_FILE CHUNK_CHARS NEW_AFTER_CHARS COMBINE_UNDER_CHARS INCLUDE_PAGE_BREAKS
}

#######################################
# Print status message if not in quiet mode
#######################################
unstructured_io::status_msg() {
    if [[ "$QUIET" != "yes" ]]; then
        echo "$@" >&2
    fi
}

#######################################
# Display usage information
#######################################
unstructured_io::usage() {
    echo "Unstructured.io Resource Manager"
    echo
    echo "Manage the Unstructured.io document processing service for Vrooli"
    echo
    echo "USAGE:"
    echo "  $0 [OPTIONS]"
    echo
    echo "ACTIONS:"
    echo "  install           Install and start Unstructured.io service (default)"
    echo "  uninstall         Stop and remove Unstructured.io service"
    echo "  start             Start the Unstructured.io container"
    echo "  stop              Stop the Unstructured.io container"
    echo "  restart           Restart the Unstructured.io container"
    echo "  status            Check service status and health"
    echo "  process           Process a document file"
    echo "  extract-tables    Extract tables from a document"
    echo "  extract-metadata  Extract metadata from a document"
    echo "  process-directory Process all documents in a directory"
    echo "  create-report     Generate a detailed processing report"
    echo "  info              Display service information and endpoints"
    echo "  logs              View container logs"
    echo "  test              Test API functionality with a sample document"
    echo "  cache-stats       Show cache statistics"
    echo "  clear-cache       Clear document cache"
    echo
    echo "PROCESSING OPTIONS:"
    echo "  --file PATH          File to process"
    echo "  --strategy STRATEGY  Processing strategy (fast|hi_res|auto)"
    echo "  --output FORMAT      Output format (json|markdown|text|elements)"
    echo "  --languages LANGS    OCR languages (e.g., eng,fra,deu)"
    echo "  --quiet, -q          Suppress status messages (only show output)"
    echo
    echo "CHUNKING OPTIONS:"
    echo "  --chunk-chars N      Maximum characters per chunk (default: 1500)"
    echo "  --new-after-chars N  Start new chunk after N chars (default: 500)"
    echo "  --combine-under-chars N  Combine elements under N chars (default: 0)"
    echo "  --include-page-breaks    Include page breaks (yes|no, default: yes)"
    echo
    echo "DIRECTORY OPTIONS:"
    echo "  --directory PATH     Directory to process"
    echo "  --recursive          Process subdirectories (yes|no)"
    echo
    echo "REPORT OPTIONS:"
    echo "  --report-file PATH   Output file for report (optional)"
    echo
    echo "EXAMPLES:"
    echo "  # Install Unstructured.io"
    echo "  $0 --action install"
    echo
    echo "  # Process a PDF document"
    echo "  $0 --action process --file document.pdf --output markdown"
    echo
    echo "  # Extract tables from a document"
    echo "  $0 --action extract-tables --file report.pdf"
    echo
    echo "  # Extract metadata from a document"
    echo "  $0 --action extract-metadata --file document.docx"
    echo
    echo "  # Process all documents in a directory"
    echo "  $0 --action process-directory --directory /path/to/docs --recursive yes"
    echo
    echo "  # Create a processing report"
    echo "  $0 --action create-report --file document.pdf --report-file analysis.json"
    echo
    echo "  # Check service status"
    echo "  $0 --action status"
    echo
    echo "  # View logs"
    echo "  $0 --action logs --follow yes"
    echo
    echo "  # Process quietly for automation"
    echo "  $0 --action process --file document.pdf --quiet yes --output json"
}

#######################################
# Check if Docker is available
#######################################
unstructured_io::check_docker() {
    if ! command -v docker &> /dev/null; then
        echo "$MSG_DOCKER_REQUIRED"
        return 1
    fi
    
    if ! docker info &> /dev/null; then
        echo "$MSG_ERROR_DOCKER_NOT_RUNNING"
        return 1
    fi
    
    return 0
}

#######################################
# Check if container exists
#######################################
unstructured_io::container_exists() {
    docker ps -a --format '{{.Names}}' | grep -q "^${UNSTRUCTURED_IO_CONTAINER_NAME}$"
}

#######################################
# Check if container is running
#######################################
unstructured_io::container_running() {
    docker ps --format '{{.Names}}' | grep -q "^${UNSTRUCTURED_IO_CONTAINER_NAME}$"
}

#######################################
# Wait for API to be ready
#######################################
unstructured_io::wait_for_api() {
    local max_attempts=30
    local attempt=1
    
    echo "$MSG_UNSTRUCTURED_IO_STARTING"
    
    while [ $attempt -le $max_attempts ]; do
        if curl -s -f "${UNSTRUCTURED_IO_BASE_URL}${UNSTRUCTURED_IO_HEALTH_ENDPOINT}" > /dev/null 2>&1; then
            return 0
        fi
        
        sleep 2
        ((attempt++))
    done
    
    return 1
}

#######################################
# Validate file for processing
#######################################
unstructured_io::validate_file() {
    local file="$1"
    
    # Check if file exists
    if [ ! -f "$file" ]; then
        echo "❌ File not found: $file"
        return 1
    fi
    
    # Check file size
    local file_size=$(stat -f%z "$file" 2>/dev/null || stat -c%s "$file" 2>/dev/null)
    if [ "$file_size" -gt "$UNSTRUCTURED_IO_MAX_FILE_SIZE_BYTES" ]; then
        echo "$MSG_FILE_TOO_LARGE"
        return 1
    fi
    
    # Check file extension
    local extension="${file##*.}"
    extension=$(echo "$extension" | tr '[:upper:]' '[:lower:]')  # Convert to lowercase
    
    local supported=false
    for fmt in "${UNSTRUCTURED_IO_SUPPORTED_FORMATS[@]}"; do
        if [ "$extension" = "$fmt" ]; then
            supported=true
            break
        fi
    done
    
    if [ "$supported" = false ]; then
        format="$extension"
        eval "echo \"$MSG_UNSUPPORTED_FORMAT\""
        return 1
    fi
    
    return 0
}

#######################################
# Format file size for display
#######################################
unstructured_io::format_size() {
    local bytes="$1"
    
    if [ "$bytes" -lt 1024 ]; then
        echo "${bytes}B"
    elif [ "$bytes" -lt $((1024 * 1024)) ]; then
        echo "$((bytes / 1024))KB"
    elif [ "$bytes" -lt $((1024 * 1024 * 1024)) ]; then
        echo "$((bytes / 1024 / 1024))MB"
    else
        echo "$((bytes / 1024 / 1024 / 1024))GB"
    fi
}

#######################################
# Display container logs
#######################################
unstructured_io::logs() {
    local follow="${1:-no}"
    local container_name="${UNSTRUCTURED_IO_CONTAINER_NAME:-unstructured-io}"
    
    if [[ "$follow" == "yes" ]]; then
        docker logs -f "$container_name"
    else
        docker logs "$container_name"
    fi
}

#######################################
# Display service information
#######################################
unstructured_io::info() {
    echo "Service: Unstructured.io Document Processing"
    echo "Container: ${UNSTRUCTURED_IO_CONTAINER_NAME:-unstructured-io}"
    echo "Base URL: ${UNSTRUCTURED_IO_BASE_URL:-http://localhost:11450}"
    echo "Port: ${UNSTRUCTURED_IO_PORT:-11450}"
}

#######################################
# Check if port is available
#######################################
unstructured_io::check_port_available() {
    local port="${UNSTRUCTURED_IO_PORT:-11450}"
    ! ports::is_port_in_use "$port"
}

#######################################
# Validate processing strategy
#######################################
unstructured_io::validate_strategy() {
    local strategy="$1"
    case "$strategy" in
        "hi_res"|"fast"|"auto")
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

#######################################
# Validate output format
#######################################
unstructured_io::validate_output_format() {
    local format="$1"
    case "$format" in
        "json"|"markdown"|"text"|"elements")
            return 0
            ;;
        *)
            return 1
            ;;
    esac
}

# Export functions for subshell availability
export -f unstructured_io::parse_arguments
export -f unstructured_io::usage
export -f unstructured_io::status_msg
export -f unstructured_io::check_docker
export -f unstructured_io::container_exists
export -f unstructured_io::container_running
export -f unstructured_io::wait_for_api
export -f unstructured_io::validate_file
export -f unstructured_io::format_size
export -f unstructured_io::logs
export -f unstructured_io::info
export -f unstructured_io::check_port_available
export -f unstructured_io::validate_strategy
export -f unstructured_io::validate_output_format