#!/usr/bin/env bash

# Unstructured.io Common Utilities
# This file contains argument parsing, usage display, and other common utility functions

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
        --options "install|uninstall|start|stop|restart|status|process|info|logs" \
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
    
    # Parse the arguments
    args::parse "$@"
    
    export ACTION=$(args::get "action")
    export FORCE=$(args::get "force")
    export YES=$(args::get "yes")
    export FILE_INPUT=$(args::get "file")
    export STRATEGY=$(args::get "strategy")
    export OUTPUT=$(args::get "output")
    export LANGUAGES=$(args::get "languages")
    export BATCH=$(args::get "batch")
    export FOLLOW=$(args::get "follow")
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
    echo "  install     Install and start Unstructured.io service (default)"
    echo "  uninstall   Stop and remove Unstructured.io service"
    echo "  start       Start the Unstructured.io container"
    echo "  stop        Stop the Unstructured.io container"
    echo "  restart     Restart the Unstructured.io container"
    echo "  status      Check service status and health"
    echo "  process     Process a document file"
    echo "  info        Display service information and endpoints"
    echo "  logs        View container logs"
    echo
    echo "PROCESSING OPTIONS:"
    echo "  --file PATH          File to process"
    echo "  --strategy STRATEGY  Processing strategy (fast|hi_res|auto)"
    echo "  --output FORMAT      Output format (json|markdown|text|elements)"
    echo "  --languages LANGS    OCR languages (e.g., eng,fra,deu)"
    echo
    echo "EXAMPLES:"
    echo "  # Install Unstructured.io"
    echo "  $0 --action install"
    echo
    echo "  # Process a PDF document"
    echo "  $0 --action process --file document.pdf --output markdown"
    echo
    echo "  # Check service status"
    echo "  $0 --action status"
    echo
    echo "  # View logs"
    echo "  $0 --action logs --follow yes"
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
    extension="${extension,,}"  # Convert to lowercase
    
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